package gamewrapper

import (
	"testing"

	"github.com/castaneai/mashimaro/pkg/x11"

	"golang.org/x/sync/errgroup"
)

func TestWatch(t *testing.T) {
	t.Skip("Only on manual test")

	xu := x11.NewDefaultXUtil(t)
	cmd := x11.StartWineProcess(t, xu, "notepad")
	w := newProcessWatcher()

	var eg errgroup.Group
	eg.Go(func() error {
		return w.Start(cmd)
	})
	if err := eg.Wait(); err != nil {
		t.Fatal(err)
	}
}
