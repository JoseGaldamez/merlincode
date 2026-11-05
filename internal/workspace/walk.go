package workspace

import (
	"io/fs"
	"path/filepath"
	"strings"

	"merlincode/internal/domain"
)

// WalkFiles returns every regular file under the active project whose extension is in exts
// (lowercase, including the dot; nil matches everything), skipping the same directories GetTree
// ignores. Unlike GetTree it is not limited to depth 4, but stops once limit files have matched
// (limit <= 0 means unlimited). Intended for callers that need a full project-wide file list,
// such as a repo map, while still only ever touching the sandboxed root.
func (s *Service) WalkFiles(exts map[string]bool, limit int) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.activeProjectDir == "" {
		return nil, domain.ErrNoActiveProject
	}
	var files []string
	err := fs.WalkDir(s.root.FS(), ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		name := d.Name()
		if d.IsDir() {
			if path != "." && ignoredDirNames[name] {
				return fs.SkipDir
			}
			return nil
		}
		if d.Type()&(fs.ModeSymlink|fs.ModeIrregular) != 0 {
			return nil
		}
		if exts != nil && !exts[strings.ToLower(filepath.Ext(name))] {
			return nil
		}
		files = append(files, filepath.ToSlash(path))
		if limit > 0 && len(files) >= limit {
			return fs.SkipAll
		}
		return nil
	})
	return files, err
}
