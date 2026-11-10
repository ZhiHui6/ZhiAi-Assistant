package main

import (
	"embed"
	"io/fs"
	"log"
	"os"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"

	"zhi-ai-assistant/internal/buildinfo"
	"zhi-ai-assistant/internal/i18n"
	"zhi-ai-assistant/internal/platform"
	"zhi-ai-assistant/internal/service"
)

//go:embed all:frontend
var embedded embed.FS

func frontendFS() fs.FS {
	sub, err := fs.Sub(embedded, "frontend")
	if err != nil {
		log.Fatal(err)
	}
	return sub
}

func main() {
	cleanupUpdateArtifacts()
	for _, arg := range os.Args[1:] {
		if arg == "--settings" {
			runSettingsWindow()
			return
		}
	}
	runSidebar()
}

func cleanupUpdateArtifacts() {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	_ = os.Remove(exe + ".old")
	_ = os.Remove(exe + ".new")
}

func runSidebar() {
	app, err := service.NewApp()
	if err != nil {
		log.Fatal(err)
	}
	app.PrepositionWindow()

	if !platform.SidebarRunning() {
		app.StartHost()
	}

	err = wails.Run(&options.App{
		Title:                    i18n.T(buildinfo.AppName),
		Width:                    340,
		Height:                   900,
		MinWidth:                 340,
		MinHeight:                400,
		Frameless:                true,
		AlwaysOnTop:              false,
		DisableResize:            true,
		EnableDefaultContextMenu: false,
		StartHidden:              false,
		BackgroundColour:         &options.RGBA{R: 0, G: 0, B: 0, A: 0},
		AssetServer: &assetserver.Options{
			Assets: frontendFS(),
		},
		OnStartup:     app.Startup,
		OnBeforeClose: app.BeforeClose,
		OnShutdown:    app.Shutdown,
		Bind: []interface{}{
			app,
			app.Notes(),
			app.Todos(),
			app.Calendar(),
			app.Shelf(),
			app.Launch(),
			app.Mount(),
			app.Timers(),
			app.Icons(),
			app.Thumbs(),
			app.Usage(),
			app.WindowService(),
		},
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId: "zhi-ai-assistant-8f2a1c",
			OnSecondInstanceLaunch: func(data options.SecondInstanceData) {
				app.RevealWindow()
			},
		},
		DragAndDrop: &options.DragAndDrop{
			EnableFileDrop:     true,
			DisableWebViewDrop: false,
		},
		Windows: &windows.Options{
			WebviewIsTransparent:              true,
			WindowIsTranslucent:               true,
			DisableWindowIcon:                 true,
			WindowClassName:                   service.WindowClass,
			DisableFramelessWindowDecorations: true,
			BackdropType:                      windows.None,
		},
	})
	if err != nil {
		log.Fatal(err)
	}
}

func runSettingsWindow() {
	if platform.FocusExistingSettingsWindow() {
		return
	}

	app, err := service.NewSettingsApp()
	if err != nil {
		log.Fatal(err)
	}

	err = wails.Run(&options.App{
		Title:                    i18n.T("智爱小助手设置"),
		Width:                    1120,
		Height:                   630,
		DisableResize:            true,
		Frameless:                true,
		StartHidden:              true,
		BackgroundColour:         &options.RGBA{R: 12, G: 14, B: 20, A: 255},
		EnableDefaultContextMenu: false,
		AssetServer: &assetserver.Options{
			Assets: frontendFS(),
		},
		OnStartup:     app.Startup,
		OnBeforeClose: app.BeforeCloseSettings,
		OnShutdown:    app.Shutdown,
		Bind:          []interface{}{app},
		Windows: &windows.Options{
			WindowClassName:      service.SettingsWindowClass,
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			BackdropType:         windows.None,
		},
	})
	if err != nil {
		log.Fatal(err)
	}
}
