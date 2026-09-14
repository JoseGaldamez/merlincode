package workspace

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"merlincode/internal/domain"
)

func TestSandboxRejectsExternalPathsAndLinks(t *testing.T) {
	project, outside := t.TempDir(), t.TempDir()
	svc := NewService()
	defer svc.Close()
	if _, err := svc.SetActive(project); err != nil {
		t.Fatal(err)
	}
	secret := filepath.Join(outside, "secret.txt")
	if err := os.WriteFile(secret, []byte("untouched"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{secret, "../secret.txt"} {
		if _, err := svc.ReadFile(name); !errors.Is(err, domain.ErrAccessDenied) {
			t.Fatalf("read %q: %v", name, err)
		}
		if err := svc.WriteFile(name, "changed"); !errors.Is(err, domain.ErrAccessDenied) {
			t.Fatalf("write %q: %v", name, err)
		}
		if err := svc.OpenExplorer(name); !errors.Is(err, domain.ErrAccessDenied) {
			t.Fatalf("open %q: %v", name, err)
		}
	}
	t.Run("link or junction", func(t *testing.T) {
		link := filepath.Join(project, "escape")
		if runtime.GOOS == "windows" {
			// Junctions do not require the symlink privilege. Fixed test paths only.
			cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command", "New-Item -ItemType Junction -Path $env:MERLIN_TEST_LINK -Target $env:MERLIN_TEST_TARGET | Out-Null")
			cmd.Env = append(os.Environ(), "MERLIN_TEST_LINK="+link, "MERLIN_TEST_TARGET="+outside)
			if output, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("create junction: %v %s", err, output)
			}
		} else if err := os.Symlink(outside, link); err != nil {
			t.Fatal(err)
		}
		defer os.Remove(link)
		for _, name := range []string{"escape/secret.txt", "escape/new/nested.txt"} {
			if err := svc.WriteFile(name, "changed"); !errors.Is(err, domain.ErrAccessDenied) {
				t.Fatalf("write escaped: %v", err)
			}
			if _, err := svc.ReadFile(name); !errors.Is(err, domain.ErrAccessDenied) {
				t.Fatalf("read escaped: %v", err)
			}
		}
		if err := svc.OpenExplorer("escape"); !errors.Is(err, domain.ErrAccessDenied) {
			t.Fatalf("explorer escaped: %v", err)
		}
		// Even bypassing the preflight validator, the OS root must reject the link.
		if err := svc.root.WriteFile("escape/secret.txt", []byte("changed"), 0600); err == nil {
			t.Fatal("root permitted escape")
		}
		if _, err := svc.root.Open("escape/secret.txt"); err == nil {
			t.Fatal("root permitted read escape")
		}
		tree, err := svc.GetTree()
		if err != nil {
			t.Fatal(err)
		}
		for _, node := range tree {
			if node.Name == "escape" {
				t.Fatal("tree exposed external link")
			}
		}
	})
	data, err := os.ReadFile(secret)
	if err != nil || string(data) != "untouched" {
		t.Fatalf("outside file changed: %v", err)
	}
}

func TestFileLimitsAndPermissionProbe(t *testing.T) {
	dir := t.TempDir()
	oldProbe := filepath.Join(dir, fmt.Sprintf(".merlin_perm_%d.tmp", os.Getpid()))
	if err := os.WriteFile(oldProbe, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	svc := NewService()
	defer svc.Close()
	if _, err := svc.SetActive(dir); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(oldProbe)
	if err != nil || string(data) != "keep" {
		t.Fatal("permission probe overwrote an existing file")
	}
	if err := svc.WriteFile("large", strings.Repeat("a", MaxFileBytes+1)); !errors.Is(err, domain.ErrFileTooLarge) {
		t.Fatal(err)
	}
	f, err := os.Create(filepath.Join(dir, "large"))
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Truncate(MaxFileBytes + 1); err != nil {
		t.Fatal(err)
	}
	f.Close()
	if _, err := svc.ReadFile("large"); !errors.Is(err, domain.ErrFileTooLarge) {
		t.Fatal(err)
	}
	if _, err := svc.ReadFile("."); !errors.Is(err, domain.ErrAccessDenied) {
		t.Fatal(err)
	}
}

func TestPermissionDeniedIsPropagated(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix mode test; Windows ACLs are tested in platform/config")
	}
	dir := t.TempDir()
	svc := NewService()
	defer svc.Close()
	if _, err := svc.SetActive(dir); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(dir, "private")
	if err := os.WriteFile(file, []byte("secret"), 0000); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(file, 0600)
	if f, err := os.Open(file); err == nil {
		f.Close()
		t.Skip("privileged process bypasses permissions")
	}
	if _, err := svc.ReadFile("private"); err == nil {
		t.Fatal("read ignored permissions")
	}
	if err := svc.WriteFile("private", "changed"); err == nil {
		t.Fatal("write ignored permissions")
	}
}
