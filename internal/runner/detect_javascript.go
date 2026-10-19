package runner

import (
	"encoding/json"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var safeToken = regexp.MustCompile(`^[A-Za-z0-9_./:@=,+-]+$`)
var scriptName = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9:_-]*$`)

func (d *profileDetector) detectJavaScript(manifest []byte, ancestorRead func(string) []byte) error {
	read, req := d.read, d.request
	read(".npmrc")
	read(".yarnrc.yml")
	var pkg struct {
		PackageManager string            `json:"packageManager"`
		Scripts        map[string]string `json:"scripts"`
	}
	if json.Unmarshal(manifest, &pkg) != nil {
		return policy("package.json no contiene JSON válido.")
	}
	manager, err := detectManager(req.Directory, pkg.PackageManager, read, ancestorRead)
	if err != nil {
		return err
	}
	for _, script := range launchScriptNames(pkg.Scripts) {
		body := strings.TrimSpace(pkg.Scripts[script])
		if body == "" {
			continue
		}
		if pkg.Scripts["pre"+script] != "" || pkg.Scripts["post"+script] != "" {
			continue
		}
		if !validJSScript(body, read) {
			continue
		}
		args := []string{"run", script}
		// Vue CLI otherwise listens on all interfaces by default. Keep the
		// launch local, including when vue.config.js specifies another host.
		if strings.Fields(body)[0] == "vue-cli-service" {
			if manager == "npm" {
				args = append(args, "--")
			}
			args = append(args, "--host", "127.0.0.1")
		}
		d.add(script+" · "+manager, displayCommand(manager, args), jsLaunchDescription(body)+" Detectado en package.json: "+body, manager, args...)
	}
	if len(d.recipes) == 0 {
		return policy("No hay un script de ejecución compatible en package.json. Se analiza el contenido de los scripts, incluidos los nombres personalizados. Los comandos compuestos y los hooks pre/post no están permitidos.")
	}
	return nil
}

// Describe the validated launcher, not the arbitrary name of its package script.
func jsLaunchDescription(body string) string {
	words := strings.Fields(body)
	if len(words) > 1 {
		if words[1] == "preview" {
			return "Vista previa: sirve el resultado compilado para revisarlo antes de publicarlo. Requiere una compilación previa y no refleja los cambios del código fuente automáticamente."
		}
		if words[1] == "start" && (words[0] == "next" || words[0] == "nuxt" || words[0] == "nuxi") {
			return "Inicio de producción: ejecuta la aplicación ya compilada. Requiere una compilación previa; para trabajar en el código usa el servidor de desarrollo, si está disponible."
		}
	}
	if len(words) > 0 {
		switch words[0] {
		case "node", "bun", "tsx", "ts-node", "nodemon":
			return "Servidor de la aplicación: ejecuta el archivo de entrada. El modo de funcionamiento y la recarga de cambios dependen de su implementación."
		}
	}
	return "Desarrollo: inicia el servidor para trabajar en el proyecto y ver los cambios al guardar el código."
}

// Prefer conventional launch scripts, then inspect custom names deterministically.
// Validation of each script body still decides whether it is an application launcher.
func launchScriptNames(scripts map[string]string) []string {
	preferred := []string{"dev", "start", "serve", "preview"}
	seen := make(map[string]bool, len(preferred))
	for _, name := range preferred {
		seen[name] = true
	}
	var custom []string
	for name := range scripts {
		if !seen[name] && scriptName.MatchString(name) {
			custom = append(custom, name)
		}
	}
	sort.Strings(custom)
	return append(preferred, custom...)
}

// validJSScript recognizes launchers rather than trusting a script's name.
func validJSScript(body string, read func(string) []byte) bool {
	words := strings.Fields(body)
	if len(words) == 0 || len(words) > 20 {
		return false
	}
	for _, w := range words {
		if !safeToken.MatchString(w) || strings.Contains(w, "..") {
			return false
		}
	}
	verbs := map[string]map[string]bool{
		"vite": {"dev": true, "serve": true, "preview": true}, "next": {"dev": true, "start": true},
		"astro": {"dev": true, "preview": true}, "nuxt": {"dev": true, "start": true, "preview": true},
		"nuxi": {"dev": true, "start": true, "preview": true}, "react-scripts": {"start": true}, "webpack": {"serve": true},
		"ng": {"serve": true, "dev": true, "s": true}, "vue-cli-service": {"serve": true},
	}
	if accepted, ok := verbs[words[0]]; ok {
		options := words[1:]
		if len(options) > 0 && accepted[options[0]] {
			options = options[1:]
		} else if words[0] != "vite" {
			return false
		}
		// Angular accepts an optional workspace application name after "serve".
		if words[0] == "ng" && len(options) > 0 && !strings.HasPrefix(options[0], "-") {
			if !scriptName.MatchString(options[0]) {
				return false
			}
			options = options[1:]
		}
		for i := 0; i < len(options); i++ {
			flag, value, inline := strings.Cut(options[i], "=")
			if flag == "--strictPort" || flag == "--strict-port" || flag == "--turbo" || flag == "--turbopack" || flag == "--no-open" {
				if inline {
					return false
				}
				continue
			}
			if !inline {
				i++
				if i >= len(options) {
					return false
				}
				value = options[i]
			}
			if words[0] == "ng" && (flag == "--configuration" || flag == "-c") {
				for _, name := range strings.Split(value, ",") {
					if !scriptName.MatchString(name) {
						return false
					}
				}
				continue
			}
			switch flag {
			case "--host", "--hostname", "-H":
				if value != "127.0.0.1" && value != "localhost" && value != "::1" {
					return false
				}
			case "--port", "-p":
				port, err := strconv.Atoi(value)
				if err != nil || port < 1 || port > 65535 {
					return false
				}
			case "--mode":
				if value != "development" && value != "production" && value != "test" {
					return false
				}
			case "--config", "-c":
				if !filepath.IsLocal(value) || strings.HasPrefix(value, "-") || read(value) == nil {
					return false
				}
			case "--proxy-config":
				if words[0] != "ng" || !filepath.IsLocal(value) || strings.HasPrefix(value, "-") || read(value) == nil {
					return false
				}
			default:
				return false
			}
		}
		return true
	}
	if len(words) != 2 || !(words[0] == "node" || words[0] == "tsx" || words[0] == "ts-node" || words[0] == "nodemon" || words[0] == "bun") {
		return false
	}
	entry := words[1]
	if strings.HasPrefix(entry, "-") {
		return false
	}
	ext := strings.ToLower(filepath.Ext(entry))
	if ext != ".js" && ext != ".mjs" && ext != ".cjs" && ext != ".ts" && ext != ".mts" && ext != ".cts" {
		return false
	}
	bodyText := string(read(entry))
	return (strings.Contains(bodyText, "listen(") && (strings.Contains(bodyText, "express") || strings.Contains(bodyText, "fastify") || strings.Contains(bodyText, "http") || strings.Contains(bodyText, "koa"))) || (words[0] == "bun" && strings.Contains(bodyText, "Bun.serve("))
}

func detectManager(directory, explicit string, read, ancestorRead func(string) []byte) (string, error) {
	managers := map[string]bool{}
	for _, item := range []struct{ file, manager string }{{"package-lock.json", "npm"}, {"npm-shrinkwrap.json", "npm"}, {"pnpm-lock.yaml", "pnpm"}, {"yarn.lock", "yarn"}, {"bun.lock", "bun"}, {"bun.lockb", "bun"}} {
		if read(item.file) != nil {
			managers[item.manager] = true
		}
	}
	// Workspace-level declarations/locks apply to packages in a monorepo too.
	if explicit == "" && directory != "." && len(managers) == 0 {
		for parent := filepath.Dir(filepath.FromSlash(directory)); ; parent = filepath.Dir(parent) {
			var parentPkg struct {
				PackageManager string `json:"packageManager"`
			}
			_ = json.Unmarshal(ancestorRead(filepath.Join(parent, "package.json")), &parentPkg)
			if parentPkg.PackageManager != "" {
				explicit = parentPkg.PackageManager
				break
			}
			if len(managers) == 0 {
				for _, item := range []struct{ file, manager string }{{"package-lock.json", "npm"}, {"pnpm-lock.yaml", "pnpm"}, {"yarn.lock", "yarn"}, {"bun.lock", "bun"}, {"bun.lockb", "bun"}} {
					if ancestorRead(filepath.Join(parent, item.file)) != nil {
						managers[item.manager] = true
					}
				}
			}
			if parent == "." || len(managers) > 0 {
				break
			}
		}
	}
	if explicit != "" {
		manager := strings.SplitN(explicit, "@", 2)[0]
		if manager != "npm" && manager != "pnpm" && manager != "yarn" && manager != "bun" {
			return "", policy("El gestor declarado en packageManager no tiene un perfil de ejecución compatible.")
		}
		return manager, nil
	}
	if len(managers) > 1 {
		return "", policy("Hay lockfiles de varios gestores. Declara packageManager en package.json para evitar una selección ambigua.")
	}
	for manager := range managers {
		return manager, nil
	}
	return "", policy("No se pudo identificar el gestor de paquetes. Declara packageManager o conserva el lockfile del proyecto.")
}
