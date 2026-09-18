package stream

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"stream-to-iptv/internal/db"
	"stream-to-iptv/internal/models"
	"stream-to-iptv/internal/repository"

	"github.com/sirupsen/logrus"
)

// ResolveFFmpegBinary dynamically finds the direct FFmpeg binary executable without hardcoded machine paths.
// Resolution order:
// 1. App configuration setting ("ffmpeg_path") in database
// 2. Environment variables ("FFMPEG_PATH" or "FFMPEG_BIN")
// 3. System PATH lookup (exec.LookPath)
// 4. Dynamic unwrapping of Windows package manager shims/symlinks (Scoop, Chocolatey, WinGet)
// 5. Fallback to "ffmpeg"
func (m *Manager) ResolveFFmpegBinary() string {
	if m != nil && m.repo != nil {
		if customPath, err := m.repo.GetSetting("ffmpeg_path", ""); err == nil && strings.TrimSpace(customPath) != "" {
			customPath = strings.TrimSpace(customPath)
			if fileExistsAndExec(customPath) {
				return customPath
			}
		}
	}

	return ResolveSystemFFmpeg()
}

// ResolveSystemFFmpeg discovers the true FFmpeg executable on the system without hardcoded paths.
func ResolveSystemFFmpeg() string {
	// 1. Check environment variables
	for _, envKey := range []string{"FFMPEG_PATH", "FFMPEG_BIN"} {
		if val := strings.TrimSpace(os.Getenv(envKey)); val != "" {
			if fileExistsAndExec(val) {
				return val
			}
		}
	}

	// 2. Lookup in system PATH
	candidate, err := exec.LookPath("ffmpeg")
	if err != nil && runtime.GOOS == "windows" {
		candidate, err = exec.LookPath("ffmpeg.exe")
	}

	if err == nil && candidate != "" {
		// Unwrap symlinks / junctions
		if resolved, err := filepath.EvalSymlinks(candidate); err == nil && resolved != "" {
			candidate = resolved
		}

		// On Windows, dynamically unwrap shims from package managers (Scoop, Chocolatey, WinGet)
		if runtime.GOOS == "windows" {
			if realBin := unwrapWindowsShim(candidate); realBin != "" {
				return realBin
			}
		}
		return candidate
	}

	// 3. Common system fallbacks if not found on PATH
	if runtime.GOOS != "windows" {
		for _, fallback := range []string{"/usr/bin/ffmpeg", "/usr/local/bin/ffmpeg", "/opt/homebrew/bin/ffmpeg"} {
			if fileExistsAndExec(fallback) {
				return fallback
			}
		}
	}

	return "ffmpeg"
}

func fileExistsAndExec(path string) bool {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return false
	}
	return true
}

// unwrapWindowsShim inspects whether candidate is a known Windows shim (Scoop, Chocolatey)
// and dynamically resolves the real binary target without hardcoding paths.
func unwrapWindowsShim(candidate string) string {
	dir := filepath.Dir(candidate)

	// A. Check for Scoop shim (.shim text file next to .exe)
	baseName := strings.TrimSuffix(filepath.Base(candidate), filepath.Ext(candidate))
	shimFile := filepath.Join(dir, baseName+".shim")
	if data, err := os.ReadFile(shimFile); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(strings.ToLower(line), "path") && strings.Contains(line, "=") {
				parts := strings.SplitN(line, "=", 2)
				if len(parts) == 2 {
					target := strings.Trim(strings.TrimSpace(parts[1]), `"'`)
					if fileExistsAndExec(target) {
						return target
					}
				}
			}
		}
	}

	// B. Check for Chocolatey shim
	// In Chocolatey, shims reside in <ChocoRoot>\bin. The real package binaries reside in
	// <ChocoRoot>\lib\ffmpeg\tools\**\ffmpeg.exe.
	var chocoRoots []string
	if envRoot := os.Getenv("ChocolateyInstall"); envRoot != "" {
		chocoRoots = append(chocoRoots, envRoot)
	}
	if strings.EqualFold(filepath.Base(dir), "bin") {
		chocoRoots = append(chocoRoots, filepath.Dir(dir))
	}

	for _, chocoRoot := range chocoRoots {
		pkgDir := filepath.Join(chocoRoot, "lib", "ffmpeg")
		if info, err := os.Stat(pkgDir); err == nil && info.IsDir() {
			var realBin string
			_ = filepath.WalkDir(pkgDir, func(path string, d os.DirEntry, err error) error {
				if err != nil {
					return nil
				}
				if !d.IsDir() && strings.EqualFold(d.Name(), "ffmpeg.exe") {
					realBin = path
					return filepath.SkipAll
				}
				return nil
			})
			if realBin != "" && fileExistsAndExec(realBin) {
				return realBin
			}
		}
	}

	return ""
}

// killProcessTree terminates the given PID and all its descendant processes.
func killProcessTree(pid int) {
	if pid <= 0 {
		return
	}
	if runtime.GOOS == "windows" {
		_ = exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(pid)).Run()
	} else {
		_ = exec.Command("kill", "-9", strconv.Itoa(pid)).Run()
	}
}

// killStreamProcessesBySlug cleans up any lingering ffmpeg processes operating on the stream's slug
func killStreamProcessesBySlug(slug string) {
	if slug == "" {
		return
	}
	if runtime.GOOS == "windows" {
		pattern := fmt.Sprintf("*%s*", slug)
		_ = exec.Command("powershell", "-NoProfile", "-Command", fmt.Sprintf(
			`Get-CimInstance Win32_Process -Filter "Name LIKE 'ffmpeg%%'" | Where-Object { $_.CommandLine -like '%s' } | ForEach-Object { Stop-Process -Id $_.ProcessId -Force -ErrorAction SilentlyContinue }`,
			pattern,
		)).Run()
	} else {
		_ = exec.Command("pkill", "-f", fmt.Sprintf("ffmpeg.*%s", slug)).Run()
	}
}

type ProcessState struct {
	StreamID       string
	Slug           string
	Pid            int
	Status         string // idle, starting, running, error, stopped
	ErrorMessage   string
	StartedAt       time.Time
	LastActivity    time.Time
	ErrorStartedAt  time.Time
	ColdStarting    bool
	TransitionUntil time.Time
	CancelFunc      context.CancelFunc
	Logs           []models.StreamLogEntry
	mu             sync.RWMutex
}

func (ps *ProcessState) AddLog(msg string) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	// Keep max 500 entries
	if len(ps.Logs) >= 500 {
		ps.Logs = ps.Logs[1:]
	}
	ps.Logs = append(ps.Logs, models.StreamLogEntry{
		Timestamp: time.Now(),
		Message:   strings.TrimSpace(msg),
	})
}

func (ps *ProcessState) GetLogs() []models.StreamLogEntry {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	res := make([]models.StreamLogEntry, len(ps.Logs))
	copy(res, ps.Logs)
	return res
}

type Manager struct {
	repo      *repository.Repository
	processes map[string]*ProcessState
	mu        sync.RWMutex
	stopChan  chan struct{}
}

var GlobalManager *Manager

func NewManager(repo *repository.Repository) *Manager {
	m := &Manager{
		repo:      repo,
		processes: make(map[string]*ProcessState),
		stopChan:  make(chan struct{}),
	}
	GlobalManager = m
	go m.idleSweeper()
	return m
}

func (m *Manager) getOrCreateState(stream *models.Stream) *ProcessState {
	m.mu.Lock()
	defer m.mu.Unlock()

	state, exists := m.processes[stream.Slug]
	if !exists {
		state = &ProcessState{
			StreamID:     stream.ID,
			Slug:         stream.Slug,
			Status:       "idle",
			LastActivity: time.Now(),
			Logs:         make([]models.StreamLogEntry, 0, 100),
		}
		m.processes[stream.Slug] = state
	}
	return state
}

func (m *Manager) GetStreamStatus(slug string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if state, ok := m.processes[slug]; ok {
		state.mu.RLock()
		defer state.mu.RUnlock()
		return state.Status
	}
	return "idle"
}

func (m *Manager) GetStreamLogs(slug string) []models.StreamLogEntry {
	m.mu.RLock()
	state, ok := m.processes[slug]
	m.mu.RUnlock()
	if ok {
		return state.GetLogs()
	}
	return nil
}

// Touch is called whenever an IPTV player requests a playlist or TS segment.
// It atomically claims the "starting" status and sets ColdStarting before spawning
// the FFmpeg goroutine, ensuring no two goroutines ever start for the same stream.
func (m *Manager) Touch(slug string) {
	m.mu.RLock()
	state, exists := m.processes[slug]
	m.mu.RUnlock()

	if exists {
		state.mu.Lock()
		state.LastActivity = time.Now()
		status := state.Status
		state.mu.Unlock()

		if status == "running" || status == "starting" {
			return
		}
	}

	// Stream is not currently active, fetch definition and start if enabled
	stream, err := m.repo.GetStreamBySlug(slug)
	if err != nil || !stream.Enabled {
		return
	}

	state = m.getOrCreateState(stream)
	state.mu.Lock()
	state.LastActivity = time.Now()
	if state.Status == "running" || state.Status == "starting" {
		state.mu.Unlock()
		return
	}
	// Claim the starting slot atomically. Setting ColdStarting=true here means
	// IsStreamReady will correctly open a discontinuity window even if StartStream
	// bails out because another goroutine already set status="starting".
	state.Status = "starting"
	state.ColdStarting = true
	state.TransitionUntil = time.Time{}
	state.mu.Unlock()

	go m.StartStream(stream)
}

// StartStream initializes and launches FFmpeg for the given stream.
// It guards against duplicate launches from any call site (Touch, checkAutoRecover, or direct).
func (m *Manager) StartStream(stream *models.Stream) error {
	state := m.getOrCreateState(stream)

	state.mu.Lock()
	// Guard: only one goroutine may start FFmpeg at a time.
	// "starting" is set by Touch() before this goroutine runs, or by checkAutoRecover().
	// Either way, a second concurrent caller must bail out.
	if state.Status == "running" || state.Status == "starting" {
		// If this was a direct call (not via Touch), we still need to launch FFmpeg.
		// Detect: if CancelFunc is nil, no FFmpeg goroutine is actually running yet.
		if state.CancelFunc != nil {
			state.mu.Unlock()
			return nil
		}
		// No cancel func means no goroutine is running — fall through to start one.
	}
	state.Status = "starting"
	state.StartedAt = time.Now()
	state.LastActivity = time.Now()
	state.ColdStarting = true
	state.TransitionUntil = time.Time{}
	state.ErrorMessage = ""

	ctx, cancel := context.WithCancel(context.Background())
	state.CancelFunc = cancel
	state.mu.Unlock()

	state.AddLog(fmt.Sprintf("Preparing FFmpeg for stream: %s (%s)", stream.Name, stream.Slug))

	streamDir := filepath.Join(db.GetStreamsDir(), stream.Slug)
	m.cleanupStreamDir(stream.Slug)
	if err := os.MkdirAll(streamDir, 0755); err != nil {
		state.mu.Lock()
		state.Status = "error"
		state.ErrorMessage = fmt.Sprintf("Failed to create stream folder: %v", err)
		state.mu.Unlock()
		state.AddLog("ERROR: " + state.ErrorMessage)
		return err
	}

	go m.runFFmpegLoop(ctx, stream, state, streamDir)
	return nil
}

func (m *Manager) StopStream(slug string) {
	m.mu.Lock()
	state, exists := m.processes[slug]
	m.mu.Unlock()

	if !exists {
		return
	}

	state.mu.Lock()
	pid := state.Pid
	if state.CancelFunc != nil {
		state.CancelFunc()
		state.CancelFunc = nil
	}
	state.Pid = 0
	state.Status = "stopped"
	state.ColdStarting = false
	state.TransitionUntil = time.Time{}
	state.mu.Unlock()

	if pid > 0 {
		killProcessTree(pid)
	}
	killStreamProcessesBySlug(slug)

	state.AddLog("Stream stopped by user/system")
	m.cleanupStreamDir(slug)
}

func (m *Manager) cleanupStreamDir(slug string) {
	streamDir := filepath.Join(db.GetStreamsDir(), slug)
	// Give the operating system a moment to release file handles after killing FFmpeg, with retry
	for attempt := 0; attempt < 3; attempt++ {
		entries, err := os.ReadDir(streamDir)
		if err != nil {
			return
		}
		if len(entries) == 0 {
			return
		}
		allRemoved := true
		for _, entry := range entries {
			if !entry.IsDir() {
				target := filepath.Join(streamDir, entry.Name())
				if err := os.Remove(target); err != nil {
					allRemoved = false
				}
			}
		}
		if allRemoved {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func (m *Manager) runFFmpegLoop(ctx context.Context, stream *models.Stream, state *ProcessState, streamDir string) {
	retryCount := 0
	maxRetries := 5

	for {
		select {
		case <-ctx.Done():
			state.mu.Lock()
			state.Status = "idle"
			state.mu.Unlock()
			state.AddLog("Stream process stopped")
			return
		default:
		}

		args := buildFFmpegArgs(stream, streamDir)
		ffmpegBin := m.ResolveFFmpegBinary()
		state.AddLog("Executing: " + ffmpegBin + " " + strings.Join(args, " "))

		cmd := exec.CommandContext(ctx, ffmpegBin, args...)
		cmd.Cancel = func() error {
			if cmd.Process != nil && cmd.Process.Pid > 0 {
				killProcessTree(cmd.Process.Pid)
			}
			return nil
		}

		// Capture combined output for logging
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			state.AddLog(fmt.Sprintf("Failed to pipe stdout: %v", err))
		}
		stderr, err := cmd.StderrPipe()
		if err != nil {
			state.AddLog(fmt.Sprintf("Failed to pipe stderr: %v", err))
		}

		if err := cmd.Start(); err != nil {
			state.mu.Lock()
			state.Status = "error"
			state.ErrorMessage = fmt.Sprintf("Failed to start FFmpeg: %v", err)
			if state.ErrorStartedAt.IsZero() {
				state.ErrorStartedAt = time.Now()
			}
			state.mu.Unlock()
			state.AddLog("ERROR: " + state.ErrorMessage)
			return
		}

		pid := cmd.Process.Pid
		state.mu.Lock()
		state.Pid = pid
		state.Status = "running"
		state.ErrorStartedAt = time.Time{}
		state.mu.Unlock()

		// Read output logs asynchronously
		go m.pipeLogs(stdout, state)
		go m.pipeLogs(stderr, state)

		// Watch context cancellation to forcefully terminate process tree on Windows/Unix
		stopWatch := make(chan struct{})
		go func() {
			select {
			case <-ctx.Done():
				killProcessTree(pid)
			case <-stopWatch:
			}
		}()

		err = cmd.Wait()
		close(stopWatch)

		state.mu.Lock()
		state.Pid = 0
		state.mu.Unlock()

		select {
		case <-ctx.Done():
			state.mu.Lock()
			state.Status = "idle"
			state.mu.Unlock()
			return
		default:
		}

		state.mu.Lock()
		state.Status = "error"
		if state.ErrorStartedAt.IsZero() {
			state.ErrorStartedAt = time.Now()
		}
		if err != nil {
			state.ErrorMessage = fmt.Sprintf("FFmpeg exited with error: %v", err)
		} else {
			state.ErrorMessage = "FFmpeg process finished unexpectedly"
		}
		state.mu.Unlock()

		state.AddLog(state.ErrorMessage)

		// Check if we should retry
		if stream.Mode == "always_on" {
			retryCount++
			if retryCount > maxRetries {
				state.AddLog(fmt.Sprintf("Max retries (%d) reached for always-on stream. Pausing...", maxRetries))
				time.Sleep(30 * time.Second)
				retryCount = 0
			} else {
				state.AddLog(fmt.Sprintf("Retrying always-on stream in 5 seconds (attempt %d)...", retryCount))
				time.Sleep(5 * time.Second)
			}
			continue
		}

		// In on-demand mode, do not endlessly loop on error; wait for next touch
		return
	}
}

func (m *Manager) pipeLogs(reader io.ReadCloser, state *ProcessState) {
	if reader == nil {
		return
	}
	defer reader.Close()

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		text := scanner.Text()
		if text != "" {
			state.AddLog(text)
		}
	}
}

func buildFFmpegArgs(stream *models.Stream, streamDir string) []string {
	var args []string

	input := stream.MediaURL
	var inputParams []string

	fifoSize := "1000000"
	if stream.FifoSize != "" {
		fifoSize = stream.FifoSize
	}

	bufferSize := "1000000"
	if stream.BufferSize != "" {
		bufferSize = stream.BufferSize
	}

	isUDP := strings.HasPrefix(strings.ToLower(stream.MediaURL), "udp://")

	if isUDP {
		localAddr := strings.TrimSpace(stream.LocalAddr)
		if localAddr == "" {
			localAddr = strings.TrimSpace(os.Getenv("IP_ADDR"))
		}
		if localAddr != "" && !strings.Contains(stream.MediaURL, "localaddr=") {
			inputParams = append(inputParams, fmt.Sprintf("localaddr=%s", localAddr))
		}
		if !strings.Contains(stream.MediaURL, "fifo_size=") {
			inputParams = append(inputParams, fmt.Sprintf("fifo_size=%s", fifoSize))
		}
		if !strings.Contains(stream.MediaURL, "buffer_size=") {
			inputParams = append(inputParams, fmt.Sprintf("buffer_size=%s", bufferSize))
		}
		if !strings.Contains(stream.MediaURL, "overrun_nonfatal=") {
			inputParams = append(inputParams, "overrun_nonfatal=1")
		}
	} else {
		if stream.FifoSize != "" && !strings.Contains(stream.MediaURL, "fifo_size=") {
			inputParams = append(inputParams, fmt.Sprintf("fifo_size=%s", stream.FifoSize))
		}
		if stream.OverrunNonfatal && !strings.Contains(stream.MediaURL, "overrun_nonfatal=") {
			inputParams = append(inputParams, "overrun_nonfatal=1")
		}
	}

	if len(inputParams) > 0 {
		sep := "?"
		if strings.Contains(input, "?") {
			sep = "&"
		}
		input = fmt.Sprintf("%s%s%s", input, sep, strings.Join(inputParams, "&"))
	}

	// Global / input options
	args = append(args, "-hide_banner", "-loglevel", "warning")

	// Probing settings:
	// Default to 2000000 (2s / 2MB) for single-channel SPTS streams for rapid cold-start.
	// For MPTS multi-channel streams (ProgramID != "" && ProgramID != "0"), auto-scale default to
	// 5000000 (5s) and 10000000 (10MB) to allow multiplexed channels time to emit SPS/PPS.
	// Explicit stream settings (AnalyzeDuration / ProbeSize) override defaults.
	analyzeDuration := "2000000"
	probeSize := "2000000"
	if stream.ProgramID != "" && stream.ProgramID != "0" {
		analyzeDuration = "5000000"
		probeSize = "10000000"
	}
	if customAnalyze := strings.TrimSpace(stream.AnalyzeDuration); customAnalyze != "" {
		analyzeDuration = customAnalyze
	}
	if customProbe := strings.TrimSpace(stream.ProbeSize); customProbe != "" {
		probeSize = customProbe
	}

	args = append(args, "-analyzeduration", analyzeDuration, "-probesize", probeSize)

	// Socket buffer size before -i for protocols that read it via command line
	if isUDP && !strings.Contains(input, "buffer_size=") {
		args = append(args, "-buffer_size", bufferSize)
	}

	if stream.UseGPU {
		args = append(args, "-hwaccel", "cuda")
	}

	// Input URL
	args = append(args, "-i", input)

	// Stream mapping
	if stream.ProgramID != "" && stream.ProgramID != "0" {
		args = append(args, "-map", fmt.Sprintf("0:p:%s", stream.ProgramID))
	}

	// Codec copy, extradata injection for keyframe segment headers, & timestamp generation/corruption handling.
	// -bsf:v dump_extra ensures each segment beginning with a keyframe includes SPS/PPS parameter sets,
	// preventing decode errors (e.g. non-existing PPS 0, MEDIA_ERR_DECODE) on Apple AVPlayer and mobile decoders.
	args = append(args, "-c", "copy", "-bsf:v", "dump_extra", "-fflags", "+genpts+discardcorrupt")

	// Fast 3-second HLS segmenting for rapid cold-start and low-latency live playback.
	// temp_file: FFmpeg writes to a temp file then atomically renames to the final .m3u8,
	// preventing IsStreamReady from reading a partially-written playlist.
	args = append(args, "-hls_time", "3", "-hls_list_size", "10", "-hls_flags", "delete_segments+temp_file")

	segmentPattern := filepath.Join(streamDir, "segment_%03d.ts")
	playlistPath := filepath.Join(streamDir, stream.Slug+".m3u8")

	args = append(args, "-hls_segment_filename", segmentPattern, playlistPath)
	return args
}

// idleSweeper checks periodically for idle on-demand streams and auto-recovers failed streams
func (m *Manager) idleSweeper() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-m.stopChan:
			return
		case <-ticker.C:
			m.checkIdleStreams()
			m.checkAutoRecover()
		}
	}
}

func (m *Manager) checkAutoRecover() {
	m.mu.RLock()
	var errorSlugs []string
	for slug, state := range m.processes {
		state.mu.RLock()
		if state.Status == "error" && !state.ErrorStartedAt.IsZero() {
			errorSlugs = append(errorSlugs, slug)
		}
		state.mu.RUnlock()
	}
	m.mu.RUnlock()

	for _, slug := range errorSlugs {
		stream, err := m.repo.GetStreamBySlug(slug)
		if err != nil || !stream.Enabled || !stream.AutoRecover {
			continue
		}

		timeout := stream.RecoverTimeoutSec
		if timeout <= 0 {
			timeout = 30
		}

		m.mu.RLock()
		state := m.processes[slug]
		m.mu.RUnlock()

		if state != nil {
			state.mu.RLock()
			durationInError := time.Since(state.ErrorStartedAt)
			isErr := state.Status == "error"
			state.mu.RUnlock()

			if isErr && durationInError >= time.Duration(timeout)*time.Second {
				logrus.Infof("Auto-recovery: stream %s has been in error for %v. Initiating auto-restart...", slug, durationInError.Round(time.Second))
				state.AddLog(fmt.Sprintf("[Auto-Recovery] Stream in error state for %v (threshold: %ds). Attempting automated restart...", durationInError.Round(time.Second), timeout))

				state.mu.Lock()
				state.ErrorStartedAt = time.Time{}
				state.Status = "starting"
				state.mu.Unlock()

				go m.StartStream(stream)
			}
		}
	}
}

func (m *Manager) GetActiveCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	count := 0
	for _, state := range m.processes {
		state.mu.RLock()
		if state.Status == "running" || state.Status == "starting" {
			count++
		}
		state.mu.RUnlock()
	}
	return count
}

func (m *Manager) GetProblemCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	count := 0
	for _, state := range m.processes {
		state.mu.RLock()
		if state.Status == "error" {
			count++
		}
		state.mu.RUnlock()
	}
	return count
}

func (m *Manager) checkIdleStreams() {
	m.mu.RLock()
	var activeSlugs []string
	for slug, state := range m.processes {
		state.mu.RLock()
		if state.Status == "running" || state.Status == "starting" {
			activeSlugs = append(activeSlugs, slug)
		}
		state.mu.RUnlock()
	}
	m.mu.RUnlock()

	for _, slug := range activeSlugs {
		stream, err := m.repo.GetStreamBySlug(slug)
		if err != nil {
			continue
		}

		if stream.Mode != "always_on" {
			timeoutSec := stream.IdleTimeoutSec
			if timeoutSec <= 0 {
				timeoutSec = 180
			}

			m.mu.RLock()
			state := m.processes[slug]
			m.mu.RUnlock()

			if state != nil {
				state.mu.RLock()
				inactiveDuration := time.Since(state.LastActivity)
				state.mu.RUnlock()

				if inactiveDuration > time.Duration(timeoutSec)*time.Second {
					logrus.Infof("Stream %s reached idle timeout (%v), stopping FFmpeg", slug, inactiveDuration)
					state.AddLog(fmt.Sprintf("Stream reached idle timeout (%v). Stopping FFmpeg.", inactiveDuration.Round(time.Second)))
					m.StopStream(slug)
				}
			}
		}
	}
}

func (m *Manager) StartAllAlwaysOnStreams() {
	streams, err := m.repo.GetAllStreams()
	if err != nil {
		logrus.Errorf("Failed to retrieve streams for always_on init: %v", err)
		return
	}

	for _, s := range streams {
		if s.Enabled && s.Mode == "always_on" {
			logrus.Infof("Auto-starting always-on stream: %s", s.Name)
			streamCopy := s
			go func() {
				_ = m.StartStream(&streamCopy)
			}()
		}
	}
}

func (m *Manager) StopAll() {
	close(m.stopChan)
	m.mu.Lock()
	defer m.mu.Unlock()

	for slug, state := range m.processes {
		state.mu.Lock()
		pid := state.Pid
		if state.CancelFunc != nil {
			state.CancelFunc()
			state.CancelFunc = nil
		}
		state.Pid = 0
		state.Status = "stopped"
		state.mu.Unlock()

		if pid > 0 {
			killProcessTree(pid)
		}
		killStreamProcessesBySlug(slug)
		m.cleanupStreamDir(slug)
	}
}

// Slugify converts channel name to URL-safe alphanumeric slug
func Slugify(name string) string {
	reg := regexp.MustCompile("[^a-zA-Z0-9]+")
	slug := strings.ToLower(reg.ReplaceAllString(name, "-"))
	slug = strings.Trim(slug, "-")
	if slug == "" {
		slug = "stream"
	}
	return slug
}

// IsStreamReady checks whether FFmpeg has written a valid playlist with actual segments
// and at least one playable TS segment on disk. Both must be true before serving live content.
func (m *Manager) IsStreamReady(slug string) (ready bool, shouldInjectDiscontinuity bool) {
	m.mu.RLock()
	state, exists := m.processes[slug]
	m.mu.RUnlock()

	if !exists {
		return false, false
	}

	streamDir := filepath.Join(db.GetStreamsDir(), slug)
	playlistPath := filepath.Join(streamDir, slug+".m3u8")

	// Read the manifest and verify it has at least one #EXTINF entry.
	// A non-zero file that only contains the header is not yet playable.
	data, err := os.ReadFile(playlistPath)
	if err != nil || !strings.Contains(string(data), "#EXTINF") {
		return false, false
	}

	// Verify that at least one real (non-loading) TS segment exists on disk with size > 0.
	entries, err := os.ReadDir(streamDir)
	if err != nil {
		return false, false
	}

	hasTS := false
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".ts") && e.Name() != "loading.ts" {
			fi, err := e.Info()
			if err == nil && fi.Size() > 0 {
				hasTS = true
				break
			}
		}
	}

	if !hasTS {
		return false, false
	}

	// Both playlist and segments exist on disk — stream is live.
	// Manage the cold-start → live discontinuity transition window.
	state.mu.Lock()
	defer state.mu.Unlock()

	now := time.Now()
	if state.ColdStarting {
		// Very first time real segments are detected after this cold-start.
		// Open a 10-second window during which we inject #EXT-X-DISCONTINUITY
		// so players switch seamlessly from the loading bumper to the live stream.
		state.ColdStarting = false
		state.TransitionUntil = now.Add(10 * time.Second)
	}

	shouldDiscontinuity := !state.TransitionUntil.IsZero() && now.Before(state.TransitionUntil)
	return true, shouldDiscontinuity
}

// WaitForStreamReady polls until the stream has written its playlist and at least one segment, or until timeout.
// Uses a tight poll interval to detect stream readiness as fast as possible for seamless cold-start transitions.
func (m *Manager) WaitForStreamReady(slug string, timeout time.Duration) (ready bool, shouldDiscontinuity bool) {
	deadline := time.Now().Add(timeout)
	for {
		ready, disc := m.IsStreamReady(slug)
		if ready {
			return true, disc
		}
		if time.Now().After(deadline) {
			return false, false
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// GenerateDynamicLoadingPlaylist generates an RFC 8216 compliant dynamic HLS playlist pointing to the loading bumper
// with an advancing media sequence and multi-segment buffer to prevent player stall during cold start.
func (m *Manager) GenerateDynamicLoadingPlaylist(slug string) string {
	m.mu.RLock()
	state := m.processes[slug]
	m.mu.RUnlock()

	seq := 0
	if state != nil {
		state.mu.RLock()
		if !state.StartedAt.IsZero() {
			seq = int(time.Since(state.StartedAt).Seconds() / 2)
		}
		state.mu.RUnlock()
	}

	return fmt.Sprintf("#EXTM3U\n#EXT-X-VERSION:3\n#EXT-X-TARGETDURATION:2\n#EXT-X-MEDIA-SEQUENCE:%d\n#EXTINF:2.000000,\n/stream/%s/loading.ts\n#EXTINF:2.000000,\n/stream/%s/loading.ts\n#EXTINF:2.000000,\n/stream/%s/loading.ts\n", seq, slug, slug, slug)
}

// GenerateLoadingPlaylist generates a fallback HLS playlist
func (m *Manager) GenerateLoadingPlaylist() string {
	return "#EXTM3U\n#EXT-X-VERSION:3\n#EXT-X-TARGETDURATION:2\n#EXT-X-MEDIA-SEQUENCE:0\n#EXTINF:2.000000,\nloading.ts\n"
}


