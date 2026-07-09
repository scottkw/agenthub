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
	"context"
	"fmt"
	"os"
	goruntime "runtime"
	"strings"

	"github.com/scottkw/agenthub/internal/daemon"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/menu/keys"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// Version is injected at build time via:
//
//	wails build -ldflags "-X main.Version=v1.9.0"
//
// Falls back to "dev" in local dev builds.
var Version = "dev"

// appCtx holds the Wails runtime context for use in menu callbacks.
// Set in app.startup() after Wails initialises.
var appCtx context.Context

// appInstance holds the App pointer for use in menu callbacks that need
// App methods (not just the Wails runtime context).
var appInstance *App

func main() {
	daemon.BuildVersion = Version
	// Phase 120-04 — hand the embedded React bundle FS to the daemon package so
	// the daemon-mode invocation (`agenthub daemon`) can mount it on the
	// webserver's /app/ route. In dev builds (no wailsassets tag) this
	// returns nil; the webserver answers /app/ with 503. In prod builds
	// (wailsassets) this returns the same fs.FS used by Wails for the
	// desktop bundle, mounted under /app/ for remote/web-share viewers.
	daemon.SetStaticAppFS(staticAppForDaemon())

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
	// Linux/Wayland only: WebKit2GTK's DMABUF GPU renderer hangs on first
	// interaction under Wayland compositors (BUG-05 / #124). Respect any
	// user-supplied value — never overwrite an existing setting.
	if goruntime.GOOS == "linux" {
		if _, ok := os.LookupEnv("WEBKIT_DISABLE_DMABUF_RENDERER"); !ok {
			os.Setenv("WEBKIT_DISABLE_DMABUF_RENDERER", "1")
		}
	}

	daemon.AugmentServicePath() // Ensure CLIs in /usr/local/bin, Homebrew, volta, nvm are on PATH
	app := NewApp()

	err := wails.Run(&options.App{
		Title:             "AgentHub",
		Width:             1200,
		Height:            800,
		MinWidth:          800,
		MinHeight:         600,
		HideWindowOnClose: true,
		StartHidden:       true,
		BackgroundColour:  &options.RGBA{R: 0x1a, G: 0x1b, B: 0x26, A: 0xff},
		Menu:              appMenu(),
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup:     app.startup,
		OnDomReady:    app.domReady,
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

// appMenu constructs the macOS application menu bar.
// Ordering is critical on macOS: AppMenu must be first, then EditMenu, then
// custom submenus. See STATE.md accumulated context for the ordering pitfall.
func appMenu() *menu.Menu {
	m := menu.NewMenu()
	// 1. AppMenu MUST be first on macOS (STATE.md pitfall)
	// darwin-only: Wails' GTK backend on Linux dereferences a nil SubMenu on
	// this role menu and segfaults on launch (BUG-05 / #124).
	if goruntime.GOOS == "darwin" {
		m.Append(menu.AppMenu())
	}
	// 2. File menu (custom — FileMenuRole is commented out in v2.10.2)
	fileMenu := m.AddSubmenu("File")
	fileMenu.AddText("New Session", keys.CmdOrCtrl("n"), nil)
	fileMenu.AddSeparator()
	fileMenu.AddText("Close Tab", keys.CmdOrCtrl("w"), nil)
	// 3. EditMenu — enables Cmd+C/V/X/Z via native NSMenu (MENU-02)
	// darwin-only: same nil-SubMenu segfault as AppMenu (BUG-05 / #124).
	if goruntime.GOOS == "darwin" {
		m.Append(menu.EditMenu())
	}
	// 4. Window menu
	// darwin-only: same nil-SubMenu segfault as AppMenu (BUG-05 / #124).
	if goruntime.GOOS == "darwin" {
		m.Append(menu.WindowMenu())
	}
	// 5. Help menu (custom — HelpSubMenuRole is commented out in v2.10.2)
	helpMenu := m.AddSubmenu("Help")
	helpMenu.AddText("AgentHub on GitHub", nil, openGitHubCallback)
	helpMenu.AddText("Check for Updates", nil, checkForUpdatesCallback)
	return m
}

// openGitHubCallback opens the AgentHub GitHub repository in the default browser.
// Uses the package-level appCtx set during app.startup().
func openGitHubCallback(_ *menu.CallbackData) {
	if appCtx != nil {
		runtime.BrowserOpenURL(appCtx, "https://github.com/scottkw/agenthub")
	}
}

// checkForUpdatesCallback triggers an immediate update check from the Help menu.
// Runs in a goroutine to avoid blocking the UI thread.
func checkForUpdatesCallback(_ *menu.CallbackData) {
	if appInstance != nil {
		go appInstance.CheckForUpdates()
	}
}

// splitDashDash partitions a command-line args slice at the first "--" element.
// Returns (before, nil) if "--" is not present.
// Returns (before, after) where after may be empty if "--" is the last element.
func splitDashDash(args []string) (before, after []string) {
	for i, a := range args {
		if a == "--" {
			return args[:i], args[i+1:]
		}
	}
	return args, nil
}

func runCLI(args []string) {
	before, extraArgs := splitDashDash(args)
	if len(before) == 0 {
		usage()
		return
	}
	cmd := before[0]

	// Daemon sub-command: run daemon mode without EnsureDaemon, or manage service lifecycle.
	// daemon status needs a running daemon — fall through to EnsureDaemon path.
	if cmd == "daemon" && len(before) > 1 && before[1] == "status" {
		// handled below in switch after EnsureDaemon
	} else if cmd == "daemon" {
		if err := cmdDaemon(before[1:], os.Stdout); err != nil {
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

	cmdArgs := before[1:]
	var err error

	switch cmd {
	case "new":
		// Phase 101 SHELL-02: `agenthub new shell ...` routes to cmdNewShell.
		// Any other `new <agent> ...` falls through to the existing cmdNew dispatch.
		if len(cmdArgs) > 0 && cmdArgs[0] == "shell" {
			err = cmdNewShell(client, cmdArgs[1:], extraArgs, os.Stdout)
		} else {
			err = cmdNew(client, cmdArgs, extraArgs, os.Stdout)
		}
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
