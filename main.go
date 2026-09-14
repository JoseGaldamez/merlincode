package main

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"

	"merlincode/internal/app"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Crear instancia de la fachada de la aplicación
	application := app.NewApp()
	initialWindowState := application.InitialWindowState()

	// Iniciar la aplicación de escritorio Wails
	err := wails.Run(&options.App{
		Title:     "Merlin Code",
		Width:     initialWindowState.Width,
		Height:    initialWindowState.Height,
		MinWidth:  800,
		MinHeight: 600,
		Frameless: true,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 11, G: 14, B: 16, A: 1},
		OnStartup:        application.Startup,
		OnBeforeClose:    application.BeforeClose,
		Bind: []interface{}{
			application,
		},
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			DisableWindowIcon:    false,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
