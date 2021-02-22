package streamerproto

import (
	"encoding/binary"
	"io"
	"time"

	"github.com/pkg/errors"
)

type SamplePacket struct {
	Data     []byte
	Duration time.Duration
}

func ReadSamplePacket(r io.Reader, p *SamplePacket) error {
	bufLen := make([]byte, 4)
	if _, err := io.ReadFull(r, bufLen); err != nil {
		return err
	}
	buf := make([]byte, binary.LittleEndian.Uint32(bufLen))
	if _, err := io.ReadFull(r, buf); err != nil {
		return err
	}
	p.Duration = time.Duration(binary.LittleEndian.Uint64(buf[:8]))
	p.Data = buf[8:]
	return nil
}

func WriteSamplePacket(w io.Writer, p *SamplePacket) error {
	buf := make([]byte, 4+8+len(p.Data))
	binary.LittleEndian.PutUint32(buf, uint32(len(buf)-4))
	binary.LittleEndian.PutUint64(buf[4:], uint64(p.Duration))
	if copied := copy(buf[4+8:], p.Data); copied != len(p.Data) {
		return errors.New("invalid cooy")
	}
	if _, err := w.Write(buf); err != nil {
		return errors.Wrap(err, "failed to write buffer")
	}
	return nil
}
