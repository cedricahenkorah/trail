package main

import (
	"runtime/debug"

	cmd "github.com/cedricahenkorah/trail/internal/cli"
)

// version is set at build time by GoReleaser via -ldflags "-X main.version=...".
var version = "dev"

func main() {
	cmd.Execute(resolveVersion())
}

// resolveVersion falls back to the module version recorded by go install
// (e.g. v0.1.0) when the binary wasn't built by GoReleaser.
func resolveVersion() string {
	if version != "dev" {
		return version
	}

	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}

	return version
}
