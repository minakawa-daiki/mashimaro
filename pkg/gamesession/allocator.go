package gamesession

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

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

type AgonesAllocator struct {
	addr      string
	namespace string
}

type allocationRequest struct {
	Namespace string `json:"namespace"`
}

type allocationPort struct {
	Name string `json:"streamer"`
	Port int    `json:"port"`
}

type allocationResponse struct {
	GameServerName string           `json:"gameServerName"`
	Address        string           `json:"address"`
	Ports          []allocationPort `json:"ports"`
}

func (r *allocationResponse) Addr() string {
	if len(r.Ports) < 1 {
		return fmt.Sprintf("%s:<NO_PORT_FOUND_IN_ALLOCATED_GAMESERVER>")
	}
	return fmt.Sprintf("%s:%d", r.Address, r.Ports[0].Port)
}

func (a *AgonesAllocator) Allocate(ctx context.Context) (*GameServer, error) {
	var body bytes.Buffer
	enc := json.NewEncoder(&body)
	if err := enc.Encode(&allocationRequest{Namespace: a.namespace}); err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", "http://"+a.addr+"/gameserverallocation", bytes.NewReader(body.Bytes()))
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	dec := json.NewDecoder(res.Body)
	var resp allocationResponse
	if err := dec.Decode(&resp); err != nil {
		return nil, err
	}
	return &GameServer{Addr: resp.Addr()}, nil
}

func NewAgonesAllocator(addr, namespace string) *AgonesAllocator {
	return &AgonesAllocator{addr: addr, namespace: namespace}
}
