// Package version provides version information for unqueryvet.
package version

import (
	"fmt"
	"runtime"
)

// Build information. Populated at build-time via ldflags.
var (
	// Version is the current version of unqueryvet.
	Version = "1.5.2"

	// Commit is the git commit hash.
	Commit = "dev"

	// Date is the build date.
	Date = "unknown"

	// BuiltBy is the builder.
	BuiltBy = "unknown"
)

// Info contains version and build information.
type Info struct {
	Version   string
	Commit    string
	Date      string
	BuiltBy   string
	GoVersion string
	Platform  string
}

// GetInfo returns version and build information.
func GetInfo() Info {
	return Info{
		Version:   Version,
		Commit:    Commit,
		Date:      Date,
		BuiltBy:   BuiltBy,
		GoVersion: runtime.Version(),
		Platform:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}
}

// String returns a formatted version string.
func (i Info) String() string {
	return fmt.Sprintf("unqueryvet version %s\n"+
		"  commit: %s\n"+
		"  built:  %s\n"+
		"  by:     %s\n"+
		"  go:     %s\n"+
		"  platform: %s",
		i.Version, i.Commit, i.Date, i.BuiltBy, i.GoVersion, i.Platform)
}

// Short returns a short version string.
func (i Info) Short() string {
	if i.Commit != "dev" && len(i.Commit) > 7 {
		return fmt.Sprintf("unqueryvet v%s (%s)", i.Version, i.Commit[:7])
	}
	return fmt.Sprintf("unqueryvet v%s", i.Version)
}
