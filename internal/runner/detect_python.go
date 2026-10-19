package runner

import (
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"merlincode/internal/domain"
)

var fastAPIEntry = regexp.MustCompile(`(?m)^([A-Za-z_]\w*)\s*(?::[^=\n]+)?=\s*FastAPI\(`)
var moduleName = regexp.MustCompile(`^[A-Za-z_]\w*(\.[A-Za-z_]\w*)*$`)

func (d *profileDetector) detectPython() error {
	root, req, read := d.root, d.request, d.read
	pyproject := string(read("pyproject.toml"))
	uv := len(read("uv.lock")) > 0
	poetry := len(read("poetry.lock")) > 0
	pipenv := len(read("Pipfile.lock")) > 0 || len(read("Pipfile")) > 0
	read("requirements.txt")
	if boolCount(uv, poetry, pipenv) > 1 {
		return policy("Hay varios gestores Python. Conserva una configuración inequívoca antes de ejecutar.")
	}
	python := "python"
	prefix := []string{}
	if uv || strings.Contains(pyproject, "[tool.uv]") {
		python, prefix = "uv", []string{"run", "--no-sync", "python"}
	} else if poetry || strings.Contains(pyproject, "[tool.poetry]") {
		python, prefix = "poetry", []string{"run", "python"}
	} else if pipenv {
		python, prefix = "pipenv", []string{"run", "python"}
	}
	if python == "python" {
		for _, venv := range []string{".venv", "venv"} {
			candidate := filepath.Join(venv, "bin", "python")
			if runtime.GOOS == "windows" {
				candidate = filepath.Join(venv, "Scripts", "python.exe")
			}
			if info, err := root.Lstat(venv); err == nil {
				if !info.IsDir() {
					return domain.ErrAccessDenied
				}
				if read(candidate) == nil {
					return policy("El entorno virtual existe pero su intérprete no es accesible sin enlaces. Revisa el entorno antes de ejecutar.")
				}
				python = filepath.ToSlash(candidate)
				break
			}
		}
	}
	entries := []string{"main.py", "app.py", "server.py", "manage.py"}
	if req.Entry != "" {
		entries = []string{req.Entry}
	}
	for _, entry := range entries {
		data := read(entry)
		if data == nil || filepath.Ext(entry) != ".py" {
			continue
		}
		body := string(data)
		scriptPath := filepath.ToSlash(entry)
		if strings.HasPrefix(scriptPath, "-") {
			scriptPath = "./" + scriptPath
		}
		title, reason := "Servidor Python · "+entry, "Entrada de servidor detectada en "+entry
		args := append([]string{}, prefix...)
		if filepath.Base(entry) == "manage.py" && strings.Contains(body, "execute_from_command_line") {
			args = append(args, scriptPath, "runserver", "127.0.0.1:8000", "--noreload")
		} else if match := fastAPIEntry.FindStringSubmatch(body); match != nil {
			module := strings.TrimSuffix(filepath.ToSlash(entry), ".py")
			if !moduleName.MatchString(strings.ReplaceAll(module, "/", ".")) {
				continue
			}
			args = append(args, "-m", "uvicorn", strings.ReplaceAll(module, "/", ".")+":"+match[1], "--host", "127.0.0.1", "--port", "8000")
		} else if strings.Contains(body, "Flask(") {
			args = append(args, "-m", "flask", "--app", scriptPath, "run", "--host", "127.0.0.1", "--port", "5000")
		} else if strings.Contains(body, "http.server") && strings.Contains(body, "serve_forever") {
			args = append(args, "-u", scriptPath)
		} else if req.Entry != "" || entry == "main.py" {
			// An explicitly selected file is an application entry too, even if it
			// prints once and exits. Never interpret a filename as a Python option.
			args = append(args, "-u", scriptPath)
			title, reason = "Script Python · "+entry, "Archivo Python seleccionado dentro del proyecto. Puede finalizar tras ejecutar su tarea."
		} else {
			continue
		}
		d.add(title, displayCommand(python, args), reason, python, args...)
	}
	if req.Entry != "" && filepath.Ext(req.Entry) == ".py" && len(d.recipes) == 0 {
		return policy("No se pudo leer el archivo Python seleccionado dentro del proyecto. Comprueba que exista y que no sea un enlace ni exceda 4 MiB.")
	}

	return nil
}
