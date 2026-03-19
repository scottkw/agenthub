//go:build darwin

package main

/*
#cgo darwin CFLAGS: -x objective-c
#cgo darwin LDFLAGS: -framework Cocoa

#import <Cocoa/Cocoa.h>

// Callback declarations — implemented in Go via //export.
extern void onTrayShow(void);
extern void onTrayQuit(void);

static NSStatusItem *statusItem = nil;

// initStatusItem creates a macOS status bar item with a menu.
// Must be called on the main thread (via dispatch_async if needed).
static void initStatusItem(const void *iconData, int iconLen) {
    dispatch_async(dispatch_get_main_queue(), ^{
        NSStatusBar *bar = [NSStatusBar systemStatusBar];
        statusItem = [bar statusItemWithLength:NSVariableStatusItemLength];
        [statusItem retain];

        // Set icon from PNG data.
        NSData *data = [NSData dataWithBytes:iconData length:iconLen];
        NSImage *icon = [[NSImage alloc] initWithData:data];
        [icon setSize:NSMakeSize(18, 18)];
        [icon setTemplate:YES];  // Adapts to light/dark menu bar.
        statusItem.button.image = icon;
        statusItem.button.toolTip = @"AgentHub";

        // Build menu.
        NSMenu *menu = [[NSMenu alloc] init];

        NSMenuItem *showItem = [[NSMenuItem alloc]
            initWithTitle:@"Show AgentHub"
            action:@selector(showClicked:)
            keyEquivalent:@""];
        showItem.target = statusItem;
        [menu addItem:showItem];

        [menu addItem:[NSMenuItem separatorItem]];

        NSMenuItem *quitItem = [[NSMenuItem alloc]
            initWithTitle:@"Quit"
            action:@selector(quitClicked:)
            keyEquivalent:@""];
        quitItem.target = statusItem;
        [menu addItem:quitItem];

        statusItem.menu = menu;
    });
}

// removeStatusItem tears down the status bar item.
static void removeStatusItem(void) {
    dispatch_async(dispatch_get_main_queue(), ^{
        if (statusItem != nil) {
            [[NSStatusBar systemStatusBar] removeStatusItem:statusItem];
            [statusItem release];
            statusItem = nil;
        }
    });
}

// Action handlers — forward to Go callbacks.
@interface NSStatusItem (AgentHubTray)
- (void)showClicked:(id)sender;
- (void)quitClicked:(id)sender;
@end

@implementation NSStatusItem (AgentHubTray)
- (void)showClicked:(id)sender { onTrayShow(); }
- (void)quitClicked:(id)sender { onTrayQuit(); }
@end
*/
import "C"

import (
	_ "embed"
	"unsafe"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed assets/appicon.png
var trayIconBytes []byte

// trayCallbackApp is the global App reference for cgo callbacks.
// Set before initTray returns. Only accessed from the main goroutine.
var trayCallbackApp *App

//export onTrayShow
func onTrayShow() {
	if trayCallbackApp != nil && trayCallbackApp.ctx != nil {
		runtime.WindowShow(trayCallbackApp.ctx)
	}
}

//export onTrayQuit
func onTrayQuit() {
	if trayCallbackApp != nil && trayCallbackApp.ctx != nil {
		runtime.Quit(trayCallbackApp.ctx)
	}
}

// initTray creates a macOS system tray icon with Show/Quit menu items.
func (a *App) initTray() {
	trayCallbackApp = a
	ptr := unsafe.Pointer(&trayIconBytes[0])
	C.initStatusItem(ptr, C.int(len(trayIconBytes)))
}

// cleanupTray removes the status bar item.
func (a *App) cleanupTray() {
	C.removeStatusItem()
}
