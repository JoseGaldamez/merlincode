package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetConfigFilePath(t *testing.T) {
	filePath, err := GetConfigFilePath("test_file.json")
	if err != nil {
		t.Fatalf("GetConfigFilePath falló: %v", err)
	}

	if filepath.Base(filePath) != "test_file.json" {
		t.Fatalf("Nombre de archivo esperado test_file.json, obtenido: %s", filepath.Base(filePath))
	}

	dir := filepath.Dir(filePath)
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("El directorio de la aplicación no fue creado: %v", err)
	}
	if !info.IsDir() {
		t.Fatalf("Se esperaba que fuera un directorio: %s", dir)
	}
}

func TestEnsurePrivateDirectory(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "merlin_config_test_*")
	if err != nil {
		t.Fatalf("Error creando tempDir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	subDir := filepath.Join(tempDir, "nested", "private")
	if err := EnsurePrivateDirectory(subDir); err != nil {
		t.Fatalf("EnsurePrivateDirectory falló: %v", err)
	}

	info, err := os.Stat(subDir)
	if err != nil {
		t.Fatalf("El directorio no existe: %v", err)
	}
	if !info.IsDir() {
		t.Fatalf("Se esperaba un directorio: %s", subDir)
	}
}
