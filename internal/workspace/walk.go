package workspace

import (
	"io/fs"
	"path/filepath"
	"strings"

	"merlincode/internal/domain"
)

// WalkFiles recorre todo el árbol del proyecto activo, sin el límite de profundidad de cuatro
// niveles que aplica GetTree, y devuelve las rutas relativas (siempre con separador "/" vía
// filepath.ToSlash) de los archivos regulares cuya extensión aparezca en exts. exts nil o vacío
// significa "todas las extensiones"; limit<=0 significa "sin límite de resultados".
//
// Reutiliza el mismo os.Root sandboxeado que el resto del servicio a través de Root.FS(), así que
// nunca sale del proyecto activo ni sigue enlaces simbólicos fuera de él; se excluyen los mismos
// directorios ignorados que usa GetTree (ignoredDirNames) y las entradas symlink/irregulares.
func (s *Service) WalkFiles(exts map[string]bool, limit int) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.activeProjectDir == "" {
		return nil, domain.ErrNoActiveProject
	}

	var paths []string
	err := fs.WalkDir(s.root.FS(), ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// Una entrada individual inaccesible no debe abortar el recorrido completo.
			if d != nil && d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			if path != "." && ignoredDirNames[d.Name()] {
				return fs.SkipDir
			}
			return nil
		}
		if d.Type()&(fs.ModeSymlink|fs.ModeIrregular) != 0 {
			return nil
		}
		if len(exts) > 0 && !exts[strings.ToLower(filepath.Ext(d.Name()))] {
			return nil
		}
		paths = append(paths, filepath.ToSlash(path))
		if limit > 0 && len(paths) >= limit {
			return fs.SkipAll
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return paths, nil
}
