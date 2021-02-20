package allocator

import (
	"context"
)

type Allocator interface {
	Allocate(ctx context.Context) (*AllocatedServer, error)
}

type MockAllocator struct {
	MockedGS *AllocatedServer
}

func NewMockAllocator(mockedGS *AllocatedServer) *MockAllocator {
	return &MockAllocator{MockedGS: mockedGS}
}

func (a *MockAllocator) Allocate(ctx context.Context) (*AllocatedServer, error) {
	return a.MockedGS, nil
}

type AllocatedServer struct {
	ID string
}
