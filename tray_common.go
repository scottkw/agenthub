package main

import "fmt"

// trayTooltip returns the tooltip string for the given session count.
// Uses an em dash (U+2014) as specified in the UI-SPEC copywriting contract.
func trayTooltip(n int) string {
	switch n {
	case 0:
		return "AgentHub \u2014 no sessions"
	case 1:
		return "AgentHub \u2014 1 session"
	default:
		return fmt.Sprintf("AgentHub \u2014 %d sessions", n)
	}
}

// MenuItemKind classifies tray menu entries.
type MenuItemKind int

const (
	MenuItemAction    MenuItemKind = iota // clickable item
	MenuItemSeparator                     // visual divider
)

// MenuItem represents a single entry in the tray context menu.
type MenuItem struct {
	Kind      MenuItemKind
	Label     string
	SessionID string // non-empty for session items
	Index     int    // session index for callback dispatch
}

// BuildMenuItems constructs the canonical tray menu structure:
// "Open AgentHub" -> separator -> [sessions] -> separator -> "Quit"
// The first separator is always present. When sessions is empty the result is:
// Open AgentHub, separator, separator, Quit (4 items).
// This is the single source of truth for menu layout across all platforms.
func BuildMenuItems(sessions []SessionInfo) []MenuItem {
	items := []MenuItem{
		{Kind: MenuItemAction, Label: "Open AgentHub"},
		{Kind: MenuItemSeparator},
	}
	for i, s := range sessions {
		items = append(items, MenuItem{
			Kind:      MenuItemAction,
			Label:     s.Name,
			SessionID: s.ID,
			Index:     i,
		})
	}
	items = append(items,
		MenuItem{Kind: MenuItemSeparator},
		MenuItem{Kind: MenuItemAction, Label: "Quit"},
	)
	return items
}
