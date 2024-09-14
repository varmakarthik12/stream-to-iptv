package ffmpeg

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"stream-to-iptv/pkg/stream"
	"stream-to-iptv/pkg/utils"

	"github.com/sirupsen/logrus"
)

type FFmpegConfig struct {
	LocalAddr string
}

// startFFmpeg starts an FFmpeg process for a given stream
func StartFFmpeg(stream stream.Stream, config FFmpegConfig) error {
	input := stream.Media
	if config.LocalAddr != "" {
		input = fmt.Sprintf("%s?localaddr=%s", stream.Media, config.LocalAddr)
	}

	ffmpegCmd := exec.Command("ffmpeg", "-i", input)
	ffmpegCmd.Args = append(ffmpegCmd.Args, "-fflags", "+genpts")
	ffmpegCmd.Args = append(ffmpegCmd.Args, "-buffer_size", utils.GetBufferSize())
	ffmpegCmd.Args = append(ffmpegCmd.Args, "-map", fmt.Sprintf("0:p:%s", stream.ProgramId))
	ffmpegCmd.Args = append(ffmpegCmd.Args, "-c:v", "copy", "-c:a", "copy")
	ffmpegCmd.Args = append(ffmpegCmd.Args, "-hls_time", utils.MaxSegmentTime(), "-hls_list_size", utils.MaxSegmentsCount(), "-hls_flags", "delete_segments",
		"-hls_segment_filename", filepath.Join(utils.GetStreamPath(utils.GetBaseFolder(), stream.Name), `segment_%03d.ts`),
		filepath.Join(utils.GetStreamPath(utils.GetBaseFolder(), stream.Name), utils.GetStreamFileName(stream.Name)))

	logrus.Infof("Triggered: %s", ffmpegCmd.String())

	if err := ffmpegCmd.Start(); err != nil {
		return fmt.Errorf("failed to start FFmpeg for stream %s: %v", stream.Name, err)
	}
	logrus.Infof("Started FFmpeg for stream %s", stream.Name)
	return nil
}
