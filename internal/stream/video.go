package stream

import (
	"bufio"
	"errors"
	"github.com/charmbracelet/log"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v4"
	"github.com/pion/webrtc/v4/pkg/media"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

func StartVideoStream(peerConnection *webrtc.PeerConnection) (*sync.WaitGroup, error) {
	// Create a video track
	videoTrack, err := webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeVP8}, "video", "pion")
	if err != nil {
		return nil, errors.Join(errors.New("failed to create video track"), err)
	}

	// Add the track to the peer connection
	rtpSender, err := peerConnection.AddTrack(videoTrack)
	if err != nil {
		return nil, errors.Join(errors.New("failed to add video track to peer connection"), err)
	}

	// Set up FFmpeg command
	ffmpegCmd := exec.Command("ffmpeg",
		"-f", "rawvideo",
		"-pixel_format", "rgb24",
		"-video_size", "640x480",
		"-framerate", "30",
		"-i", "pipe:0",
		"-c:v", "libvpx",
		"-b:v", "1M",
		"-deadline", "realtime",
		"-cpu-used", "4",
		"-auto-alt-ref", "0",
		"-f", "rtp",
		"rtp://127.0.0.1:5004")

	ffmpegIn, _ := ffmpegCmd.StdinPipe()
	ffmpegOut, _ := ffmpegCmd.StdoutPipe()
	ffmpegErr := strings.Builder{}
	ffmpegCmd.Stderr = &ffmpegErr
	if err := ffmpegCmd.Start(); err != nil {
		return nil, errors.Join(errors.New("failed to start ffmpeg"), err)
	}

	running := atomic.Bool{}
	running.Store(true)

	go func() {
		// wait for ffmpeg to finish
		if err := ffmpegCmd.Wait(); err != nil {
			log.Error("ffmpeg exited with error", "err", err)
		}
		running.Store(false)
		if ffmpegErr.Len() > 0 {
			log.Error("ffmpeg wrote to stderr", "content", ffmpegErr.String())
		}
	}()

	go func() {
		reader := bufio.NewReader(ffmpegOut)
		for running.Load() {
			// Read RTP packets from FFmpeg
			packet := &rtp.Packet{}
			buffer, err := reader.ReadBytes('\n')
			if err != nil {
				//log.Error("Failed to read from FFmpeg", "err", err)
				continue
			}

			if err := packet.Unmarshal(buffer); err != nil {
				//log.Error("Failed to unmarshal RTP packet", "err", err)
				continue
			}

			// Create a media.Sample from the RTP packet
			sample := media.Sample{
				Data:     packet.Payload,
				Duration: time.Millisecond * 33, // Assuming 30 fps
			}

			// Write the packet to the video track
			if err := videoTrack.WriteSample(sample); err != nil {
				log.Error("Failed to write to video track", "err", err)
			}
		}
	}()

	// Your main loop to send frames to FFmpeg
	go func() {
		for running.Load() {
			start := time.Now()
			// Generate or obtain your frame data
			frameData := getNextFrame()

			// Write frame data to FFmpeg's stdin
			if _, err := ffmpegIn.Write(frameData); err != nil {
				log.Error("Failed to write frame to FFmpeg", "err", err)
			}
			timeSpend := time.Since(start)
			sleep := time.Second/30 - timeSpend
			if sleep > 0 {
				time.Sleep(sleep)
			} else {
				log.Warn("Frame generation is too slow", "timeSpend", timeSpend)
			}
		}
	}()

	wg := &sync.WaitGroup{}
	wg.Add(1)

	go func() {
		wg.Wait()
		running.Store(false)
		<-time.After(time.Second)
		if err := ffmpegCmd.Process.Kill(); err != nil {
			log.Error("Failed to kill ffmpeg", "err", err)
		}
		if err := peerConnection.RemoveTrack(rtpSender); err != nil {
			log.Error("Failed to remove video track from peer connection", "err", err)
		}
	}()

	return wg, nil
}

func getNextFrame() []byte {
	// Create a simple color gradient (now in RGB format)
	frame := make([]byte, 640*480*3)
	for y := 0; y < 480; y++ {
		for x := 0; x < 640; x++ {
			i := (y*640 + x) * 3
			frame[i] = uint8(x * 255 / 640)   // R
			frame[i+1] = uint8(y * 255 / 480) // G
			frame[i+2] = 128                  // B
		}
	}
	return frame
}
