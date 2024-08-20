package stream

import (
	"fmt"
	"github.com/charmbracelet/log"
	"github.com/pion/mediadevices"
	"github.com/pion/mediadevices/pkg/codec/opus"
	"github.com/pion/mediadevices/pkg/codec/x264"
	"github.com/pion/mediadevices/pkg/frame"
	"github.com/pion/mediadevices/pkg/prop"
	"github.com/pion/webrtc/v3"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	_ "github.com/pion/mediadevices/pkg/driver/screen"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error("Failed to upgrade connection", "err", err)
		return
	}
	defer conn.Close()

	x264Params, err := x264.NewParams()
	if err != nil {
		panic(err)
	}
	x264Params.BitRate = 500_000 // 500kbps

	opusParams, err := opus.NewParams()
	if err != nil {
		panic(err)
	}
	codecSelector := mediadevices.NewCodecSelector(mediadevices.WithVideoEncoders(&x264Params), mediadevices.WithAudioEncoders(&opusParams))

	mediaEngine := webrtc.MediaEngine{}
	codecSelector.Populate(&mediaEngine)

	s, err := mediadevices.GetDisplayMedia(mediadevices.MediaStreamConstraints{
		Video: func(c *mediadevices.MediaTrackConstraints) {
			c.FrameFormat = prop.FrameFormat(frame.FormatI420)
			c.Width = prop.Int(640)
			c.Height = prop.Int(480)
		},
		Audio: func(c *mediadevices.MediaTrackConstraints) {
		},
		Codec: codecSelector,
	})
	if err != nil {
		panic(err)
	}

	for _, track := range s.GetTracks() {
		track.OnEnded(func(err error) {
			fmt.Printf("Track (ID: %s) ended with error: %v\n", track.ID(), err)
		})
	}

	// read data from the track and send it via the websocket
	for {
		wg := sync.WaitGroup{}
		for _, track := range s.GetTracks() {
			wg.Add(1)
			go func() {
				defer wg.Done()

				if track.Kind() == webrtc.RTPCodecTypeVideo {
					readCloser, err := track.NewEncodedReader("h264")
					if err != nil {
						log.Error("Failed to create reader", "err", err)
						return
					}
					defer readCloser.Close()

					encoded, release, err := readCloser.Read()
					if err != nil {
						log.Error("Failed to read", "err", err)
						return
					}
					defer release()

					err = conn.WriteMessage(websocket.BinaryMessage, encoded.Data)
				}
			}()
		}
		wg.Wait()
	}
}
