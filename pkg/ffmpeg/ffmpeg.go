package ffmpeg

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"stream-to-iptv/pkg/stream"
	"stream-to-iptv/pkg/utils"
	"time"

	"github.com/sirupsen/logrus"
)

type FFmpegConfig struct {
	LocalAddr string
}

type Retry struct {
	RetryCount int
	LastRetry  time.Time
}

var retryMem = make(map[string]Retry)

// startFFmpeg starts an FFmpeg process for a given stream
func StartFFmpeg(stream stream.Stream, config FFmpegConfig) error {
	input := stream.Media
	if config.LocalAddr != "" {
		input = fmt.Sprintf("%s?localaddr=%s", stream.Media, config.LocalAddr)
	}

	ffmpegCmd := exec.Command("ffmpeg")
	if utils.GetUseGPU() {
		logrus.Infof("Using GPU for stream %s", stream.Name)
		ffmpegCmd.Args = append(ffmpegCmd.Args, "-hwaccel", "cuda")
	}
	ffmpegCmd.Args = append(ffmpegCmd.Args, "-hide_banner", "-loglevel", "error")
	ffmpegCmd.Args = append(ffmpegCmd.Args, "-i", input)
	// ffmpegCmd.Args = append(ffmpegCmd.Args, "-fflags", "+genpts")
	ffmpegCmd.Args = append(ffmpegCmd.Args, "-buffer_size", utils.GetBufferSize())
	ffmpegCmd.Args = append(ffmpegCmd.Args, "-map", fmt.Sprintf("0:p:%s", stream.ProgramId))
	ffmpegCmd.Args = append(ffmpegCmd.Args, "-c", "copy")
	ffmpegCmd.Args = append(ffmpegCmd.Args, "-hls_time", utils.MaxSegmentTime(), "-hls_list_size", utils.MaxSegmentsCount(), "-hls_flags", "delete_segments",
		"-hls_segment_filename", filepath.Join(utils.GetStreamPath(utils.GetBaseFolder(), stream.Name), `segment_%03d.ts`),
		filepath.Join(utils.GetStreamPath(utils.GetBaseFolder(), stream.Name), utils.GetStreamFileName(stream.Name)))

	logrus.Infof("Triggered: %s", ffmpegCmd.String())

	if err := ffmpegCmd.Start(); err != nil {
		return fmt.Errorf("failed to start FFmpeg for stream %s: %v", stream.Name, err)
	}
	logrus.Infof("Started FFmpeg for stream %s", stream.Name)

	err := ffmpegCmd.Wait()

	logrus.Errorf("FFmpeg for stream %s exited with error: %v \n Retrying...", stream.Name, err)
	currentRetry, notExist := retryMem[stream.Name]
	if !notExist {
		retryMem[stream.Name] = Retry{RetryCount: 1, LastRetry: time.Now()}
	} else {
		if time.Since(currentRetry.LastRetry) > utils.GetRetryCleanInterval() {
			retryMem[stream.Name] = Retry{RetryCount: 1, LastRetry: time.Now()}
		} else if currentRetry.RetryCount > utils.GetMaxRetries() {
			logrus.Errorf("Max retries reached for stream %s. Exiting...", stream.Name)
			return nil
		} else {
			retryMem[stream.Name] = Retry{RetryCount: currentRetry.RetryCount + 1, LastRetry: time.Now()}
		}
	}
	logrus.Infof("Waiting for %s before retrying stream %s", utils.GetRetryWaitTime(), stream.Name)
	time.Sleep(utils.GetRetryWaitTime())
	logrus.Infof("Retrying stream %s, Retries %d", stream.Name, retryMem[stream.Name].RetryCount)
	StartFFmpeg(stream, config)

	return nil
}
