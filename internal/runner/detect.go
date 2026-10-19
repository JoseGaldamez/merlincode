// Package runner detects and supervises application launch profiles. It never
// accepts a command line from a model or from the WebView.
package runner

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"merlincode/internal/domain"
	"merlincode/internal/workspace"
)

type recipe struct {
	domain.RunProposal
	executable string
	args       []string
	static     bool
}

func policy(message string) error { return &domain.RunPolicyError{Message: message} }

// readProject reads only validated regular files through the pinned root.
func readProject(root *os.Root, base, name string) ([]byte, error) {
	if !filepath.IsLocal(name) {
		return nil, domain.ErrAccessDenied
	}
	if _, err := workspace.ValidatePathInProject(base, name); err != nil {
		return nil, err
	}
	f, err := root.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return nil, domain.ErrAccessDenied
	}
	if info.Size() > workspace.MaxFileBytes {
		return nil, domain.ErrFileTooLarge
	}
	b, err := io.ReadAll(io.LimitReader(f, workspace.MaxFileBytes+1))
	if len(b) > workspace.MaxFileBytes {
		return nil, domain.ErrFileTooLarge
	}
	return b, err
}

func detect(root, projectRoot *os.Root, directory, project string, req domain.RunRequest) ([]recipe, error) {
	if req.Directory == "" {
		req.Directory = "."
	}
	req.Directory = filepath.ToSlash(filepath.Clean(req.Directory))
	if req.Entry != "" && (!filepath.IsLocal(req.Entry) || strings.ContainsAny(req.Entry, "\x00\r\n")) {
		return nil, domain.ErrAccessDenied
	}
	// A JavaScript manifest selects its launch scripts, not an executable entry.
	if entry := filepath.ToSlash(req.Entry); entry == "package.json" || entry == "./package.json" {
		req.Entry = ""
	}
	// The fingerprint covers all files consulted, including absent lockfiles and hooks.
	fingerprint := sha256.New()
	read := func(name string) []byte {
		data, err := readProject(root, directory, name)
		fmt.Fprintf(fingerprint, "%s:%t:", name, err == nil)
		fingerprint.Write(data)
		return data
	}
	d := &profileDetector{root: root, request: req, project: project, read: read}
	manifest := read("package.json")
	var jsErr error
	if len(manifest) > 0 && req.Entry == "" {
		ancestorRead := func(name string) []byte {
			data, err := readProject(projectRoot, project, name)
			fmt.Fprintf(fingerprint, "ancestor:%s:%t:", name, err == nil)
			fingerprint.Write(data)
			return data
		}
		jsErr = d.detectJavaScript(manifest, ancestorRead)
	}
	if len(d.recipes) == 0 {
		if err := d.detectPython(); err != nil {
			return nil, err
		}
		if len(d.recipes) == 0 {
			d.detectGo()
		}
		if len(d.recipes) == 0 && len(manifest) == 0 && len(read("index.html")) > 0 {
			d.add("Sitio HTML", "merlin:serve . --host 127.0.0.1 --port auto", "Servidor estático integrado; no requiere instalar dependencias.", "")
			d.recipes[len(d.recipes)-1].static = true
		}
	}
	if len(d.recipes) == 0 {
		if jsErr != nil {
			return nil, jsErr
		}
		return nil, policy("No se encontró una entrada de aplicación compatible. Indica la subcarpeta o el archivo de entrada de la aplicación o script.")
	}
	digest := fingerprint.Sum(nil)
	for i := range d.recipes {
		h := sha256.New()
		h.Write(digest)
		fmt.Fprintf(h, "%s\x00%s\x00%s\x00%s", project, req.Directory, req.Entry, d.recipes[i].Command)
		d.recipes[i].ID = hex.EncodeToString(h.Sum(nil))[:32]
	}
	return d.recipes, nil
}

type profileDetector struct {
	root    *os.Root
	request domain.RunRequest
	project string
	read    func(string) []byte
	recipes []recipe
}

func (d *profileDetector) add(title, command, reason, exe string, args ...string) {
	d.recipes = append(d.recipes, recipe{RunProposal: domain.RunProposal{Project: d.project, Directory: d.request.Directory, Title: title, Command: command, Reason: reason, Request: d.request}, executable: exe, args: args})
}

func boolCount(values ...bool) int {
	n := 0
	for _, v := range values {
		if v {
			n++
		}
	}
	return n
}

func displayCommand(exe string, args []string) string {
	parts := []string{exe}
	for _, arg := range args {
		if strings.ContainsAny(arg, " \t") {
			b, _ := json.Marshal(arg)
			parts = append(parts, string(b))
		} else {
			parts = append(parts, arg)
		}
	}
	return strings.Join(parts, " ")
}
