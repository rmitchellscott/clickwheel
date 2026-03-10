package main

import (
	"embed"
	"fmt"
	"os"
	"strings"

	"clickwheel/internal/restore"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
)

//go:embed all:frontend/dist
var assets embed.FS

var version = "dev"

func main() {
	if len(os.Args) > 1 && strings.HasPrefix(os.Args[1], "--restore-") {
		var err error
		switch os.Args[1] {
		case "--restore-partition":
			err = restore.RunPartitionSubcommand(os.Args[2:])
		case "--restore-write-fw":
			err = restore.RunWriteFirmwareSubcommand(os.Args[2:])
		default:
			err = fmt.Errorf("unknown restore subcommand: %s", os.Args[1])
		}
		if err != nil {
			os.Stderr.WriteString(err.Error())
			os.Exit(1)
		}
		os.Exit(0)
	}

	app := NewApp()

	err := wails.Run(&options.App{
		Title:     "Clickwheel",
		Width:     1050,
		Height:    700,
		MinWidth:  800,
		MinHeight: 500,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		Bind: []interface{}{
			app,
		},
		Mac: &mac.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
