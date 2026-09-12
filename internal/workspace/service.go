package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"merlincode/internal/domain"
)

// ignoredDirNames define directorios que deben excluirse del árbol y listas de archivos
// Declarado a nivel de paquete para evitar asignaciones repetitivas de memoria en recursión
var ignoredDirNames = map[string]bool{
	".git":         true,
	"node_modules": true,
	"dist":         true,
	".cache":       true,
	".idea":        true,
	".vscode":      true,
	"build":        true,
}

// Service administra las operaciones sobre la carpeta activa del proyecto y su árbol de archivos
type Service struct {
	mu               sync.RWMutex
	activeProjectDir string
}

// NewService crea una nueva instancia del servicio de espacio de trabajo
func NewService() *Service {
	return &Service{}
}

// SetActive valida permisos de escritura y define el directorio del proyecto activo
func (s *Service) SetActive(dirPath string) (*domain.ProjectInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if dirPath == "" {
		s.activeProjectDir = ""
		return nil, nil
	}

	cleanPath := filepath.Clean(dirPath)
	if err := VerifyFolderWritePermissions(cleanPath); err != nil {
		return nil, err
	}

	s.activeProjectDir = cleanPath
	return &domain.ProjectInfo{
		ID:   cleanPath,
		Name: filepath.Base(cleanPath),
		Path: cleanPath,
	}, nil
}

// GetActive devuelve la metadata del proyecto activo actualmente o nil
func (s *Service) GetActive() *domain.ProjectInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.activeProjectDir == "" {
		return nil
	}
	return &domain.ProjectInfo{
		ID:   s.activeProjectDir,
		Name: filepath.Base(s.activeProjectDir),
		Path: s.activeProjectDir,
	}
}

// GetActiveDir retorna la ruta absoluta del directorio activo
func (s *Service) GetActiveDir() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.activeProjectDir
}

// WriteFile escribe contenido en un archivo dentro del proyecto activo con verificación de seguridad
func (s *Service) WriteFile(relativePath string, content string) error {
	s.mu.RLock()
	activeDir := s.activeProjectDir
	s.mu.RUnlock()

	fullPath, err := ValidatePathInProject(activeDir, relativePath)
	if err != nil {
		return err
	}

	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("error creando directorios intermedios: %w", err)
	}

	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("error escribiendo archivo: %w", err)
	}

	return nil
}

// ReadFile lee el contenido de un archivo dentro del proyecto activo asegurando sandboxing
func (s *Service) ReadFile(relativePath string) (string, error) {
	s.mu.RLock()
	activeDir := s.activeProjectDir
	s.mu.RUnlock()

	fullPath, err := ValidatePathInProject(activeDir, relativePath)
	if err != nil {
		return "", err
	}

	data, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("error leyendo archivo: %w", err)
	}

	return string(data), nil
}

// ListFiles retorna los archivos y carpetas directamente en la raíz del proyecto activo
func (s *Service) ListFiles() ([]domain.FileItem, error) {
	s.mu.RLock()
	activeDir := s.activeProjectDir
	s.mu.RUnlock()

	if activeDir == "" {
		return nil, domain.ErrNoActiveProject
	}

	entries, err := os.ReadDir(activeDir)
	if err != nil {
		return nil, fmt.Errorf("error leyendo directorio de proyecto: %w", err)
	}

	var items []domain.FileItem
	for _, entry := range entries {
		name := entry.Name()
		if ignoredDirNames[name] {
			continue
		}
		info, err := entry.Info()
		size := int64(0)
		if err == nil {
			size = info.Size()
		}
		items = append(items, domain.FileItem{
			Name:  name,
			Path:  filepath.Join(activeDir, name),
			IsDir: entry.IsDir(),
			Size:  size,
		})
	}

	return items, nil
}

// GetTree retorna la estructura jerárquica de archivos y carpetas del proyecto activo
func (s *Service) GetTree() ([]domain.FileNode, error) {
	s.mu.RLock()
	activeDir := s.activeProjectDir
	s.mu.RUnlock()

	if activeDir == "" {
		return nil, domain.ErrNoActiveProject
	}

	return s.readDirTree(activeDir, "", 1, 4)
}

// readDirTree lee recursivamente directorios hasta una profundidad máxima (maxDepth)
func (s *Service) readDirTree(absDir string, relDir string, currentDepth, maxDepth int) ([]domain.FileNode, error) {
	entries, err := os.ReadDir(absDir)
	if err != nil {
		return nil, err
	}

	var dirs []domain.FileNode
	var files []domain.FileNode

	for _, entry := range entries {
		name := entry.Name()
		if ignoredDirNames[name] || strings.HasPrefix(name, ".merlin_perm_") {
			continue
		}

		childRel := filepath.Join(relDir, name)
		childAbs := filepath.Join(absDir, name)
		isDir := entry.IsDir()

		var size int64
		if info, err := entry.Info(); err == nil {
			size = info.Size()
		}

		node := domain.FileNode{
			Name:     name,
			RelPath:  filepath.ToSlash(childRel),
			FullPath: childAbs,
			IsDir:    isDir,
			Size:     size,
		}

		if isDir && currentDepth < maxDepth {
			children, err := s.readDirTree(childAbs, childRel, currentDepth+1, maxDepth)
			if err == nil {
				node.Children = children
			}
		}

		if isDir {
			dirs = append(dirs, node)
		} else {
			files = append(files, node)
		}
	}

	sort.Slice(dirs, func(i, j int) bool {
		return strings.ToLower(dirs[i].Name) < strings.ToLower(dirs[j].Name)
	})
	sort.Slice(files, func(i, j int) bool {
		return strings.ToLower(files[i].Name) < strings.ToLower(files[j].Name)
	})

	var nodes []domain.FileNode
	nodes = append(nodes, dirs...)
	nodes = append(nodes, files...)
	return nodes, nil
}

// OpenExplorer delega la apertura del explorador en la ruta indicada o en el proyecto activo
func (s *Service) OpenExplorer(targetPath string) error {
	target := targetPath
	if target == "" {
		s.mu.RLock()
		target = s.activeProjectDir
		s.mu.RUnlock()
	}

	if target == "" {
		return domain.ErrNoTargetPath
	}

	return OpenInExplorer(target)
}
