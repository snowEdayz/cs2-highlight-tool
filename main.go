package main

import (
	application "cs2-highlight-tool-v2/internal/app"
	"cs2-highlight-tool-v2/internal/updater"
	"embed"
	"os"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed wails.json
var wailsConfigData []byte

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--apply-update" {
		if err := updater.RunApplyMode(os.Args[2:]); err != nil {
			println("Update failed:", err.Error())
			os.Exit(1)
		}
		return
	}

	backend := application.New(wailsConfigData)

	// Create application with options
	err := wails.Run(&options.App{
		Title:     "CS2 Highlight Tool",
		Width:     920,
		Height:    720,
		Frameless: true,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 17, G: 19, B: 18, A: 1},
		OnStartup:        backend.Startup,
		OnShutdown:       backend.Shutdown,
		Bind: []interface{}{
			backend,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
