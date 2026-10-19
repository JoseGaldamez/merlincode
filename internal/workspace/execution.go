package workspace

import (
	"os"
	"path/filepath"
	"strings"

	"merlincode/internal/domain"
)

// WithExecutionRoot pins a validated directory for the duration of fn. Project
// changes wait until fn returns. Callers may retain their own root for a server.
func (s *Service) WithExecutionRoot(relative string, fn func(root, projectRoot *os.Root, directory, project string) error) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if relative == "" {
		relative = "."
	}
	if filepath.IsAbs(relative) || !filepath.IsLocal(relative) || strings.ContainsAny(relative, "\x00\r\n") {
		return domain.ErrAccessDenied
	}
	full, err := ValidatePathInProject(s.activeProjectDir, relative)
	if err != nil {
		return err
	}
	name, err := filepath.Rel(s.activeProjectDir, full)
	if err != nil {
		return domain.ErrAccessDenied
	}
	root, err := s.root.OpenRoot(name)
	if err != nil {
		return domain.ErrAccessDenied
	}
	defer root.Close()
	return fn(root, s.root, full, s.activeProjectDir)
}
