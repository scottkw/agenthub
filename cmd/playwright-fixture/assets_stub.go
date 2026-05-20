//go:build playwrightfixture && !wailsassets

// Phase 120-06 Task 3 — dev-build stub. Mirrors the repo-root assets_stub.go.
//
// When the fixture is built without `-tags wailsassets` the React bundle is
// not embedded; staticAppFixture() returns nil so the webserver's /app/ route
// answers 503 instead of accidentally exposing the working directory tree
// over the test web stack. This keeps `go build -tags=playwrightfixture` (no
// wailsassets) viable for local debugging without forcing a frontend rebuild.

package main

import "io/fs"

func staticAppFixture() fs.FS { return nil }
