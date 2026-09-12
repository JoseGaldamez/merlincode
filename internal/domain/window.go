package domain

// PanelsState representa el estado abierto o colapsado de los paneles laterales
type PanelsState struct {
	LeftOpen  bool `json:"leftOpen"`
	RightOpen bool `json:"rightOpen"`
}

// WindowState almacena las dimensiones, modo de maximizado y estado de los paneles
type WindowState struct {
	Width          int  `json:"width"`
	Height         int  `json:"height"`
	Maximised      bool `json:"maximised"`
	LeftPanelOpen  bool `json:"left_panel_open"`
	RightPanelOpen bool `json:"right_panel_open"`
}
