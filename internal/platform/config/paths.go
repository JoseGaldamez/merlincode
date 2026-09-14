package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// AppDirName es el nombre del subdirectorio donde la aplicación almacena su configuración y metadata.
const AppDirName = "merlincode"

// EnsurePrivateDirectory crea la carpeta si no existe y asegura que sus permisos sean 0700.
func EnsurePrivateDirectory(dir string) error {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("no se pudo crear el directorio privado %s: %w", dir, err)
	}
	if err := restrictPermissions(dir, true); err != nil {
		return fmt.Errorf("no se pudieron restringir los permisos del directorio %s: %w", dir, err)
	}
	return nil
}

// GetAppDir retorna la ruta absoluta del directorio de datos de merlincode dentro de UserConfigDir
// y se asegura de que exista con permisos restringidos (0700).
func GetAppDir() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("no se pudo resolver el directorio de configuración de usuario: %w", err)
	}
	appDir := filepath.Join(configDir, AppDirName)
	if err := EnsurePrivateDirectory(appDir); err != nil {
		return "", err
	}
	return appDir, nil
}

// GetConfigFilePath retorna la ruta completa hacia un archivo dentro del directorio de configuración de la app.
func GetConfigFilePath(fileName string) (string, error) {
	appDir, err := GetAppDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(appDir, fileName), nil
}

// EnsurePrivateFile repairs permissions before reading any legacy credentials.
func EnsurePrivateFile(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("la configuración no es un archivo regular")
	}
	return restrictPermissions(path, false)
}
