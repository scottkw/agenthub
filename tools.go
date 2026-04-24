//go:build tools
// +build tools

// Package tools documents build-tool dependencies. It is excluded from normal
// builds by the `tools` build tag. The blank imports cause `go mod tidy` to
// keep these modules in go.mod alongside runtime dependencies, making Dependabot
// aware of them via the gomod ecosystem.
//
// CI and build.sh install these tools using:
//
//	go install <path>@$(go list -m -f '{{.Version}}' <module>)
//
// See .planning/phases/90-release-pipeline-hardening/ for rationale.
package main

import (
	_ "github.com/goreleaser/nfpm/v2"
	_ "github.com/wailsapp/wails/v2"
)
