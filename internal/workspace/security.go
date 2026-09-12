package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"merlincode/internal/domain"
)

// VerifyFolderWritePermissions comprueba que la ruta exista, sea una carpeta y tenga permisos de escritura
func VerifyFolderWritePermissions(dirPath string) error {
	info, err := os.Stat(dirPath)
	if err != nil {
		return fmt.Errorf("%w: %v", domain.ErrFolderNotExist, err)
	}
	if !info.IsDir() {
		return domain.ErrNotADirectory
	}

	// Probar creando y eliminando un archivo temporal para verificar permisos reales de escritura
	testFile := filepath.Join(dirPath, fmt.Sprintf(".merlin_perm_%d.tmp", os.Getpid()))
	if err := os.WriteFile(testFile, []byte("ok"), 0644); err != nil {
		return fmt.Errorf("%w: %v", domain.ErrWritePermissionDenied, err)
	}
	_ = os.Remove(testFile)
	return nil
}

// ValidatePathInProject asegura que la ruta relativa o absoluta esté estrictamente contenida
// dentro del directorio del proyecto activo, evitando ataques de Path Traversal (ej. "../archivo")
func ValidatePathInProject(activeDir string, relOrAbsPath string) (string, error) {
	if activeDir == "" {
		return "", domain.ErrNoActiveProject
	}

	cleanProject, err := filepath.Abs(filepath.Clean(activeDir))
	if err != nil {
		return "", fmt.Errorf("error resolviendo ruta base del proyecto: %w", err)
	}

	var targetAbs string
	if filepath.IsAbs(relOrAbsPath) {
		targetAbs = filepath.Clean(relOrAbsPath)
	} else {
		targetAbs = filepath.Clean(filepath.Join(cleanProject, relOrAbsPath))
	}

	targetAbs, err = filepath.Abs(targetAbs)
	if err != nil {
		return "", fmt.Errorf("error resolviendo ruta de destino: %w", err)
	}

	rel, err := filepath.Rel(cleanProject, targetAbs)
	if err != nil {
		return "", fmt.Errorf("error calculando ruta relativa: %w", err)
	}

	// Comprobar si la ruta intenta escapar de la carpeta raíz del proyecto
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("%w (%s)", domain.ErrAccessDenied, activeDir)
	}

	return targetAbs, nil
}
