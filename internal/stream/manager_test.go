package stream

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"stream-to-iptv/internal/models"
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

func TestBuildFFmpegArgs_LocalAddrAndProgramID(t *testing.T) {
	tempDir := t.TempDir()

	// 1. Explicit LocalAddr on stream
	s1 := &models.Stream{
		Slug:       "test-stream-1",
		MediaURL:   "udp://@239.255.10.10:1234",
		ProgramID:  "105",
		LocalAddr:  "192.168.1.50",
		BufferSize: "2000000",
		FifoSize:   "500000",
	}
	args1 := buildFFmpegArgs(s1, tempDir)
	args1Str := strings.Join(args1, " ")

	if !strings.Contains(args1Str, "localaddr=192.168.1.50") {
		t.Errorf("Expected args to contain localaddr=192.168.1.50, got: %s", args1Str)
	}
	if !strings.Contains(args1Str, "-map 0:p:105") {
		t.Errorf("Expected args to contain -map 0:p:105, got: %s", args1Str)
	}
	if !strings.Contains(args1Str, "fifo_size=500000") {
		t.Errorf("Expected args to contain fifo_size=500000, got: %s", args1Str)
	}

	// 2. Fallback to IP_ADDR environment variable when LocalAddr is empty
	t.Setenv("IP_ADDR", "10.0.0.99")
	s2 := &models.Stream{
		Slug:       "test-stream-2",
		MediaURL:   "udp://@239.255.10.10:1234",
		ProgramID:  "200",
		LocalAddr:  "",
		BufferSize: "1000000",
	}
	args2 := buildFFmpegArgs(s2, tempDir)
	args2Str := strings.Join(args2, " ")

	if !strings.Contains(args2Str, "localaddr=10.0.0.99") {
		t.Errorf("Expected fallback to IP_ADDR env var (localaddr=10.0.0.99), got: %s", args2Str)
	}
	if !strings.Contains(args2Str, "-map 0:p:200") {
		t.Errorf("Expected args to contain -map 0:p:200, got: %s", args2Str)
	}

	// 3. MediaURL with existing query parameters
	s3 := &models.Stream{
		Slug:       "test-stream-3",
		MediaURL:   "udp://@239.255.10.10:1234?pkt_size=1316",
		ProgramID:  "1",
		LocalAddr:  "172.16.0.2",
	}
	args3 := buildFFmpegArgs(s3, tempDir)
	args3Str := strings.Join(args3, " ")

	if !strings.Contains(args3Str, "pkt_size=1316&") && !strings.Contains(args3Str, "pkt_size=1316") {
		t.Errorf("Expected existing query param pkt_size=1316 to be preserved, got: %s", args3Str)
	}
	if !strings.Contains(args3Str, "localaddr=172.16.0.2") {
		t.Errorf("Expected localaddr=172.16.0.2 to be appended, got: %s", args3Str)
	}
}

func TestBuildFFmpegArgs_ProbingSettings(t *testing.T) {
	tempDir := t.TempDir()

	// 1. Standard SPTS stream without ProgramID -> fast defaults (2000000 / 2000000)
	sptsStream := &models.Stream{
		Slug:     "spts-stream",
		MediaURL: "http://example.com/live.m3u8",
	}
	sptsArgs := strings.Join(buildFFmpegArgs(sptsStream, tempDir), " ")
	if !strings.Contains(sptsArgs, "-analyzeduration 2000000 -probesize 2000000") {
		t.Errorf("Expected SPTS stream to default to 2000000/2000000, got: %s", sptsArgs)
	}

	// 2. MPTS stream with ProgramID -> auto-scaled defaults (5000000 / 10000000)
	mptsStream := &models.Stream{
		Slug:      "mpts-stream",
		MediaURL:  "udp://239.239.10.3:5555",
		ProgramID: "10303",
	}
	mptsArgs := strings.Join(buildFFmpegArgs(mptsStream, tempDir), " ")
	if !strings.Contains(mptsArgs, "-analyzeduration 5000000 -probesize 10000000") {
		t.Errorf("Expected MPTS stream to auto-scale to 5000000/10000000, got: %s", mptsArgs)
	}

	// 3. Stream with explicit custom overrides -> overrides defaults
	customStream := &models.Stream{
		Slug:            "custom-stream",
		MediaURL:        "udp://239.239.10.3:5555",
		ProgramID:       "10303",
		AnalyzeDuration: "8000000",
		ProbeSize:       "16000000",
	}
	customArgs := strings.Join(buildFFmpegArgs(customStream, tempDir), " ")
	if !strings.Contains(customArgs, "-analyzeduration 8000000 -probesize 16000000") {
		t.Errorf("Expected custom overrides to take precedence, got: %s", customArgs)
	}
}


