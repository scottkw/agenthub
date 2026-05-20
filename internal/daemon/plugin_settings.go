package daemon

// CurrentSchemaVersion is the on-disk daemonSettings schema version.
// v3.1 settings.json files have no schemaVersion field (treated as 0);
// v3.2 introduces the plugins block and bumps to 2; v3.4 (Phase 118)
// adds the daemon-wide FilesRead flag and bumps to 3. Future migrations
// bump and re-save via the same defaults-merge load path.
const CurrentSchemaVersion = 3

// SearchConfig persists per-flag default state for the find-bar toggle row.
// Phase 94 (SRC-02). All defaults FALSE — the toggles ship in their "off"
// position, and the user's choice is remembered for next session.
//
// Field order matches UI-SPEC §"Find bar Settings integration".
// JSON tags are camelCase to match daemonSettings vocabulary.
//
// NO omitempty: missing keys must round-trip as the user's saved choice
// (which may legitimately be `false`).
type SearchConfig struct {
	Regex         bool `json:"regex"`
	CaseSensitive bool `json:"caseSensitive"`
	WholeWord     bool `json:"wholeWord"`
}

// WebLinksConfig persists per-plugin runtime configuration for the
// web-links toggle (Phase 95 LNK-02, LNK-03, LNK-05). Defaults are
// platform-correct + ALL confirmations ON (security-first posture
// per ROADMAP §"Phase 95 — v3.1-WS-Origin-allowlist rigor").
//
// JSON tags are camelCase to match daemonSettings vocabulary; field
// ordering matches the ## Files to Create / Modify table in
// .planning/phases/95-web-links-addon-security-hardening/95-RESEARCH.md
// §"Pattern 3: WebLinksConfig Persistence".
type WebLinksConfig struct {
	// Modifier: "platform" | "cmd" | "ctrl" | "none"
	//   "platform" — Cmd on macOS, Ctrl elsewhere (default)
	//   "cmd"      — always require Cmd (macOS-only convenience)
	//   "ctrl"     — always require Ctrl (universal alternative)
	//   "none"     — disable modifier requirement (still gated by
	//                scheme allowlist + risk detection — see
	//                95-RESEARCH §"Pitfall 9: Modifier Configuration 'none'")
	Modifier         string `json:"modifier"`
	ConfirmOSC8      bool   `json:"confirmOSC8"`
	ConfirmIDN       bool   `json:"confirmIDN"`
	ConfirmTyposquat bool   `json:"confirmTyposquat"`
}

// ImageConfig persists per-plugin runtime configuration for the
// inline-image addon (Phase 96 IMG-02). Defaults to a 16 MB cap on
// decoded RGBA storage, overriding upstream addon-image's 128 MB
// default to prevent tab-OOM on multi-tab AgentHub sessions
// (STATE.md §Decisions, ROADMAP Phase 96 SC-3).
//
// JSON tag is camelCase to match daemonSettings vocabulary; field
// ordering follows .planning/phases/96-image-addon-csp-audit/
// 96-RESEARCH.md §"Pattern 2: ImageConfig Persistence".
//
// Phase 96 ships ONLY this single field. The Advanced <details> UI
// exposing storageLimit defers to Phase 99 / PUI-03 (mirror of
// Phase 95 WebLinksConfig.modifier deferral shape).
type ImageConfig struct {
	StorageLimit int `json:"storageLimit"`
}

// PluginSettings is the persisted user choice for each xterm.js addon
// shipped by the v3.2 plugin suite. Field names match UI-SPEC ordering.
// JSON tags are camelCase to match daemonSettings vocabulary (cliPaths,
// startMinimized).
//
// NO omitempty: missing keys must round-trip as the user's saved choice
// (which may legitimately be `false`), and Pitfall #14 mandates the
// parent `plugins` key always serialize so future loads observe it.
type PluginSettings struct {
	WebGL          bool           `json:"webgl"`
	Unicode11      bool           `json:"unicode11"`
	Search         bool           `json:"search"`
	SearchConfig   SearchConfig   `json:"searchConfig"`
	WebLinks       bool           `json:"webLinks"`
	WebLinksConfig WebLinksConfig `json:"webLinksConfig"`
	Image          bool           `json:"image"`
	ImageConfig    ImageConfig    `json:"imageConfig"`
	Serialize      bool           `json:"serialize"`
	Clipboard      bool           `json:"clipboard"`
	Progress       bool           `json:"progress"`
}

// defaultPluginSettings returns the v3.2 default plugin enable/disable
// state per ROADMAP `## Decisions` and UI-SPEC §"Default ON / OFF on
// first launch": all plugins ON except Progress (default OFF, flips
// ON in v3.3 after field validation).
//
// This function is the single source of truth for plugin defaults.
// Returning v3.1 users hit it via the defaults-merge in
// loadSettingsFromDisk (Pitfall #14 mitigation).
func defaultPluginSettings() PluginSettings {
	return PluginSettings{
		WebGL:        true,
		Unicode11:    true,
		Search:       true,
		SearchConfig: SearchConfig{Regex: false, CaseSensitive: false, WholeWord: false},
		WebLinks:     true,
		WebLinksConfig: WebLinksConfig{
			Modifier:         "platform",
			ConfirmOSC8:      true,
			ConfirmIDN:       true,
			ConfirmTyposquat: true,
		},
		Image:       true,
		ImageConfig: ImageConfig{StorageLimit: 16},
		Serialize:   true,
		Clipboard:   true,
		Progress:    false,
	}
}
