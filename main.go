package main

import (
	"fmt"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
)

func main() {
	app := NewApp()

	err := wails.Run(&options.App{
		Title:            "CS2 击杀集锦制作工具",
		Width:            1200,
		Height:           800,
		DisableResize:    false,
		Assets:           assets,
		BackgroundColour: &options.RGBA{R: 20, G: 20, B: 20, A: 1},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		Bind:             []interface{}{app},
	})
	if err != nil {
		printError(fmt.Sprintf("启动失败: %v", err))
	}
}
