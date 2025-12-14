package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Create an instance of the app structure (Business Logic)
	app := NewApp()

	// Create application with options
	err := wails.Run(&options.App{
		Title:  "Kizuna Editor", // Added space for better readability in taskbar
		Width:  1200,            // Wider default for score viewing
		Height: 800,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		// Sync Background Color with CSS (#1e1e1e) to avoid color flashing on startup
		BackgroundColour: &options.RGBA{R: 30, G: 30, B: 30, A: 255},
		OnStartup:        app.startup,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		log.Fatal("Error launching application:", err)
	}
}
