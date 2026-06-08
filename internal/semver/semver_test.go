package semver

import (
	"strings"
	"testing"
	"time"
)

func TestString(t *testing.T) {
	ver := String()
	if ver == "" {
		t.Fatal("version should not be empty")
	}
}

func TestCommitString(t *testing.T) {
	commit := CommitString()
	if commit == "" {
		t.Fatal("commit should not be empty")
	}
	// Either "unknown" or a hex hash
	if commit != "unknown" && len(commit) < 7 {
		t.Fatalf("unexpected commit format: %q", commit)
	}
}

func TestDateString(t *testing.T) {
	ds := DateString()
	_, err := time.Parse("2006-01-02T15:04:05Z", ds)
	if err != nil {
		t.Fatalf("DateString should be ISO 8601, got %q: %v", ds, err)
	}
	if !strings.HasSuffix(ds, "Z") {
		t.Fatal("DateString should end with Z (UTC)")
	}
}
