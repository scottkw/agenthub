//go:build windows

package main

import (
	"bytes"
	_ "embed"
	"image"
	"image/png"
	"log"
	goruntime "runtime"
	"sync"
	"syscall"
	"unsafe"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/sys/windows"
)

//go:embed assets/tray_icon.png
var trayIconBytes []byte

//go:embed assets/tray_icon_error.png
var trayIconErrorBytes []byte

// Win32 constants for Shell_NotifyIcon and window messaging.
const (
	NIM_ADD          = 0x00000000
	NIM_MODIFY       = 0x00000001
	NIM_DELETE       = 0x00000002
	NIF_ICON         = 0x00000002
	NIF_TIP          = 0x00000004
	NIF_MESSAGE      = 0x00000001
	WM_USER          = 0x0400
	WM_TRAYICON      = WM_USER + 1
	WM_COMMAND       = 0x0111
	WM_RBUTTONUP     = 0x0205
	WM_LBUTTONDBLCLK = 0x0203
	MF_STRING        = 0x00000000
	MF_SEPARATOR     = 0x00000800
	MF_GRAYED        = 0x00000001
	TPM_BOTTOMALIGN  = 0x0020
	TPM_LEFTALIGN    = 0x0000
	IDM_OPEN         = uintptr(1000)
	IDM_QUIT         = uintptr(1001)
	IDM_SESSION      = uintptr(1100)
	// HWND_MESSAGE creates a message-only window (no screen presence).
	HWND_MESSAGE = ^uintptr(2) // (uintptr)(-3)
)

// notifyIconData mirrors NOTIFYICONDATA for Shell_NotifyIconW.
type notifyIconData struct {
	CbSize           uint32
	HWnd             uintptr
	UID              uint32
	UFlags           uint32
	UCallbackMessage uint32
	HIcon            uintptr
	SzTip            [128]uint16
}

// bitmapInfoHeader mirrors BITMAPINFOHEADER for CreateDIBSection.
type bitmapInfoHeader struct {
	BiSize          uint32
	BiWidth         int32
	BiHeight        int32
	BiPlanes        uint16
	BiBitCount      uint16
	BiCompression   uint32
	BiSizeImage     uint32
	BiXPelsPerMeter int32
	BiYPelsPerMeter int32
	BiClrUsed       uint32
	BiClrImportant  uint32
}

// bitmapInfo mirrors BITMAPINFO (header + color table).
type bitmapInfo struct {
	BmiHeader bitmapInfoHeader
	BmiColors [1]uint32
}

// iconInfo mirrors ICONINFO.
type iconInfo struct {
	FIcon    int32
	XHotspot uint32
	YHotspot uint32
	HbmMask  uintptr
	HbmColor uintptr
}

// point mirrors POINT.
type point struct {
	X int32
	Y int32
}

// wndClassEx mirrors WNDCLASSEX.
type wndClassEx struct {
	CbSize        uint32
	Style         uint32
	LpfnWndProc   uintptr
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     uintptr
	HIcon         uintptr
	HCursor       uintptr
	HbrBackground uintptr
	LpszMenuName  *uint16
	LpszClassName *uint16
	HIconSm       uintptr
}

// Lazy-loaded Win32 DLL procedures.
var (
	shell32              = windows.NewLazySystemDLL("shell32.dll")
	user32               = windows.NewLazySystemDLL("user32.dll")
	gdi32                = windows.NewLazySystemDLL("gdi32.dll")
	pShellNotifyIconW    = shell32.NewProc("Shell_NotifyIconW")
	pCreateWindowExW     = user32.NewProc("CreateWindowExW")
	pRegisterClassExW    = user32.NewProc("RegisterClassExW")
	pCreatePopupMenu     = user32.NewProc("CreatePopupMenu")
	pAppendMenuW         = user32.NewProc("AppendMenuW")
	pTrackPopupMenu      = user32.NewProc("TrackPopupMenu")
	pGetCursorPos        = user32.NewProc("GetCursorPos")
	pSetForegroundWindow = user32.NewProc("SetForegroundWindow")
	pDestroyMenu         = user32.NewProc("DestroyMenu")
	pDestroyWindow       = user32.NewProc("DestroyWindow")
	pDestroyIcon         = user32.NewProc("DestroyIcon")
	pDefWindowProcW      = user32.NewProc("DefWindowProcW")
	pGetMessageW         = user32.NewProc("GetMessageW")
	pTranslateMessage    = user32.NewProc("TranslateMessage")
	pDispatchMessageW    = user32.NewProc("DispatchMessageW")
	pPostMessageW        = user32.NewProc("PostMessageW")
	pGetModuleHandleW    = user32.NewProc("GetModuleHandleW")
	pCreateIconIndirect  = user32.NewProc("CreateIconIndirect")
	pCreateDIBSection    = gdi32.NewProc("CreateDIBSection")
	pCreateBitmap        = gdi32.NewProc("CreateBitmap")
	pDeleteObject        = gdi32.NewProc("DeleteObject")
)

// windowsTray holds all state for the Windows system tray icon.
type windowsTray struct {
	hwnd      uintptr
	hIcon     uintptr
	hIconErr  uintptr
	nid       notifyIconData
	menuItems []MenuItem
	mu        sync.Mutex
	app       *App
	ready     chan struct{}
	disabled  bool
}

// globalWinTray is the package-level reference used by the wndProc callback.
// Win32 window procedure callbacks cannot capture closures.
var globalWinTray *windowsTray

// createIconFromPNG converts PNG bytes to a Win32 HICON handle using GDI APIs.
// Returns (0, error) on failure. Caller is responsible for calling DestroyIcon
// on the returned handle when done.
func createIconFromPNG(pngData []byte) (uintptr, error) {
	img, err := png.Decode(bytes.NewReader(pngData))
	if err != nil {
		return 0, err
	}
	bounds := img.Bounds()
	width := bounds.Max.X - bounds.Min.X
	height := bounds.Max.Y - bounds.Min.Y

	// Build BITMAPINFO for a 32-bit top-down DIB (negative height = top-down).
	bi := bitmapInfo{}
	bi.BmiHeader.BiSize = uint32(unsafe.Sizeof(bi.BmiHeader))
	bi.BmiHeader.BiWidth = int32(width)
	bi.BmiHeader.BiHeight = -int32(height) // negative = top-down
	bi.BmiHeader.BiPlanes = 1
	bi.BmiHeader.BiBitCount = 32
	bi.BmiHeader.BiCompression = 0 // BI_RGB

	// CreateDIBSection: create device-independent bitmap + pointer to pixel data.
	var ppvBits unsafe.Pointer
	hColorBmp, _, _ := pCreateDIBSection.Call(
		0, // hdc = NULL (use default)
		uintptr(unsafe.Pointer(&bi)),
		0, // DIB_RGB_COLORS
		uintptr(unsafe.Pointer(&ppvBits)),
		0, 0,
	)
	if hColorBmp == 0 {
		return 0, syscall.EINVAL
	}
	defer func() {
		if hColorBmp != 0 {
			pDeleteObject.Call(hColorBmp)
		}
	}()

	// Copy pixels from image.Image into the DIB memory as BGRA.
	pixelBytes := (*[1 << 30]byte)(ppvBits)
	nrgbaImg, isNRGBA := img.(*image.NRGBA)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			offset := (y*width + x) * 4
			var r, g, b, a uint8
			if isNRGBA {
				// Fast path for NRGBA images (most PNGs).
				pix := nrgbaImg.NRGBAAt(x+bounds.Min.X, y+bounds.Min.Y)
				r, g, b, a = pix.R, pix.G, pix.B, pix.A
			} else {
				// Generic path via RGBA() (returns 16-bit values).
				rr, gg, bb, aa := img.At(x+bounds.Min.X, y+bounds.Min.Y).RGBA()
				r = byte(rr >> 8)
				g = byte(gg >> 8)
				b = byte(bb >> 8)
				a = byte(aa >> 8)
			}
			// Win32 DIB pixel order is BGRA.
			pixelBytes[offset+0] = b
			pixelBytes[offset+1] = g
			pixelBytes[offset+2] = r
			pixelBytes[offset+3] = a
		}
	}

	// Create a monochrome mask bitmap (all zeros = fully opaque).
	hMaskBmp, _, _ := pCreateBitmap.Call(
		uintptr(width),
		uintptr(height),
		1,  // planes
		1,  // bits per pixel (monochrome)
		0,  // no pixel data (zeros = opaque)
	)
	if hMaskBmp == 0 {
		return 0, syscall.EINVAL
	}
	defer pDeleteObject.Call(hMaskBmp)

	// Build ICONINFO and call CreateIconIndirect.
	ii := iconInfo{
		FIcon:    1, // TRUE = icon (not cursor)
		HbmMask:  hMaskBmp,
		HbmColor: hColorBmp,
	}
	hIcon, _, _ := pCreateIconIndirect.Call(uintptr(unsafe.Pointer(&ii)))
	if hIcon == 0 {
		return 0, syscall.EINVAL
	}

	// Prevent the deferred DeleteObject from destroying hColorBmp since
	// CreateIconIndirect has taken ownership of the bitmap.
	hColorBmp = 0
	return hIcon, nil
}

// menuIDForItem maps a MenuItem to its Win32 menu command ID.
// Separators return 0. This is exported for testability.
func menuIDForItem(item MenuItem) uintptr {
	if item.Kind == MenuItemSeparator {
		return 0
	}
	switch item.Label {
	case "Open AgentHub":
		return IDM_OPEN
	case "Quit":
		return IDM_QUIT
	default:
		// Session item — use the session index.
		return IDM_SESSION + uintptr(item.Index)
	}
}

// wndProc is the Win32 window procedure for the message-only tray window.
// Must be a package-level function — Win32 callbacks cannot capture closures.
func wndProc(hwnd, msg, wParam, lParam uintptr) uintptr {
	wt := globalWinTray
	if wt == nil {
		ret, _, _ := pDefWindowProcW.Call(hwnd, msg, wParam, lParam)
		return ret
	}

	switch msg {
	case WM_TRAYICON:
		loWord := lParam & 0xFFFF
		switch loWord {
		case WM_RBUTTONUP:
			wt.showPopupMenu()
		case WM_LBUTTONDBLCLK:
			go func() {
				if wt.app != nil && wt.app.ctx != nil {
					runtime.WindowShow(wt.app.ctx)
					wt.app.setDockVisible(true)
				}
			}()
		}
		return 0

	case WM_COMMAND:
		menuID := wParam & 0xFFFF
		switch menuID {
		case IDM_OPEN:
			go func() {
				if wt.app != nil && wt.app.ctx != nil {
					runtime.WindowShow(wt.app.ctx)
					wt.app.setDockVisible(true)
				}
			}()
		case IDM_QUIT:
			go func() {
				app := wt.app
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
		default:
			// T-67-06: validate session index bounds before dispatching.
			if menuID >= IDM_SESSION {
				idx := int(menuID - IDM_SESSION)
				go func() {
					if wt.app == nil || wt.app.ctx == nil {
						return
					}
					sessions := wt.app.ListSessions()
					if idx >= 0 && idx < len(sessions) {
						runtime.WindowShow(wt.app.ctx)
						wt.app.setDockVisible(true)
						runtime.EventsEmit(wt.app.ctx, "tray:focus-session", sessions[idx].ID)
					}
				}()
			}
		}
		return 0
	}

	ret, _, _ := pDefWindowProcW.Call(hwnd, msg, wParam, lParam)
	return ret
}

// showPopupMenu creates and displays the tray context menu at the cursor position.
func (wt *windowsTray) showPopupMenu() {
	wt.mu.Lock()
	items := make([]MenuItem, len(wt.menuItems))
	copy(items, wt.menuItems)
	wt.mu.Unlock()

	hmenu, _, _ := pCreatePopupMenu.Call()
	if hmenu == 0 {
		return
	}
	defer pDestroyMenu.Call(hmenu)

	for _, item := range items {
		if item.Kind == MenuItemSeparator {
			pAppendMenuW.Call(hmenu, MF_SEPARATOR, 0, 0)
			continue
		}
		id := menuIDForItem(item)
		label, _ := syscall.UTF16PtrFromString(item.Label)
		pAppendMenuW.Call(hmenu, MF_STRING, id, uintptr(unsafe.Pointer(label)))
	}

	var pt point
	pGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	pSetForegroundWindow.Call(wt.hwnd)
	pTrackPopupMenu.Call(
		hmenu,
		TPM_BOTTOMALIGN|TPM_LEFTALIGN,
		uintptr(pt.X),
		uintptr(pt.Y),
		0,
		wt.hwnd,
		0,
	)
}

// initTray creates the Windows system tray icon via Shell_NotifyIconW.
// It launches a goroutine that owns the Win32 message pump.
func (a *App) initTray() {
	wt := &windowsTray{
		app:   a,
		ready: make(chan struct{}),
	}
	globalWinTray = wt

	// Convert PNG assets to HICON handles.
	var err error
	wt.hIcon, err = createIconFromPNG(trayIconBytes)
	if err != nil {
		log.Printf("system tray: failed to create normal icon: %v; tray icon disabled", err)
		wt.disabled = true
		close(wt.ready)
		return
	}
	wt.hIconErr, err = createIconFromPNG(trayIconErrorBytes)
	if err != nil {
		log.Printf("system tray: failed to create error icon: %v; tray icon disabled", err)
		pDestroyIcon.Call(wt.hIcon)
		wt.disabled = true
		close(wt.ready)
		return
	}

	// Populate initial menu items (no sessions yet).
	wt.menuItems = BuildMenuItems(nil)

	go func() {
		// Win32 message pump MUST run on a locked OS thread.
		goruntime.LockOSThread()
		defer goruntime.UnlockOSThread()

		// Get module handle (hInstance).
		hInstance, _, _ := pGetModuleHandleW.Call(0)

		// Register window class.
		className, _ := syscall.UTF16PtrFromString("AgentHubTrayClass")
		wndClass := wndClassEx{
			LpfnWndProc:   syscall.NewCallback(wndProc),
			HInstance:     hInstance,
			LpszClassName: className,
		}
		wndClass.CbSize = uint32(unsafe.Sizeof(wndClass))
		pRegisterClassExW.Call(uintptr(unsafe.Pointer(&wndClass)))

		// Create message-only window (no screen presence).
		windowName, _ := syscall.UTF16PtrFromString("AgentHub Tray")
		hwnd, _, _ := pCreateWindowExW.Call(
			0,            // dwExStyle
			uintptr(unsafe.Pointer(className)),
			uintptr(unsafe.Pointer(windowName)),
			0,            // dwStyle
			0, 0, 0, 0,  // x, y, width, height
			HWND_MESSAGE, // message-only parent
			0, hInstance, 0,
		)
		if hwnd == 0 {
			log.Printf("system tray: Shell_NotifyIcon failed, tray icon disabled")
			wt.disabled = true
			close(wt.ready)
			return
		}
		wt.hwnd = hwnd

		// Build the NOTIFYICONDATA struct.
		nid := notifyIconData{
			CbSize:           uint32(unsafe.Sizeof(notifyIconData{})),
			HWnd:             hwnd,
			UID:              1,
			UFlags:           NIF_ICON | NIF_TIP | NIF_MESSAGE,
			UCallbackMessage: WM_TRAYICON,
			HIcon:            wt.hIcon,
		}
		// Set initial tooltip.
		tip := trayTooltip(0)
		tipUTF16 := syscall.StringToUTF16(tip)
		copyLen := len(tipUTF16)
		if copyLen > len(nid.SzTip) {
			copyLen = len(nid.SzTip)
		}
		copy(nid.SzTip[:], tipUTF16[:copyLen])
		wt.nid = nid

		// Add the tray icon.
		ret, _, _ := pShellNotifyIconW.Call(NIM_ADD, uintptr(unsafe.Pointer(&wt.nid)))
		if ret == 0 {
			log.Printf("system tray: Shell_NotifyIcon failed, tray icon disabled")
			wt.disabled = true
			pDestroyWindow.Call(hwnd)
			close(wt.ready)
			return
		}

		// Signal that tray is ready.
		close(wt.ready)

		// Win32 message loop.
		var msg struct {
			Hwnd    uintptr
			Message uint32
			WParam  uintptr
			LParam  uintptr
			Time    uint32
			Pt      point
		}
		for {
			ret, _, _ := pGetMessageW.Call(
				uintptr(unsafe.Pointer(&msg)),
				0, 0, 0,
			)
			if ret == 0 {
				// WM_QUIT received.
				break
			}
			pTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
			pDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
		}
	}()

	// Wait for the message pump to be ready.
	<-wt.ready
}

// updateTray updates the tray icon and tooltip based on current session state.
func (a *App) updateTray(sessions []SessionInfo, connected bool) {
	wt := globalWinTray
	if wt == nil || wt.disabled {
		return
	}

	wt.mu.Lock()
	defer wt.mu.Unlock()

	// Switch icon based on connectivity.
	if connected {
		wt.nid.HIcon = wt.hIcon
	} else {
		wt.nid.HIcon = wt.hIconErr
	}

	// Update tooltip.
	tip := trayTooltip(len(sessions))
	tipUTF16 := syscall.StringToUTF16(tip)
	var tipArr [128]uint16
	copyLen := len(tipUTF16)
	if copyLen > len(tipArr) {
		copyLen = len(tipArr)
	}
	copy(tipArr[:], tipUTF16[:copyLen])
	wt.nid.SzTip = tipArr

	pShellNotifyIconW.Call(NIM_MODIFY, uintptr(unsafe.Pointer(&wt.nid)))

	// Rebuild menu items.
	wt.menuItems = BuildMenuItems(sessions)
}

// cleanupTray removes the tray icon and exits the message loop.
func (a *App) cleanupTray() {
	wt := globalWinTray
	if wt == nil || wt.disabled {
		return
	}

	pShellNotifyIconW.Call(NIM_DELETE, uintptr(unsafe.Pointer(&wt.nid)))
	pDestroyIcon.Call(wt.hIcon)
	pDestroyIcon.Call(wt.hIconErr)
	// Post WM_QUIT to exit the message loop goroutine.
	pPostMessageW.Call(wt.hwnd, 0x0012 /* WM_QUIT */, 0, 0)
	pDestroyWindow.Call(wt.hwnd)
}

// setDockVisible is a no-op on Windows — Wails manages taskbar visibility.
func (a *App) setDockVisible(visible bool) {
	// Windows has no Dock equivalent; Wails manages taskbar visibility.
}
