//go:build playwrightfixture && wailsassets

// Phase 120-06 Task 3 — embedded React bundle for the playwright fixture.
//
// `//go:embed` patterns cannot escape the package directory (they may not
// contain `..` and may not match files outside the package tree). The
// fixture therefore embeds `cmd/playwright-fixture/dist/` rather than the
// canonical `frontend/dist/` — global-setup.ts copies the latter into the
// former immediately before running `go build -tags=playwrightfixture,wailsassets`.
//
// The strategy mirrors the repo-root assets_prod.go (//go:embed all:frontend/dist)
// but is local to this package so the fixture stays self-contained.
//
// When the dist subtree is missing at build time, the embed declaration
// would normally fail compilation; we keep `all:dist` with no quotes to
// require the dist subtree be present. global-setup.ts always copies it in
// before invoking go build, so this contract is enforced one layer up.

package main

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var embeddedAppFS embed.FS

// staticAppFixture returns the React bundle's fs.FS scoped to the dist/
// subdirectory so file paths read as `index.html` rather than `dist/index.html`.
// Mirrors the repo-root `fs.Sub(embeddedAssets, "frontend/dist")` pattern.
func staticAppFixture() fs.FS {
	sub, err := fs.Sub(embeddedAppFS, "dist")
	if err != nil {
		// fs.Sub only errors on an invalid path — `dist` is a valid name.
		// Returning the raw FS would cause /app/ to serve `dist/index.html`
		// for the root request, which is wrong; panicking surfaces the
		// build-time misconfiguration loudly rather than silently 404'ing.
		panic("playwright-fixture: fs.Sub(dist) failed: " + err.Error())
	}
	return sub
}
