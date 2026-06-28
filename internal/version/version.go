// Package version holds the plus3 version, kept in sync with the root VERSION
// file by tools/syncver.sh. The Version string may also be overridden at build
// time via -ldflags "-X github.com/ha1tch/plus3/internal/version.Version=...".
package version

// Version is the current plus3 version (MAJOR.MINOR.PATCH).
// Synced from the root VERSION file by tools/syncver.sh.
var Version = "0.9.6"
