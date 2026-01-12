package version

import (
	"runtime"
	"strings"
	"testing"
)

func TestVersion(t *testing.T) {
	if Version == "" {
		t.Error("Version should not be empty")
	}

	// Version should follow semver format
	parts := strings.Split(Version, ".")
	if len(parts) != 3 {
		t.Errorf("Version should be in semver format (x.y.z), got: %s", Version)
	}
}

func TestGetInfo(t *testing.T) {
	info := GetInfo()

	if info.Version != Version {
		t.Errorf("GetInfo().Version = %s, want %s", info.Version, Version)
	}

	if info.Commit != Commit {
		t.Errorf("GetInfo().Commit = %s, want %s", info.Commit, Commit)
	}

	if info.Date != Date {
		t.Errorf("GetInfo().Date = %s, want %s", info.Date, Date)
	}

	if info.BuiltBy != BuiltBy {
		t.Errorf("GetInfo().BuiltBy = %s, want %s", info.BuiltBy, BuiltBy)
	}

	if info.GoVersion != runtime.Version() {
		t.Errorf("GetInfo().GoVersion = %s, want %s", info.GoVersion, runtime.Version())
	}

	expectedPlatform := runtime.GOOS + "/" + runtime.GOARCH
	if info.Platform != expectedPlatform {
		t.Errorf("GetInfo().Platform = %s, want %s", info.Platform, expectedPlatform)
	}
}

func TestInfoString(t *testing.T) {
	info := GetInfo()
	str := info.String()

	// Should contain version
	if !strings.Contains(str, Version) {
		t.Error("Info.String() should contain version")
	}

	// Should contain "unqueryvet version"
	if !strings.Contains(str, "unqueryvet version") {
		t.Error("Info.String() should contain 'unqueryvet version'")
	}

	// Should contain commit
	if !strings.Contains(str, "commit:") {
		t.Error("Info.String() should contain 'commit:'")
	}

	// Should contain platform
	if !strings.Contains(str, "platform:") {
		t.Error("Info.String() should contain 'platform:'")
	}
}

func TestInfoShort(t *testing.T) {
	tests := []struct {
		name     string
		commit   string
		contains string
	}{
		{
			name:     "dev commit",
			commit:   "dev",
			contains: "unqueryvet v",
		},
		{
			name:     "short commit",
			commit:   "abc123",
			contains: "unqueryvet v",
		},
		{
			name:     "long commit",
			commit:   "abc1234567890",
			contains: "(abc1234)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original
			origCommit := Commit
			defer func() { Commit = origCommit }()

			Commit = tt.commit
			info := GetInfo()
			short := info.Short()

			if !strings.Contains(short, tt.contains) {
				t.Errorf("Info.Short() = %s, should contain %s", short, tt.contains)
			}

			if !strings.Contains(short, Version) {
				t.Errorf("Info.Short() = %s, should contain version %s", short, Version)
			}
		})
	}
}

func TestInfoShortWithLongCommit(t *testing.T) {
	// Save original
	origCommit := Commit
	defer func() { Commit = origCommit }()

	Commit = "abcdef1234567890"
	info := GetInfo()
	short := info.Short()

	// Should truncate commit to 7 characters
	if !strings.Contains(short, "abcdef1") {
		t.Errorf("Info.Short() should truncate long commit to 7 chars, got: %s", short)
	}

	// Should not contain full commit
	if strings.Contains(short, "abcdef1234567890") {
		t.Errorf("Info.Short() should not contain full commit, got: %s", short)
	}
}
