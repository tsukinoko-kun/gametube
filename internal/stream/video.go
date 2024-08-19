package stream

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/charmbracelet/log"
	"github.com/kbinani/screenshot"
	"github.com/pion/webrtc/v4"
	"github.com/pion/webrtc/v4/pkg/media"
	"image"
	"io"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
	"time"
)

func StartVideoStream(peerConnection *webrtc.PeerConnection) (*sync.WaitGroup, error) {
	// Create a video track
	videoTrack, err := webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeVP8}, "video", "pion")
	if err != nil {
		return nil, errors.Join(errors.New("failed to create video track"), err)
	} else {
		log.Debug("created video track")
	}

	// Add the track to the peer connection
	rtpSender, err := peerConnection.AddTrack(videoTrack)
	if err != nil {
		log.Fatalf("failed to add video track to peer connection: %v", err)
		return nil, errors.Join(errors.New("failed to add video track to peer connection"), err)
	} else {
		log.Debug("added video track to peer connection")
	}

	var resolution string
	if resVar, ok := os.LookupEnv("RESOLUTION"); ok {
		resolution = resVar
	} else {
		rect := screenshot.GetDisplayBounds(0)
		resolution = fmt.Sprintf("%dx%d", rect.Dx(), rect.Dy())
	}

	// Set up FFmpeg command
	ffmpegCmd := exec.Command("ffmpeg",
		"-f", "rawvideo",
		"-pixel_format", "rgb24",
		"-video_size", resolution,
		"-framerate", "30",
		"-i", "pipe:0",
		"-c:v", "libvpx",
		"-deadline", "realtime",
		"-cpu-used", "4",
		"-b:v", "1M",
		"-maxrate", "1M",
		"-bufsize", "2M",
		"-qmin", "4",
		"-qmax", "48",
		"-keyint_min", "30",
		"-g", "30",
		"-sc_threshold", "0",
		"-error-resilient", "1",
		"-auto-alt-ref", "0",
		"-lag-in-frames", "0",
		"-an",
		"-f", "webm",
		"-dash", "1",
		"pipe:1")
	ffmpegIn, _ := ffmpegCmd.StdinPipe()
	ffmpegOut, _ := ffmpegCmd.StdoutPipe()
	ffmpegErr, _ := ffmpegCmd.StderrPipe()

	log.Printf("starting ffmpeg with command: %v", ffmpegCmd.Args)

	if err := ffmpegCmd.Start(); err != nil {
		return nil, errors.Join(errors.New("failed to start ffmpeg"), err)
	}

	// Log FFmpeg stderr output
	go func() {
		scanner := bufio.NewScanner(ffmpegErr)
		for scanner.Scan() {
			log.Debug("ffmpeg: " + scanner.Text())
		}
	}()

	running := atomic.Bool{}
	running.Store(true)

	sampleCounter := atomic.Int32{}

	go func() {
		reader := bufio.NewReader(ffmpegOut)
		for running.Load() {
			buffer := make([]byte, 4096)
			n, err := reader.Read(buffer)
			if err != nil {
				if err == io.EOF {
					log.Warn("ffmpeg stream ended")
					break
				}
				log.Error("failed to read from ffmpeg", "err", err)
				continue
			} else {
				log.Debug("read successfully from ffmpeg", "n", n)
			}

			// Create a media.Sample from the encoded data
			sample := media.Sample{
				Data:     buffer[:n],
				Duration: time.Millisecond * 33, // Assuming 30 fps
			}

			// Write the sample to the video track
			if err := videoTrack.WriteSample(sample); err != nil {
				log.Error("failed to write to video track", "err", err)
			} else {
				log.Debug("wrote sample to video track", "size", len(sample.Data))
				sampleCounter.Add(1)
			}
		}
	}()

	go func() {
		sampleCounter.Store(0)
		ticker := time.NewTicker(5 * time.Second)
		for range ticker.C {
			log.Printf("Video streaming stats: Samples written: %d", sampleCounter.Load())
			sampleCounter.Store(0)
		}
	}()

	// Your main loop to send frames to FFmpeg
	go func() {
		for running.Load() {
			frameData := getNextFrame()

			// Write frame data to FFmpeg's stdin
			if n, err := ffmpegIn.Write(frameData); err != nil {
				log.Error("failed to write frame to ffmpeg", "err", err)
			} else {
				log.Debug("wrote frame to ffmpeg", "n", n)
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
			log.Error("failed to kill ffmpeg", "err", err)
		}
		if err := peerConnection.RemoveTrack(rtpSender); err != nil {
			log.Warn("failed to remove video track from peer connection", "err", err)
		}
	}()

	return wg, nil
}

func getNextFrame() []byte {
	img, err := screenshot.CaptureDisplay(0)
	if err != nil {
		log.Error("failed to capture display", "err", err)
		return []byte{}
	}

	return rgbaToRgb(img)
}

func rgbaToRgb(img *image.RGBA) []byte {
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	rgb := make([]byte, width*height*3)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			i := (y-bounds.Min.Y)*width + (x - bounds.Min.X)
			rgb[i*3] = byte(r >> 8)
			rgb[i*3+1] = byte(g >> 8)
			rgb[i*3+2] = byte(b >> 8)
		}
	}
	return rgb
}
