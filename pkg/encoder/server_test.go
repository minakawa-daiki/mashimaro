package encoder

import (
	"context"
	"fmt"
	"log"
	"net"
	"testing"
	"time"

	"github.com/castaneai/mashimaro/pkg/encoder/encoderproto"
	"github.com/castaneai/mashimaro/pkg/proto"
	"google.golang.org/grpc"

	"github.com/stretchr/testify/assert"

	"github.com/castaneai/mashimaro/pkg/testutils"
)

func TestEncoderServer(t *testing.T) {
	lis := testutils.ListenTCPWithRandomPort(t)
	s := grpc.NewServer()
	proto.RegisterEncoderServer(s, NewEncoderServer())
	go func() {
		if err := s.Serve(lis); err != nil {
			log.Printf("failed to serve gRPC server: %+v", err)
		}
	}()

	for i := 0; i < 3; i++ {
		cc, err := grpc.Dial(lis.Addr().String(), grpc.WithInsecure())
		assert.NoError(t, err)
		c := proto.NewEncoderClient(cc)
		ctx := context.Background()
		resp, err := c.StartEncoding(ctx, &proto.StartEncodingRequest{
			PipelineId:  "video",
			GstPipeline: "videotestsrc",
			Port:        0,
		})
		assert.NoError(t, err)
		conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", resp.ListenPort))
		assert.NoError(t, err)
		var sp encoderproto.SamplePacket
		assert.NoError(t, encoderproto.ReadSamplePacket(conn, &sp))
		assert.True(t, sp.Duration > 0)
		assert.True(t, len(sp.Data) > 0)
		time.Sleep(500 * time.Millisecond)
		assert.NoError(t, cc.Close())
	}
}
