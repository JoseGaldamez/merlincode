package runner

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// resolveExecutable resolves profile-selected programs through the host PATH and
// platform extension rules. The platform adapter handles Windows batch launchers.
func resolveExecutable(name string, args []string, root *os.Root, directory string) (string, []string, error) {
	if strings.Contains(name, "/") {
		if _, err := readProject(root, directory, name); err != nil {
			return "", nil, policy("No se pudo acceder al intérprete del entorno virtual.")
		}
		return filepath.Join(directory, filepath.FromSlash(name)), args, nil
	}
	path, err := exec.LookPath(name)
	if err != nil && name == "python" && runtime.GOOS != "windows" {
		path, err = exec.LookPath("python3")
	}
	if err != nil {
		return "", nil, policy("No está instalado o disponible el ejecutable " + name + ". Merlin no instala herramientas automáticamente.")
	}
	return path, args, nil
}

func executionEnvironment() []string {
	// Environment comes from the host, never from model-supplied name/value pairs.
	// Provider credentials live in the native keyring and are never read here.
	var env []string
	for _, item := range os.Environ() {
		key, _, _ := strings.Cut(item, "=")
		upper := strings.ToUpper(key)
		if strings.Contains(upper, "API_KEY") || strings.Contains(upper, "TOKEN") || strings.Contains(upper, "SECRET") || strings.Contains(upper, "PASSWORD") || upper == "NODE_OPTIONS" || strings.HasPrefix(upper, "NPM_CONFIG_") || upper == "PYTHONSTARTUP" || upper == "PYTHONPATH" || upper == "PYTHONUNBUFFERED" || upper == "COREPACK_ENABLE_NETWORK" || upper == "COREPACK_ENABLE_DOWNLOAD_PROMPT" || upper == "NO_COLOR" || upper == "GOFLAGS" || upper == "GOTOOLCHAIN" || upper == "GOPROXY" || upper == "GOSUMDB" || upper == "UV_PYTHON_DOWNLOADS" || upper == "UV_OFFLINE" {
			continue
		}
		env = append(env, item)
	}
	env = append(env, "PYTHONUNBUFFERED=1", "NO_COLOR=1", "COREPACK_ENABLE_NETWORK=0", "COREPACK_ENABLE_DOWNLOAD_PROMPT=0", "GOTOOLCHAIN=local", "GOPROXY=off", "GOSUMDB=off", "GOFLAGS=-mod=readonly", "UV_PYTHON_DOWNLOADS=never", "UV_OFFLINE=1")
	if runtime.GOOS == "windows" {
		env = append(env, "NPM_CONFIG_SCRIPT_SHELL="+filepath.Join(os.Getenv("SystemRoot"), "System32", "cmd.exe"))
	} else {
		env = append(env, "NPM_CONFIG_SCRIPT_SHELL=/bin/sh")
	}
	return env
}
