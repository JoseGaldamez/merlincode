package window

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"merlincode/internal/domain"
)

// Service gestiona la configuración y persistencia de las dimensiones de la ventana y paneles
type Service struct {
	mu       sync.RWMutex
	state    domain.WindowState
	filePath string
}

// GetDefaultState retorna los valores iniciales predeterminados (1920x1080, panel izq abierto, der cerrado)
func GetDefaultState() domain.WindowState {
	return domain.WindowState{
		Width:          1920,
		Height:         1080,
		Maximised:      false,
		LeftPanelOpen:  true,
		RightPanelOpen: false,
	}
}

// resolveConfigFilePath obtiene la ruta absoluta hacia %APPDATA%/merlincode/window.json
func resolveConfigFilePath() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir = "."
	}
	appDir := filepath.Join(configDir, "merlincode")
	_ = os.MkdirAll(appDir, 0755)
	return filepath.Join(appDir, "window.json")
}

// NewService crea una nueva instancia cargando el estado persistido o inicializando con defaults
func NewService() *Service {
	filePath := resolveConfigFilePath()
	return NewServiceWithCustomPath(filePath)
}

// NewServiceWithCustomPath permite inyectar una ruta personalizada para facilitar pruebas unitarias aisladas
func NewServiceWithCustomPath(filePath string) *Service {
	state := loadStateFromFile(filePath)
	return &Service{
		state:    state,
		filePath: filePath,
	}
}

// loadStateFromFile lee y deserializa el archivo de configuración si existe
func loadStateFromFile(filePath string) domain.WindowState {
	state := GetDefaultState()

	data, err := os.ReadFile(filePath)
	if err != nil {
		return state
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return state
	}

	if w, ok := raw["width"].(float64); ok && int(w) >= 600 {
		state.Width = int(w)
	}
	if h, ok := raw["height"].(float64); ok && int(h) >= 400 {
		state.Height = int(h)
	}
	if max, ok := raw["maximised"].(bool); ok {
		state.Maximised = max
	}
	if l, ok := raw["left_panel_open"].(bool); ok {
		state.LeftPanelOpen = l
	}
	if r, ok := raw["right_panel_open"].(bool); ok {
		state.RightPanelOpen = r
	}

	return state
}

// persistState serializa el estado a disco (debe llamarse con el candado adquirido o bloqueado internamente)
func (s *Service) persistState() {
	if s.state.Width < 600 || s.state.Height < 400 {
		return
	}
	data, err := json.MarshalIndent(s.state, "", "  ")
	if err == nil {
		_ = os.WriteFile(s.filePath, data, 0644)
	}
}

// GetState devuelve una copia segura del estado actual de la ventana
func (s *Service) GetState() domain.WindowState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state
}

// SaveSize actualiza las dimensiones de la ventana y las persiste de forma concurrente y segura
func (s *Service) SaveSize(width int, height int, isMaximised bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !isMaximised && width >= 600 && height >= 400 {
		s.state.Width = width
		s.state.Height = height
	}
	s.state.Maximised = isMaximised
	s.persistState()
}

// SavePanelsState actualiza el estado de apertura de los paneles laterales y lo persiste
func (s *Service) SavePanelsState(leftOpen bool, rightOpen bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.state.LeftPanelOpen = leftOpen
	s.state.RightPanelOpen = rightOpen
	s.persistState()
}

// GetPanelsState retorna el estado actual de los paneles
func (s *Service) GetPanelsState() domain.PanelsState {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return domain.PanelsState{
		LeftOpen:  s.state.LeftPanelOpen,
		RightOpen: s.state.RightPanelOpen,
	}
}

// UpdateWindowState permite actualizar de forma atómica todas las propiedades del estado
func (s *Service) UpdateWindowState(state domain.WindowState) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if state.Width >= 600 && state.Height >= 400 {
		s.state = state
		s.persistState()
	}
}
