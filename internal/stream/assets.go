package stream

import (
	_ "embed"
	"os"
	"path/filepath"
)

//go:embed assets/loading.ts
var DefaultLoadingBumper []byte

// EnsureLoadingBumper guarantees that loading.ts exists in dataDir.
// If it doesn't exist, it writes the embedded default bumper.
func EnsureLoadingBumper(dataDir string) error {
	target := filepath.Join(dataDir, "loading.ts")
	if _, err := os.Stat(target); err == nil {
		return nil
	}
	return os.WriteFile(target, DefaultLoadingBumper, 0644)
}
