package streamer

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"

	"github.com/notedit/gstreamer-go"
)

func StartPollingRawOverTCP(width, height, depth int) (<-chan []byte, error) {
	pstr := fmt.Sprintf("appsrc name=rawtcp format=time is-live=true do-timestamp=true ! videoconvert ! vp8enc ! avmux_ivf ! appsink name=out")
	log.Printf("pipeline: %s", pstr)
	pipeline, err := gstreamer.New(pstr)
	if err != nil {
		log.Fatalf("failed to create gst pipeline: %+v", err)
	}
	appsrc := pipeline.FindElement("rawtcp")
	appsrc.SetCap(fmt.Sprintf("video/x-raw,format=BGRA,width=%d,height=%d,bpp=%d,depth=%d", width, height, depth, depth))

	lis, err := net.Listen("tcp", "0.0.0.0:9999")
	if err != nil {
		return nil, fmt.Errorf("failed to listen: %+v", err)
	}
	log.Printf("listening...")
	go func() {
		for {
			conn, err := lis.Accept()
			if err != nil {
				log.Printf("failed to listen: %+v", err)
				continue
			}
			log.Printf("new connection established")
			for {
				lenbuf := make([]byte, 4)
				if _, err := io.ReadFull(conn, lenbuf); err != nil {
					log.Printf("failed to read len: %+v", err)
					break
				}
				plen := int(binary.LittleEndian.Uint32(lenbuf))
				payload := make([]byte, plen)
				if _, err := io.ReadFull(conn, payload); err != nil {
					log.Printf("failed to read payload: %+v", err)
					break
				}
				if len(payload) != width*height*4 {
					log.Printf("payload len mismatch with %d*%d*%d=%d", width, height, 4, width*height*4)
					break
				}
				appsrc.Push(payload)
			}
		}
	}()
	oute := pipeline.FindElement("out")
	out := oute.Poll()
	pipeline.Start()
	return out, nil
}
