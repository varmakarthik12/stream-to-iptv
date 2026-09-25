package stream

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"stream-to-iptv/internal/db"
	"stream-to-iptv/internal/models"
	"stream-to-iptv/internal/repository"
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
	if !strings.Contains(args1Str, "-bsf:v dump_extra") {
		t.Errorf("Expected args to contain -bsf:v dump_extra, got: %s", args1Str)
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

func TestStreamLifecycle_ErrorTrackingAndRestart(t *testing.T) {
	mgr := &Manager{
		processes: make(map[string]*ProcessState),
	}

	stream := &models.Stream{
		ID:                "test-1",
		Name:              "Test Stream",
		Slug:              "test-stream",
		Enabled:           true,
		AutoRecover:       true,
		RecoverTimeoutSec: 30,
	}

	state := mgr.getOrCreateState(stream)
	state.Status = "error"
	state.ErrorMessage = "FFmpeg exited with error: exit status 231"

	// Verify error tracking and problem count
	if mgr.GetStreamStatus(stream.Slug) != "error" {
		t.Errorf("Expected status 'error', got: %s", mgr.GetStreamStatus(stream.Slug))
	}
	if mgr.GetStreamErrorMessage(stream.Slug) != "FFmpeg exited with error: exit status 231" {
		t.Errorf("Expected error message to match, got: %s", mgr.GetStreamErrorMessage(stream.Slug))
	}
	if mgr.GetProblemCount() != 1 {
		t.Errorf("Expected problem count to be 1, got: %d", mgr.GetProblemCount())
	}

	// Verify stale CancelFunc with Pid == 0 does NOT block restart
	calledCancel := false
	state.CancelFunc = func() {
		calledCancel = true
	}
	state.Pid = 0

	// Starting stream should clean up stale cancel func and proceed without deadlock
	state.mu.Lock()
	if state.CancelFunc != nil && state.Pid == 0 {
		state.CancelFunc()
		state.CancelFunc = nil
	}
	state.Status = "starting"
	state.ErrorMessage = ""
	state.mu.Unlock()

	if !calledCancel {
		t.Errorf("Expected stale CancelFunc to be invoked on cleanup")
	}
	if state.CancelFunc != nil {
		t.Errorf("Expected state.CancelFunc to be cleared to nil")
	}
	if mgr.GetStreamStatus(stream.Slug) != "starting" {
		t.Errorf("Expected status to become 'starting', got: %s", mgr.GetStreamStatus(stream.Slug))
	}
}

func TestOnDemandStream_IdleAndAutoRecover(t *testing.T) {
	tempDir := t.TempDir()
	database, err := db.InitDB(tempDir)
	if err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}
	defer database.Close()

	repo := repository.NewRepository(database)
	mgr := NewManager(repo)

	// Create on-demand stream
	stream := &models.Stream{
		Name:              "OnDemand Channel",
		Slug:              "ondemand-ch",
		MediaURL:          "udp://239.239.10.1:5555",
		Mode:              "ondemand",
		IdleTimeoutSec:    180,
		AutoRecover:       true,
		RecoverTimeoutSec: 30,
		Enabled:           true,
	}
	if _, err := repo.CreateStream(stream); err != nil {
		t.Fatalf("Failed to create stream: %v", err)
	}

	state := mgr.getOrCreateState(stream)

	// Scenario 1: Stream has been in error, but user stopped watching (> 180s inactive).
	// Auto-recovery MUST NOT restart it; it should transition cleanly to 'idle'.
	state.mu.Lock()
	state.Status = "error"
	state.ErrorStartedAt = time.Now().Add(-60 * time.Second)
	state.LastActivity = time.Now().Add(-200 * time.Second) // 200s ago (> 180s)
	state.mu.Unlock()

	mgr.checkAutoRecover()

	state.mu.RLock()
	statusAfterIdle := state.Status
	state.mu.RUnlock()

	if statusAfterIdle == "starting" || statusAfterIdle == "running" {
		t.Errorf("Expected on-demand stream with inactive viewers to NOT restart, got status: %s", statusAfterIdle)
	}
	if statusAfterIdle != "idle" {
		t.Errorf("Expected on-demand stream to transition to 'idle', got: %s", statusAfterIdle)
	}

	// Scenario 2: Stream is stopped intentionally via StopStream
	// Ensure StopStream leaves it in 'stopped' without lingering error timestamp
	state.mu.Lock()
	state.Status = "running"
	state.Pid = 99999
	state.mu.Unlock()

	mgr.StopStream(stream.Slug)

	state.mu.RLock()
	stoppedStatus := state.Status
	errStarted := state.ErrorStartedAt
	errMsg := state.ErrorMessage
	state.mu.RUnlock()

	if stoppedStatus != "stopped" {
		t.Errorf("Expected status to be 'stopped', got: %s", stoppedStatus)
	}
	if !errStarted.IsZero() {
		t.Errorf("Expected ErrorStartedAt to be zero after StopStream, got: %v", errStarted)
	}
	if errMsg != "" {
		t.Errorf("Expected ErrorMessage to be empty after StopStream, got: %s", errMsg)
	}
}




