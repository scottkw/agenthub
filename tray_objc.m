// tray_objc.m — Objective-C implementation for macOS system tray.
// Compiled once by cgo; keeps ObjC class symbols out of Go's per-file
// compilation units, avoiding duplicate-symbol linker errors.
//
// Build constraint: this file is only compiled on Darwin via cgo tags in
// tray.go (cgo CFLAGS: -x objective-c applies only to .go files with the
// darwin build tag; .m files are always compiled as ObjC by cgo on macOS).

#import <Cocoa/Cocoa.h>

// Forward-declare Go callbacks so they can be invoked from ObjC.
extern void onTrayShow(void);
extern void onTrayQuit(void);
extern void onTraySession(int idx);

NSStatusItem *statusItem = nil;
NSMutableArray *menuSessionNames = nil;
NSMutableArray *menuSessionIDs = nil;
BOOL daemonConnected = YES;

// AgentHubMenuDelegate rebuilds the menu on every open so it always reflects
// the current session list without a separate polling update path.
@interface AgentHubMenuDelegate : NSObject <NSMenuDelegate>
@end

@implementation AgentHubMenuDelegate
- (void)menuWillOpen:(NSMenu *)menu {
    [menu removeAllItems];

    NSMenuItem *openItem = [[NSMenuItem alloc]
        initWithTitle:@"Open AgentHub"
        action:@selector(showClicked:)
        keyEquivalent:@""];
    openItem.target = statusItem;
    [menu addItem:openItem];

    if (menuSessionNames != nil && menuSessionNames.count > 0) {
        [menu addItem:[NSMenuItem separatorItem]];
        for (NSUInteger i = 0; i < menuSessionNames.count; i++) {
            NSMenuItem *item = [[NSMenuItem alloc]
                initWithTitle:menuSessionNames[i]
                action:@selector(sessionClicked:)
                keyEquivalent:@""];
            item.target = statusItem;
            item.tag = (NSInteger)i;
            [menu addItem:item];
        }
    }

    [menu addItem:[NSMenuItem separatorItem]];
    NSMenuItem *quitItem = [[NSMenuItem alloc]
        initWithTitle:@"Quit"
        action:@selector(quitClicked:)
        keyEquivalent:@""];
    quitItem.target = statusItem;
    [menu addItem:quitItem];
}
@end

static AgentHubMenuDelegate *menuDelegate = nil;

// initStatusItem creates a macOS status bar item with a dynamic menu delegate.
void initStatusItem(const void *iconData, int iconLen) {
    // Copy icon data synchronously — iconData is a Go pointer only valid during
    // the cgo call. The dispatch_async block runs later when Go may have reclaimed it.
    NSData *data = [NSData dataWithBytes:iconData length:iconLen];
    dispatch_async(dispatch_get_main_queue(), ^{
        NSStatusBar *bar = [NSStatusBar systemStatusBar];
        statusItem = [bar statusItemWithLength:NSVariableStatusItemLength];
        [statusItem retain];

        // Set icon from PNG data.
        NSImage *icon = [[NSImage alloc] initWithData:data];
        [icon setSize:NSMakeSize(18, 18)];
        [icon setTemplate:YES];  // Adapts to light/dark menu bar.
        statusItem.button.image = icon;
        statusItem.button.toolTip = @"AgentHub";

        // Build menu with delegate for dynamic rebuilding.
        NSMenu *menu = [[NSMenu alloc] init];
        menuDelegate = [[AgentHubMenuDelegate alloc] init];
        menu.delegate = menuDelegate;
        statusItem.menu = menu;

        // Hide Dock icon — Wails overrides LSUIElement by setting its own
        // activation policy, so we must set accessory mode programmatically.
        [NSApp setActivationPolicy:NSApplicationActivationPolicyAccessory];
    });
}

// removeStatusItem tears down the status bar item.
void removeStatusItem(void) {
    dispatch_async(dispatch_get_main_queue(), ^{
        if (statusItem != nil) {
            [[NSStatusBar systemStatusBar] removeStatusItem:statusItem];
            [statusItem release];
            statusItem = nil;
        }
    });
}

// updateTrayIcon swaps the tray icon PNG at runtime (normal vs error state).
void updateTrayIcon(const void *iconData, int iconLen) {
    // Copy before dispatch — iconData is a Go pointer only valid during cgo call.
    NSData *data = [NSData dataWithBytes:iconData length:iconLen];
    dispatch_async(dispatch_get_main_queue(), ^{
        if (statusItem == nil) return;
        NSImage *icon = [[NSImage alloc] initWithData:data];
        [icon setSize:NSMakeSize(18, 18)];
        [icon setTemplate:YES];
        statusItem.button.image = icon;
    });
}

// updateTrayTooltip updates the hover tooltip of the tray icon.
void updateTrayTooltip(const char *tooltip) {
    NSString *tip = [NSString stringWithUTF8String:tooltip];
    dispatch_async(dispatch_get_main_queue(), ^{
        if (statusItem != nil) {
            statusItem.button.toolTip = tip;
        }
    });
}

// setTraySessionData updates the session names/IDs used by the menu delegate.
void setTraySessionData(const char **names, const char **ids, int count) {
    // Copy strings synchronously — names/ids are Go pointers freed after cgo returns.
    NSMutableArray *nameArr = [NSMutableArray arrayWithCapacity:count];
    NSMutableArray *idArr = [NSMutableArray arrayWithCapacity:count];
    for (int i = 0; i < count; i++) {
        [nameArr addObject:[NSString stringWithUTF8String:names[i]]];
        [idArr addObject:[NSString stringWithUTF8String:ids[i]]];
    }
    dispatch_async(dispatch_get_main_queue(), ^{
        menuSessionNames = nameArr;
        menuSessionIDs = idArr;
    });
}

// Action handlers — forward to Go callbacks.
@interface NSStatusItem (AgentHubTray)
- (void)showClicked:(id)sender;
- (void)quitClicked:(id)sender;
- (void)sessionClicked:(NSMenuItem *)sender;
@end

@implementation NSStatusItem (AgentHubTray)
- (void)showClicked:(id)sender { onTrayShow(); }
- (void)quitClicked:(id)sender { onTrayQuit(); }
- (void)sessionClicked:(NSMenuItem *)sender { onTraySession((int)sender.tag); }
@end
