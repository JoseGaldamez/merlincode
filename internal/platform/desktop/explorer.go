package desktop

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"merlincode/internal/domain"
)

// OpenInExplorer abre la carpeta o selecciona el archivo en el explorador nativo del sistema operativo
// (Windows Explorer, macOS Finder, o Linux file manager vía xdg-open).
func OpenInExplorer(targetPath string) error {
	if targetPath == "" {
		return domain.ErrNoTargetPath
	}

	cleanPath := filepath.Clean(targetPath)
	info, err := os.Stat(cleanPath)
	if err != nil {
		return fmt.Errorf("la ruta no existe: %w", err)
	}

	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		if info.IsDir() {
			cmd = exec.Command("explorer", cleanPath)
		} else {
			cmd = exec.Command("explorer", "/select,", cleanPath)
		}
	case "darwin":
		if info.IsDir() {
			cmd = exec.Command("open", cleanPath)
		} else {
			cmd = exec.Command("open", "-R", cleanPath)
		}
	default: // linux, bsd, etc.
		if info.IsDir() {
			cmd = exec.Command("xdg-open", cleanPath)
		} else {
			cmd = exec.Command("xdg-open", filepath.Dir(cleanPath))
		}
	}

	return cmd.Start()
}
