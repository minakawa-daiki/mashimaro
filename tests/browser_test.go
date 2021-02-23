package tests

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/castaneai/mashimaro/pkg/streamer/streamerproto"

	"github.com/castaneai/mashimaro/pkg/proto"
	"github.com/castaneai/mashimaro/pkg/streamer"
	"google.golang.org/grpc"

	"github.com/stretchr/testify/assert"

	"github.com/pion/webrtc/v3/pkg/media"

	"github.com/notedit/gst"

	"github.com/pion/webrtc/v3"
	"github.com/sclevine/agouti"
)

const (
	testHtmlFile      = "test.html"
	testOggFile       = "example.ogg"
	streamerServer    = "localhost:50502"
	streamerAudioPort = 50601 // see docker-compose.yml
)

var drivers = map[string]func() *agouti.WebDriver{
	"Chrome": func() *agouti.WebDriver {
		width := 320
		height := 240
		return agouti.ChromeDriver(
			agouti.ChromeOptions("args", []string{
				// "--headless",
				"--disable-gpu",
				"--no-sandbox",
				fmt.Sprintf("--window-size=%d,%d", width, height),
			}),
			agouti.Desired(agouti.Capabilities{
				"loggingPrefs": map[string]string{
					"browser": "INFO",
				},
			}),
		)
	},
}

func TestAudio(t *testing.T) {
	for name, d := range drivers {
		driver := d()
		t.Run(name, func(t *testing.T) {
			if err := driver.Start(); err != nil {
				t.Fatalf("Failed to start WebDriver: %v", err)
			}
			t.Cleanup(func() { _ = driver.Stop() })
			page, errPage := driver.NewPage()
			if errPage != nil {
				t.Fatalf("Failed to open page: %v", errPage)
			}
			if err := page.SetPageLoad(1000); err != nil {
				t.Fatalf("Failed to load page: %v", err)
			}
			if err := page.SetImplicitWait(1000); err != nil {
				t.Fatalf("Failed to set wait: %v", err)
			}

			chSDP := make(chan *webrtc.SessionDescription)
			chStarted := make(chan struct{})
			go logParseLoop(context.Background(), t, page, chStarted, chSDP)

			pwd, errPwd := os.Getwd()
			if errPwd != nil {
				t.Fatalf("Failed to get working directory: %v", errPwd)
			}
			if err := page.Navigate(
				fmt.Sprintf("file://%s/%s", pwd, testHtmlFile),
			); err != nil {
				t.Fatalf("Failed to navigate: %v", err)
			}

			sdp := <-chSDP
			pc, answer, track, errTrack := createTrack(*sdp)
			if errTrack != nil {
				t.Fatalf("Failed to create track: %v", errTrack)
			}
			defer func() {
				_ = pc.Close()
			}()

			answerBytes, errAnsSDP := json.Marshal(answer)
			if errAnsSDP != nil {
				t.Fatalf("Failed to marshal SDP: %v", errAnsSDP)
			}
			var result string
			if err := page.RunScript(
				"pc.setRemoteDescription(new RTCSessionDescription(JSON.parse(answer)))",
				map[string]interface{}{"answer": string(answerBytes)},
				&result,
			); err != nil {
				t.Fatalf("Failed to run script to set SDP: %v", err)
			}
			assert.NoError(t, page.Click(agouti.SingleClick, agouti.LeftButton)) // to avoid "play() failed because the user didn't interact with the document first"

			ctx := context.Background()
			go startPushPulseAudioFromStreamer(ctx, t, track, streamerServer)
			select {}
		})
	}
}

func startPushOggFile(t *testing.T, track *webrtc.TrackLocalStaticSample, oggFile string) {
	p, err := gst.ParseLaunch(fmt.Sprintf("filesrc location=%s ! oggdemux ! vorbisdec ! audioconvert ! audioresample ! opusenc ! appsink name=out", oggFile))
	if err != nil {
		panic(fmt.Errorf("failed to parse pipeline: %+v", err))
	}
	out := p.GetByName("out")
	p.SetState(gst.StatePlaying)
	t.Cleanup(func() { p.SetState(gst.StateNull) })
	for {
		sample, err := out.PullSample()
		if err != nil {
			panic(fmt.Errorf("failed to pull sample from gst appsink: %+v", err))
		}
		if err := track.WriteSample(media.Sample{
			Data:     sample.Data,
			Duration: time.Duration(sample.Duration),
		}); err != nil {
			panic(fmt.Errorf("failed to write sample to track: %+v", err))
		}
	}
}

func startPushPulseAudioFromStreamer(ctx context.Context, t *testing.T, track *webrtc.TrackLocalStaticSample, streamerAddr string) {
	pulse := streamer.NewPulseAudioCapturer("localhost:4713")
	p := streamer.NewOpusEncoder(pulse)
	gstPipeline, err := p.CompileGstPipeline()
	if err != nil {
		panic(fmt.Errorf("failed to complie gst pipeline: %+v", err))
	}
	cc, err := grpc.Dial(streamerAddr, grpc.WithInsecure())
	if err != nil {
		panic(fmt.Errorf("failed to dial to streamer: %+v", err))
	}
	sc := proto.NewStreamerClient(cc)
	resp, err := sc.StartAudioStreaming(ctx, &proto.StartAudioStreamingRequest{
		GstPipeline: gstPipeline,
		Port:        streamerAudioPort,
	})
	if err != nil {
		panic(fmt.Errorf("failed to start audio streaming: %+v", err))
	}
	audioAddr := fmt.Sprintf("%s:%d", strings.Split(streamerAddr, ":")[0], resp.ListenPort)
	conn, err := net.Dial("tcp", audioAddr)
	if err != nil {
		panic(fmt.Errorf("failed to dial to audio streamer: %+v", err))
	}
	r := bufio.NewReader(conn)
	for {
		var sp streamerproto.SamplePacket
		if err := streamerproto.ReadSamplePacket(r, &sp); err != nil {
			panic(fmt.Errorf("failed to read sample packet: %+v", err))
		}
		if err := track.WriteSample(media.Sample{
			Data:     sp.Data,
			Duration: sp.Duration,
		}); err != nil {
			panic(fmt.Errorf("failed to write sample: %+v", err))
		}
	}
}

func logParseLoop(ctx context.Context, t *testing.T, page *agouti.Page, chStarted chan struct{}, chSDP chan *webrtc.SessionDescription) {
	for {
		select {
		case <-time.After(time.Second):
		case <-ctx.Done():
			return
		}
		logs, errLog := page.ReadNewLogs("browser")
		if errLog != nil {
			t.Errorf("Failed to read log: %v", errLog)
			return
		}
		for _, log := range logs {
			k, v, ok := parseLog(log)
			if !ok {
				t.Log(log.Message)
				continue
			}
			switch k {
			case "connection":
				switch v {
				case "connected":
					close(chStarted)
				case "failed":
					t.Error("Browser reported connection failed")
					return
				}
			case "sdp":
				sdp := &webrtc.SessionDescription{}
				if err := json.Unmarshal([]byte(v), sdp); err != nil {
					t.Errorf("Failed to unmarshal SDP: %v", err)
					return
				}
				chSDP <- sdp
			case "stats":
				// TODO: stats
			default:
				t.Log(log.Message)
			}
		}
	}
}

func parseLog(log agouti.Log) (string, string, bool) {
	l := strings.SplitN(log.Message, " ", 4)
	if len(l) != 4 {
		return "", "", false
	}
	k, err1 := strconv.Unquote(l[2])
	if err1 != nil {
		return "", "", false
	}
	v, err2 := strconv.Unquote(l[3])
	if err2 != nil {
		return "", "", false
	}
	return k, v, true
}

func createTrack(offer webrtc.SessionDescription) (*webrtc.PeerConnection, *webrtc.SessionDescription, *webrtc.TrackLocalStaticSample, error) {
	pc, errPc := webrtc.NewPeerConnection(webrtc.Configuration{})
	if errPc != nil {
		return nil, nil, nil, errPc
	}

	track, errTrack := webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{MimeType: "audio/opus"}, "audio", "pion")
	if errTrack != nil {
		return nil, nil, nil, errTrack
	}
	if _, err := pc.AddTrack(track); err != nil {
		return nil, nil, nil, err
	}
	if err := pc.SetRemoteDescription(offer); err != nil {
		return nil, nil, nil, err
	}
	answer, errAns := pc.CreateAnswer(nil)
	if errAns != nil {
		return nil, nil, nil, errAns
	}
	if err := pc.SetLocalDescription(answer); err != nil {
		return nil, nil, nil, err
	}
	return pc, &answer, track, nil
}
