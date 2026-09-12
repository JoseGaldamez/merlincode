package domain

// ProjectInfo representa un proyecto anclado a un directorio local en el disco
type ProjectInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Path string `json:"path"`
}

// FileItem representa un archivo o directorio dentro del espacio de trabajo
type FileItem struct {
	Name  string `json:"name"`
	Path  string `json:"path"`
	IsDir bool   `json:"isDir"`
	Size  int64  `json:"size"`
}

// FileNode representa un nodo jerárquico en el árbol de directorios del proyecto
type FileNode struct {
	Name     string     `json:"name"`
	RelPath  string     `json:"relPath"`
	FullPath string     `json:"fullPath"`
	IsDir    bool       `json:"isDir"`
	Size     int64      `json:"size"`
	Children []FileNode `json:"children,omitempty"`
}
