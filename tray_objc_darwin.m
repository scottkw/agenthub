// tray_objc.m — Objective-C implementation for macOS system tray.
// Compiled once by cgo; keeps ObjC class symbols out of Go's per-file
// compilation units, avoiding duplicate-symbol linker errors.
//
// Build constraint: this file is only compiled on Darwin via cgo tags in
// tray.go (cgo CFLAGS: -x objective-c applies only to .go files with the
// darwin build tag; .m files are always compiled as ObjC by cgo on macOS).

#import <Cocoa/Cocoa.h>
#import <UserNotifications/UserNotifications.h>

// Forward-declare Go callbacks so they can be invoked from ObjC.
extern void onTrayShow(void);
extern void onTrayQuit(void);
extern void onTraySession(int idx);
// onNotificationAuthResult reports the outcome of a proactive
// UNUserNotificationCenter authorization request back to Go (Phase 167-06,
// M-41 gap closure) so the frontend can be told when the user denies
// permission (notification_darwin.go //export onNotificationAuthResult).
extern void onNotificationAuthResult(int granted);

// Forward-declare hasValidBundleIdentifier (defined below, near sendNotification)
// so the delegate-registration/authorization helpers declared earlier in this
// file (which run before its definition point) can call it.
int hasValidBundleIdentifier(void);

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

// AgentHubNotificationDelegate presents notifications while AgentHub is the
// frontmost app (Phase 167-06, M-41 gap closure). Without a registered
// UNUserNotificationCenterDelegate, UNUserNotificationCenter suppresses
// banners/sounds for the foreground app by default, silently swallowing the
// notification even when authorization was granted and delivery succeeded.
@interface AgentHubNotificationDelegate : NSObject <UNUserNotificationCenterDelegate>
@end

@implementation AgentHubNotificationDelegate
- (void)userNotificationCenter:(UNUserNotificationCenter *)center
        willPresentNotification:(UNNotification *)notification
          withCompletionHandler:(void (^)(UNNotificationPresentationOptions))completionHandler {
    completionHandler(UNNotificationPresentationOptionBanner |
                       UNNotificationPresentationOptionList |
                       UNNotificationPresentationOptionSound);
}
@end

static AgentHubNotificationDelegate *notificationDelegate = nil;

// registerNotificationDelegate installs the foreground-presentation delegate
// on UNUserNotificationCenter. Idempotent (safe to call repeatedly) and
// bundle-guarded — mirrors the 167-05 fail-safe pattern so an unbundled
// process (wails dev / go test) log-and-swallows instead of raising an
// uncaught NSInternalInconsistencyException.
void registerNotificationDelegate(void) {
    if (!hasValidBundleIdentifier()) {
        NSLog(@"AgentHub: skipping notification delegate registration — no valid app-bundle identifier (unbundled/unsigned process)");
        return;
    }
    @try {
        if (notificationDelegate == nil) {
            notificationDelegate = [[AgentHubNotificationDelegate alloc] init];
        }
        [UNUserNotificationCenter currentNotificationCenter].delegate = notificationDelegate;
    } @catch (NSException *e) {
        NSLog(@"AgentHub: swallowed exception registering notification delegate: %@", e);
    }
}

// requestNotificationAuthorization proactively surfaces the macOS
// notification-permission prompt (Phase 167-06, M-41 gap closure — the
// leading suspected fix: during UAT the app showed as "Off" in System
// Settings with NO prompt ever seen). Called from Go when the user enables
// the NotifyOnWaiting toggle, instead of waiting for the first lazy
// sendNotification call. Bundle-guarded + dispatched to the main queue,
// mirroring sendNotification's fail-safe contract.
void requestNotificationAuthorization(void) {
    if (!hasValidBundleIdentifier()) {
        NSLog(@"AgentHub: skipping notification authorization request — no valid app-bundle identifier (unbundled/unsigned process)");
        return;
    }
    dispatch_async(dispatch_get_main_queue(), ^{
        @try {
            registerNotificationDelegate();
            [[UNUserNotificationCenter currentNotificationCenter]
                requestAuthorizationWithOptions:(UNAuthorizationOptionAlert | UNAuthorizationOptionSound)
                completionHandler:^(BOOL granted, NSError *error) {
                NSLog(@"AgentHub: proactive notification authorization result granted=%d error=%@", granted, error);
                onNotificationAuthResult(granted ? 1 : 0);
            }];
        } @catch (NSException *e) {
            NSLog(@"AgentHub: swallowed exception requesting proactive notification authorization: %@", e);
        }
    });
}

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

// setDockVisible shows or hides the macOS Dock icon by toggling the
// application activation policy between Regular (visible) and Accessory (hidden).
// When showing, the app is also activated so the window comes forward.
void setDockVisible(int visible) {
    dispatch_async(dispatch_get_main_queue(), ^{
        if (visible) {
            [NSApp setActivationPolicy:NSApplicationActivationPolicyRegular];
            [NSApp activateIgnoringOtherApps:YES];
        } else {
            [NSApp setActivationPolicy:NSApplicationActivationPolicyAccessory];
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

// hasValidBundleIdentifier reports whether the running process has a valid
// app-bundle identifier. UNUserNotificationCenter requires one; an unbundled
// process (e.g. `wails dev`, a `go test` binary) has none, and calling into
// UNUserNotificationCenter without this guard raises an uncaught
// NSInternalInconsistencyException ("bundleProxyForCurrentProcess is nil")
// that aborts the whole process (Phase 167 M-41 regression).
int hasValidBundleIdentifier(void) {
    return [[NSBundle mainBundle] bundleIdentifier] != nil ? 1 : 0;
}

// sendNotification sends a macOS notification using UNUserNotificationCenter.
// Requests permission lazily on first call; no-ops if the user denies.
// The identifier is caller-supplied so concurrent notifications (e.g. two
// sessions transitioning to waiting close together) do not collide and
// silently replace one another (RESEARCH Pitfall 2).
//
// Fail-safe (Phase 167-05 gap closure, M-41): if the process has no valid
// app-bundle identifier, UNUserNotificationCenter is unusable and calling it
// aborts the process. Guard synchronously BEFORE dispatch so the crash-prone
// API is never reached in that case — log-and-swallow, mirroring the beeep
// Windows/Linux wrappers' contract. On a properly bundled/signed .app this
// guard passes and the notification fires exactly as before.
void sendNotification(const char *identifier, const char *title, const char *body) {
    if (!hasValidBundleIdentifier()) {
        NSLog(@"AgentHub: skipping native notification — no valid app-bundle identifier (unbundled/unsigned process)");
        return;
    }
    NSString *nsIdentifier = [NSString stringWithUTF8String:identifier];
    NSString *nsTitle = [NSString stringWithUTF8String:title];
    NSString *nsBody  = [NSString stringWithUTF8String:body];
    dispatch_async(dispatch_get_main_queue(), ^{
        NSLog(@"AgentHub: dispatching native notification for identifier %@", nsIdentifier);
        registerNotificationDelegate();
        @try {
            UNUserNotificationCenter *center = [UNUserNotificationCenter currentNotificationCenter];
            [center requestAuthorizationWithOptions:(UNAuthorizationOptionAlert | UNAuthorizationOptionSound)
                completionHandler:^(BOOL granted, NSError *error) {
                if (!granted) {
                    NSLog(@"AgentHub: notification authorization NOT granted for identifier %@ (error=%@)", nsIdentifier, error);
                    return;
                }
                dispatch_async(dispatch_get_main_queue(), ^{
                    @try {
                        UNMutableNotificationContent *content = [[UNMutableNotificationContent alloc] init];
                        content.title = nsTitle;
                        content.body  = nsBody;
                        UNNotificationRequest *req = [UNNotificationRequest
                            requestWithIdentifier:nsIdentifier
                            content:content
                            trigger:nil];
                        [center addNotificationRequest:req withCompletionHandler:^(NSError * _Nullable error) {
                            if (error != nil) {
                                NSLog(@"AgentHub: notification delivery FAILED for identifier %@: %@", nsIdentifier, error);
                            } else {
                                NSLog(@"AgentHub: notification delivery accepted for identifier %@", nsIdentifier);
                            }
                        }];
                    } @catch (NSException *e) {
                        NSLog(@"AgentHub: swallowed exception adding notification request: %@", e);
                    }
                });
            }];
        } @catch (NSException *e) {
            NSLog(@"AgentHub: swallowed exception requesting notification authorization: %@", e);
        }
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
