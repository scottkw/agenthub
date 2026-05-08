//go:build linux

package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"image/png"
	"log"
	"os"
	"sync"

	"github.com/godbus/dbus/v5"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed assets/tray_icon.png
var trayIconBytes []byte

//go:embed assets/tray_icon_error.png
var trayIconErrorBytes []byte

//go:embed assets/tray_icon_progress_25.png
var trayIconProgress25Bytes []byte

//go:embed assets/tray_icon_progress_50.png
var trayIconProgress50Bytes []byte

//go:embed assets/tray_icon_progress_75.png
var trayIconProgress75Bytes []byte

//go:embed assets/tray_icon_progress_100.png
var trayIconProgress100Bytes []byte

// dbusMenuItem is a single entry in the D-Bus menu tree, with sequential ID
// and a string property map for dbusmenu protocol.
type dbusMenuItem struct {
	id       int32
	props    map[string]string
	children []*dbusMenuItem
}

// dbusMenuNode is the root of the D-Bus menu layout returned from buildDbusMenuLayout.
type dbusMenuNode struct {
	id       int32
	props    map[string]string
	children []*dbusMenuItem
}

// linuxTray holds all D-Bus state for the Linux system tray.
type linuxTray struct {
	conn       *dbus.Conn
	busName    string
	iconPixmap [][3]interface{} // []{ int32 width, int32 height, []byte ARGB32 }
	errPixmap  [][3]interface{}
	menuItems  []MenuItem // from tray_common.go BuildMenuItems
	menuRev    uint32     // incremented on every menu rebuild
	mu         sync.Mutex
	app        *App
	disabled   bool // true if D-Bus not available; all ops become no-ops
}

// pngToARGB32Pixmap decodes a PNG image to ARGB32 pixel data for D-Bus IconPixmap.
// Returns width, height (int32) and big-endian ARGB32 pixel bytes.
// The D-Bus IconPixmap property type is a(iiay).
func pngToARGB32Pixmap(data []byte) (width, height int32, pixels []byte, err error) {
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return 0, 0, nil, fmt.Errorf("pngToARGB32Pixmap: decode: %w", err)
	}
	bounds := img.Bounds()
	w := bounds.Max.X - bounds.Min.X
	h := bounds.Max.Y - bounds.Min.Y
	pixels = make([]byte, w*h*4)
	idx := 0
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			// Convert from 16-bit to 8-bit and write as ARGB (big-endian).
			pixels[idx] = byte(a >> 8)
			pixels[idx+1] = byte(r >> 8)
			pixels[idx+2] = byte(g >> 8)
			pixels[idx+3] = byte(b >> 8)
			idx += 4
		}
	}
	return int32(w), int32(h), pixels, nil
}

// makePixmap converts a PNG byte slice into the D-Bus a(iiay) pixmap struct.
func makePixmap(data []byte) [][3]interface{} {
	w, h, pixels, err := pngToARGB32Pixmap(data)
	if err != nil {
		return nil
	}
	return [][3]interface{}{{w, h, pixels}}
}

// buildDbusMenuLayout builds a dbusMenuNode from a slice of MenuItem values.
// Items are assigned sequential IDs starting from 1.
func buildDbusMenuLayout(items []MenuItem) *dbusMenuNode {
	root := &dbusMenuNode{
		id:    0,
		props: map[string]string{"children-display": "submenu"},
	}
	for i, item := range items {
		child := &dbusMenuItem{
			id: int32(i + 1),
		}
		switch item.Kind {
		case MenuItemSeparator:
			child.props = map[string]string{"type": "separator"}
		default:
			child.props = map[string]string{
				"label":   item.Label,
				"type":    "standard",
				"enabled": "true",
			}
		}
		root.children = append(root.children, child)
	}
	return root
}

// getLayout returns the dbusmenu revision and layout variant for the current menu state.
func (t *linuxTray) getLayout() (uint32, dbus.Variant) {
	t.mu.Lock()
	items := t.menuItems
	rev := t.menuRev
	t.mu.Unlock()

	root := buildDbusMenuLayout(items)

	// Build the D-Bus variant for the root layout item.
	// Each entry is (id int32, props map[string]dbus.Variant, children []dbus.Variant).
	type dbusMenuEntry struct {
		ID       int32
		Props    map[string]dbus.Variant
		Children []dbus.Variant
	}

	makeEntry := func(id int32, props map[string]string) dbusMenuEntry {
		p := make(map[string]dbus.Variant, len(props))
		for k, v := range props {
			p[k] = dbus.MakeVariant(v)
		}
		return dbusMenuEntry{ID: id, Props: p}
	}

	var children []dbus.Variant
	for _, child := range root.children {
		e := makeEntry(child.id, child.props)
		children = append(children, dbus.MakeVariant(e))
	}

	rootEntry := dbusMenuEntry{
		ID:       0,
		Props:    map[string]dbus.Variant{"children-display": dbus.MakeVariant("submenu")},
		Children: children,
	}

	return rev, dbus.MakeVariant(rootEntry)
}

// StatusNotifierItemExporter exports the org.kde.StatusNotifierItem D-Bus interface.
type StatusNotifierItemExporter struct {
	tray *linuxTray
}

// Activate is called when the user left-clicks the tray icon.
func (e *StatusNotifierItemExporter) Activate(x, y int32) *dbus.Error {
	e.tray.onShow()
	return nil
}

// ContextMenu is a no-op — menu is handled by dbusmenu.
func (e *StatusNotifierItemExporter) ContextMenu(x, y int32) *dbus.Error {
	return nil
}

// SecondaryActivate is a no-op.
func (e *StatusNotifierItemExporter) SecondaryActivate(x, y int32) *dbus.Error {
	return nil
}

// Scroll is a no-op.
func (e *StatusNotifierItemExporter) Scroll(delta int32, orientation string) *dbus.Error {
	return nil
}

// DbusMenuExporter exports the com.canonical.dbusmenu D-Bus interface.
type DbusMenuExporter struct {
	tray *linuxTray
}

// GetLayout returns the full menu layout.
func (e *DbusMenuExporter) GetLayout(parentId int32, recursionDepth int32, propertyNames []string) (uint32, dbus.Variant, *dbus.Error) {
	rev, layout := e.tray.getLayout()
	return rev, layout, nil
}

// Event handles menu item click events from the tray host.
// Validates item ID bounds to prevent out-of-bounds session index access (T-67-02).
func (e *DbusMenuExporter) Event(id int32, eventId string, data dbus.Variant, timestamp uint32) *dbus.Error {
	if eventId != "clicked" {
		return nil
	}
	e.tray.mu.Lock()
	items := e.tray.menuItems
	e.tray.mu.Unlock()

	// Validate ID bounds: item IDs are 1-based (T-67-02: prevents out-of-bounds access).
	if id < 1 || int(id) > len(items) {
		return nil
	}
	item := items[id-1]

	switch item.Kind {
	case MenuItemSeparator:
		// no-op
	case MenuItemAction:
		switch item.Label {
		case "Open AgentHub":
			e.tray.onShow()
		case "Quit":
			e.tray.onQuit()
		default:
			if item.SessionID != "" {
				e.tray.onSession(item.Index)
			}
		}
	}
	return nil
}

// GetGroupProperties returns properties for the requested item IDs.
func (e *DbusMenuExporter) GetGroupProperties(ids []int32, propertyNames []string) ([]dbus.Variant, *dbus.Error) {
	e.tray.mu.Lock()
	items := e.tray.menuItems
	e.tray.mu.Unlock()

	var result []dbus.Variant
	for _, id := range ids {
		if id < 1 || int(id) > len(items) {
			continue
		}
		item := items[id-1]
		props := map[string]dbus.Variant{"id": dbus.MakeVariant(id)}
		if item.Kind == MenuItemSeparator {
			props["type"] = dbus.MakeVariant("separator")
		} else {
			props["label"] = dbus.MakeVariant(item.Label)
		}
		result = append(result, dbus.MakeVariant(props))
	}
	return result, nil
}

// AboutToShow returns false (no dynamic submenu needed).
func (e *DbusMenuExporter) AboutToShow(id int32) (bool, *dbus.Error) {
	return false, nil
}

// AboutToShowGroup is a batch variant of AboutToShow.
func (e *DbusMenuExporter) AboutToShowGroup(ids []int32) ([]int32, []int32, *dbus.Error) {
	return nil, nil, nil
}

// EventGroup handles batch click events.
func (e *DbusMenuExporter) EventGroup(events []struct {
	ID        int32
	EventID   string
	Data      dbus.Variant
	Timestamp uint32
}) ([]int32, *dbus.Error) {
	for _, ev := range events {
		_ = e.Event(ev.ID, ev.EventID, ev.Data, ev.Timestamp)
	}
	return nil, nil
}

// onShow shows the application window (mirrors macOS onTrayShow callback).
func (t *linuxTray) onShow() {
	app := t.app
	go func() {
		if app != nil && app.ctx != nil {
			runtime.WindowShow(app.ctx)
			app.setDockVisible(true)
		}
	}()
}

// onQuit shuts down the daemon and exits (mirrors macOS onTrayQuit callback).
func (t *linuxTray) onQuit() {
	app := t.app
	go func() {
		if app != nil && app.client != nil {
			_ = app.client.ShutdownDaemon()
		}
		if app != nil {
			app.quitting = true
			if app.ctx != nil {
				runtime.Quit(app.ctx)
			}
		}
	}()
}

// onSession focuses the session at the given index (mirrors macOS onTraySession callback).
func (t *linuxTray) onSession(idx int) {
	app := t.app
	go func() {
		if app == nil || app.ctx == nil {
			return
		}
		sessions := app.ListSessions()
		if idx >= 0 && idx < len(sessions) {
			runtime.WindowShow(app.ctx)
			app.setDockVisible(true)
			runtime.EventsEmit(app.ctx, "tray:focus-session", sessions[idx].ID)
		}
	}()
}

// linuxTrayInstance is the global linuxTray (set by initTray, used by updateTray/cleanupTray).
var linuxTrayInstance *linuxTray

// initTray creates a Linux system tray icon using D-Bus StatusNotifierItem.
// If the D-Bus session bus or StatusNotifierWatcher is unavailable, the tray
// gracefully degrades to a no-op (logs a warning, sets disabled=true).
func (a *App) initTray() {
	tray := &linuxTray{app: a}
	linuxTrayInstance = tray

	// Pre-decode icon pixmaps.
	tray.iconPixmap = makePixmap(trayIconBytes)
	tray.errPixmap = makePixmap(trayIconErrorBytes)
	tray.menuItems = BuildMenuItems(nil)

	// Connect to the session D-Bus.
	conn, err := dbus.SessionBusPrivate()
	if err != nil {
		log.Printf("system tray: D-Bus session bus unavailable: %v", err)
		tray.disabled = true
		return
	}
	if err := conn.Auth(nil); err != nil {
		conn.Close()
		log.Printf("system tray: D-Bus auth failed: %v", err)
		tray.disabled = true
		return
	}
	if err := conn.Hello(); err != nil {
		conn.Close()
		log.Printf("system tray: D-Bus Hello failed: %v", err)
		tray.disabled = true
		return
	}
	tray.conn = conn

	// Request a unique bus name for the StatusNotifierItem.
	pid := os.Getpid()
	busName := fmt.Sprintf("org.kde.StatusNotifierItem-%d-1", pid)
	reply, err := conn.RequestName(busName, dbus.NameFlagDoNotQueue)
	if err != nil || reply != dbus.RequestNameReplyPrimaryOwner {
		log.Printf("system tray: could not acquire D-Bus name %s: %v", busName, err)
		tray.disabled = true
		return
	}
	tray.busName = busName

	// Export StatusNotifierItem object at /StatusNotifierItem.
	sniExporter := &StatusNotifierItemExporter{tray: tray}
	if err := conn.ExportAll(sniExporter, "/StatusNotifierItem", "org.kde.StatusNotifierItem"); err != nil {
		log.Printf("system tray: failed to export StatusNotifierItem: %v", err)
		tray.disabled = true
		return
	}

	// Export dbusmenu object at /MenuBar.
	menuExporter := &DbusMenuExporter{tray: tray}
	if err := conn.ExportAll(menuExporter, "/MenuBar", "com.canonical.dbusmenu"); err != nil {
		log.Printf("system tray: failed to export dbusmenu: %v", err)
		tray.disabled = true
		return
	}

	// Register with StatusNotifierWatcher.
	watcher := conn.Object("org.kde.StatusNotifierWatcher", "/StatusNotifierWatcher")
	call := watcher.Call("org.kde.StatusNotifierWatcher.RegisterStatusNotifierItem", 0, busName)
	if call.Err != nil {
		log.Printf("system tray: StatusNotifierWatcher not available, tray icon disabled: %v", call.Err)
		// Do not set disabled=true — the SNI service is still exported and some
		// tray hosts auto-discover services without the watcher.
	}
}

// cleanupTray closes the D-Bus connection and removes the tray icon.
func (a *App) cleanupTray() {
	if linuxTrayInstance == nil {
		return
	}
	tray := linuxTrayInstance
	tray.mu.Lock()
	conn := tray.conn
	tray.conn = nil
	tray.mu.Unlock()
	if conn != nil {
		conn.Close()
	}
}

// trayIconBytesForState returns the appropriate tray icon byte slice for the
// given connection state and current progress quartile (Phase 98 PRG-03).
//
// Error precedence (Pitfall #8): when connected=false, always returns
// trayIconErrorBytes regardless of a.lastTrayQuartile to ensure daemon-
// disconnect is not masked by a progress glyph.
//
// This helper is defined verbatim in tray.go (darwin), tray_linux.go, and
// tray_windows.go — three identical copies required because each file has its
// own //go:build tag and the trayIconProgress* byte slices embedded in each
// file are not visible across build-tag boundaries.
func (a *App) trayIconBytesForState(connected bool) []byte {
	if !connected {
		return trayIconErrorBytes
	}
	switch a.lastTrayQuartile {
	case 1:
		return trayIconProgress25Bytes
	case 2:
		return trayIconProgress50Bytes
	case 3:
		return trayIconProgress75Bytes
	case 4:
		return trayIconProgress100Bytes
	default:
		return trayIconBytes
	}
}

// updateTray updates the tray icon state and menu on Linux.
// Called from refreshTrayState() every 5 seconds via startTrayPoller.
func (a *App) updateTray(sessions []SessionInfo, connected bool) {
	if linuxTrayInstance == nil || linuxTrayInstance.disabled {
		return
	}
	tray := linuxTrayInstance

	tray.mu.Lock()
	// Update icon pixmap based on connection state and progress quartile.
	bytes := a.trayIconBytesForState(connected)
	tray.iconPixmap = makePixmap(bytes)
	// Rebuild menu from current sessions.
	tray.menuItems = BuildMenuItems(sessions)
	tray.menuRev++
	rev := tray.menuRev
	conn := tray.conn
	tray.mu.Unlock()

	if conn == nil {
		return
	}

	// Emit NewIcon signal.
	_ = conn.Emit("/StatusNotifierItem", "org.kde.StatusNotifierItem.NewIcon")
	// Emit NewToolTip signal.
	_ = conn.Emit("/StatusNotifierItem", "org.kde.StatusNotifierItem.NewToolTip")
	// Emit LayoutUpdated signal for dbusmenu.
	_ = conn.Emit("/MenuBar", "com.canonical.dbusmenu.LayoutUpdated", rev, int32(0))
}

// setDockVisible is a no-op on Linux.
// Linux has no Dock equivalent; Wails manages taskbar visibility.
func (a *App) setDockVisible(visible bool) {}
