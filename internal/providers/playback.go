package providers

import (
	"fmt"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"streamtui/internal/domain"
)

type WebtorrentPlaybackAdapter struct{}

func NewWebtorrentPlaybackAdapter() WebtorrentPlaybackAdapter {
	return WebtorrentPlaybackAdapter{}
}

func (adapter WebtorrentPlaybackAdapter) PlayMagnet(request domain.PlaybackRequest) error {
	args, err := BuildWebtorrentArgs(request)
	if err != nil {
		return err
	}
	command := exec.Command("webtorrent", args...)
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := command.Start(); err != nil {
		return err
	}
	return waitForImmediateExit(command, 500*time.Millisecond)
}

func waitForImmediateExit(command *exec.Cmd, gracePeriod time.Duration) error {
	done := make(chan error, 1)
	go func() {
		done <- command.Wait()
	}()

	select {
	case err := <-done:
		if err != nil {
			return fmt.Errorf("webtorrent exited immediately: %w", err)
		}
		return fmt.Errorf("webtorrent exited immediately")
	case <-time.After(gracePeriod):
		return nil
	}
}

func BuildWebtorrentArgs(request domain.PlaybackRequest) ([]string, error) {
	if err := domain.ValidateTorrentID(request.Magnet); err != nil {
		return nil, err
	}
	if !request.VLC && request.Device == "" {
		return nil, fmt.Errorf("device is required for Chromecast playback")
	}

	args := []string{request.Magnet}
	if request.VLC {
		args = append(args, "--vlc", "--not-on-top", "-s", fmt.Sprintf("%d", request.FileIndex))
		playerArgs := vlcPlayerArgs(request)
		if playerArgs != "" {
			args = append(args, "--player-args="+playerArgs)
		}
		return args, nil
	}

	args = append(args, "--chromecast", request.Device, "--not-on-top", "-s", fmt.Sprintf("%d", request.FileIndex))
	if request.SubtitleFile != "" {
		args = append(args, "-t", request.SubtitleFile)
	}
	return args, nil
}

func vlcPlayerArgs(request domain.PlaybackRequest) string {
	args := []string{}
	if request.SubtitleFile != "" {
		args = append(args, "--sub-file="+request.SubtitleFile)
	}
	if request.SubtitleDelay != "" {
		args = append(args, "--sub-delay="+request.SubtitleDelay)
	}
	return strings.Join(args, " ")
}
