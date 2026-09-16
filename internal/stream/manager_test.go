package stream

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestResolveSystemFFmpeg(t *testing.T) {
	bin := ResolveSystemFFmpeg()
	if bin == "" {
		t.Fatal("Expected non-empty FFmpeg binary resolution")
	}
	t.Logf("Resolved system FFmpeg: %s", bin)

	// If on Windows and Chocolatey is present, ensure it didn't return the shimgen wrapper
	if runtime.GOOS == "windows" {
		if _, err := os.Stat(`C:\ProgramData\chocolatey\bin\ffmpeg.exe`); err == nil {
			if filepath.Base(filepath.Dir(bin)) == "bin" && filepath.Base(filepath.Dir(filepath.Dir(bin))) == "chocolatey" {
				t.Fatalf("Resolved to chocolatey bin shim wrapper instead of unwrapped binary: %s", bin)
			}
		}
	}
}

func TestResolveFFmpegBinary_EnvOverride(t *testing.T) {
	tempDir := t.TempDir()
	dummyBin := filepath.Join(tempDir, "ffmpeg-custom")
	if runtime.GOOS == "windows" {
		dummyBin += ".exe"
	}
	if err := os.WriteFile(dummyBin, []byte("echo test"), 0755); err != nil {
		t.Fatalf("Failed to create dummy binary: %v", err)
	}

	t.Setenv("FFMPEG_PATH", dummyBin)

	bin := ResolveSystemFFmpeg()
	if bin != dummyBin {
		t.Fatalf("Expected env FFMPEG_PATH override %s, got %s", dummyBin, bin)
	}
}

func TestUnwrapWindowsShim_Scoop(t *testing.T) {
	tempDir := t.TempDir()
	shimDir := filepath.Join(tempDir, "shims")
	appsDir := filepath.Join(tempDir, "apps", "ffmpeg", "current", "bin")
	_ = os.MkdirAll(shimDir, 0755)
	_ = os.MkdirAll(appsDir, 0755)

	realTarget := filepath.Join(appsDir, "ffmpeg.exe")
	_ = os.WriteFile(realTarget, []byte("binary"), 0755)

	shimExe := filepath.Join(shimDir, "ffmpeg.exe")
	_ = os.WriteFile(shimExe, []byte("shim"), 0755)

	shimConfig := filepath.Join(shimDir, "ffmpeg.shim")
	_ = os.WriteFile(shimConfig, []byte("path = \""+realTarget+"\"\r\nargs = \"\"\r\n"), 0644)

	resolved := unwrapWindowsShim(shimExe)
	if resolved != realTarget {
		t.Fatalf("Expected Scoop shim unwrapped to %s, got %s", realTarget, resolved)
	}
}
