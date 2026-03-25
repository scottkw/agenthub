// AgentHub — unified binary: desktop GUI, CLI commands, and daemon mode.
//
// Dispatch strategy (locked in STATE.md):
//   - No args, or first arg starts with "-" (except --help/-h) → GUI
//   - --help or -h → print usage and exit
//   - Otherwise → CLI command dispatch
//
// Build for production:  wails build
// Run in dev mode:       wails dev
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/agenthub/agenthub/internal/daemon"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

func main() {
	// GUI mode: no args, or first arg is a flag (except help).
	if len(os.Args) == 1 || strings.HasPrefix(os.Args[1], "-") {
		if len(os.Args) > 1 && (os.Args[1] == "--help" || os.Args[1] == "-h") {
			usage()
			return
		}
		runGUI()
		return
	}
	// CLI mode: dispatch to command handler.
	runCLI(os.Args[1:])
}

func runGUI() {
	app := NewApp()

	err := wails.Run(&options.App{
		Title:             "AgentHub",
		Width:             1200,
		Height:            800,
		MinWidth:          800,
		MinHeight:         600,
		HideWindowOnClose: true,
		BackgroundColour:  &options.RGBA{R: 0x1b, G: 0x26, B: 0x36, A: 0xff},
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup:     app.startup,
		OnShutdown:    app.shutdown,
		OnBeforeClose: app.beforeClose,
		Bind: []interface{}{
			app,
		},
	})
	if err != nil {
		panic(err)
	}
}

func runCLI(args []string) {
	cmd := args[0]

	// Daemon sub-command: run daemon mode without EnsureDaemon, or manage service lifecycle.
	// daemon status needs a running daemon — fall through to EnsureDaemon path.
	if cmd == "daemon" && len(args) > 1 && args[1] == "status" {
		// handled below in switch after EnsureDaemon
	} else if cmd == "daemon" {
		if err := cmdDaemon(args[1:], os.Stdout); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
		return
	}

	// All other commands: auto-start daemon, create client, dispatch.
	socketPath := daemon.DefaultSocketPath()
	if err := daemon.EnsureDaemon(socketPath); err != nil {
		fmt.Fprintf(os.Stderr, "agenthub: %v\n", err)
		os.Exit(1)
	}
	client := daemon.NewDaemonClient(socketPath)

	cmdArgs := args[1:]
	var err error

	switch cmd {
	case "new":
		err = cmdNew(client, cmdArgs, os.Stdout)
	case "list":
		err = cmdList(client, cmdArgs, os.Stdout)
	case "kill":
		err = cmdKill(client, cmdArgs)
	case "rename":
		err = cmdRename(client, cmdArgs)
	case "attach":
		err = cmdAttach(client, cmdArgs)
	case "web":
		err = cmdWeb(client, cmdArgs, os.Stdout)
	case "serve":
		err = cmdServe(client, cmdArgs)
	case "unserve":
		err = cmdUnserve(client, cmdArgs)
	case "health":
		err = cmdHealth(cmdArgs, os.Stdout)
	case "qr":
		err = cmdQR(client, cmdArgs, os.Stdout)
	case "settings":
		err = cmdSettings(client, os.Stdout)
	case "daemon":
		// Only "daemon status" reaches here (others handled above).
		err = cmdDaemonStatus(client, cmdArgs[1:], os.Stdout)
	default:
		fmt.Fprintf(os.Stderr, "agenthub: unknown command %q\nRun 'agenthub --help' for usage.\n", cmd)
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
