package gamesession

import "context"

type Allocator interface {
	Allocate(ctx context.Context) (*GameServer, error)
}

type MockAllocator struct {
	MockedGS *GameServer
}

func (a *MockAllocator) Allocate(ctx context.Context) (*GameServer, error) {
	return a.MockedGS, nil
}

type GameServer struct {
	Addr string
}
