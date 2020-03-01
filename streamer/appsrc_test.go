package streamer

import "testing"

func TestStartPollingRawOverTCP(t *testing.T) {
	if _, err := StartPollingRawOverTCP(1282, 747, 32); err != nil {
		t.Fatal(err)
	}
}
