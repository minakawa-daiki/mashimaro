// Code generated by protoc-gen-go. DO NOT EDIT.
// source: proto/broker.proto

/*
Package proto is a generated protocol buffer package.

It is generated from these files:
	proto/broker.proto
	proto/gameprocess.proto

It has these top-level messages:
	WatchSessionRequest
	WatchSessionResponse
	DeleteSessionRequest
	DeleteSessionResponse
	Session
	GetGameMetadataRequest
	GetGameMetadataResponse
	GameMetadata
	StartGameRequest
	StartGameResponse
	ExitGameRequest
	ExitGameResponse
*/
package proto

import proto1 "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

import (
	context "golang.org/x/net/context"
	grpc "google.golang.org/grpc"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto1.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto1.ProtoPackageIsVersion2 // please upgrade the proto package

type WatchSessionRequest struct {
	AllocatedServerId string `protobuf:"bytes,1,opt,name=allocated_server_id,json=allocatedServerId" json:"allocated_server_id,omitempty"`
}

func (m *WatchSessionRequest) Reset()                    { *m = WatchSessionRequest{} }
func (m *WatchSessionRequest) String() string            { return proto1.CompactTextString(m) }
func (*WatchSessionRequest) ProtoMessage()               {}
func (*WatchSessionRequest) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

func (m *WatchSessionRequest) GetAllocatedServerId() string {
	if m != nil {
		return m.AllocatedServerId
	}
	return ""
}

type WatchSessionResponse struct {
	Found   bool     `protobuf:"varint,1,opt,name=found" json:"found,omitempty"`
	Session *Session `protobuf:"bytes,2,opt,name=session" json:"session,omitempty"`
}

func (m *WatchSessionResponse) Reset()                    { *m = WatchSessionResponse{} }
func (m *WatchSessionResponse) String() string            { return proto1.CompactTextString(m) }
func (*WatchSessionResponse) ProtoMessage()               {}
func (*WatchSessionResponse) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{1} }

func (m *WatchSessionResponse) GetFound() bool {
	if m != nil {
		return m.Found
	}
	return false
}

func (m *WatchSessionResponse) GetSession() *Session {
	if m != nil {
		return m.Session
	}
	return nil
}

type DeleteSessionRequest struct {
	SessionId         string `protobuf:"bytes,1,opt,name=session_id,json=sessionId" json:"session_id,omitempty"`
	AllocatedServerId string `protobuf:"bytes,2,opt,name=allocated_server_id,json=allocatedServerId" json:"allocated_server_id,omitempty"`
}

func (m *DeleteSessionRequest) Reset()                    { *m = DeleteSessionRequest{} }
func (m *DeleteSessionRequest) String() string            { return proto1.CompactTextString(m) }
func (*DeleteSessionRequest) ProtoMessage()               {}
func (*DeleteSessionRequest) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{2} }

func (m *DeleteSessionRequest) GetSessionId() string {
	if m != nil {
		return m.SessionId
	}
	return ""
}

func (m *DeleteSessionRequest) GetAllocatedServerId() string {
	if m != nil {
		return m.AllocatedServerId
	}
	return ""
}

type DeleteSessionResponse struct {
}

func (m *DeleteSessionResponse) Reset()                    { *m = DeleteSessionResponse{} }
func (m *DeleteSessionResponse) String() string            { return proto1.CompactTextString(m) }
func (*DeleteSessionResponse) ProtoMessage()               {}
func (*DeleteSessionResponse) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{3} }

type Session struct {
	SessionId         string `protobuf:"bytes,1,opt,name=session_id,json=sessionId" json:"session_id,omitempty"`
	AllocatedServerId string `protobuf:"bytes,2,opt,name=allocated_server_id,json=allocatedServerId" json:"allocated_server_id,omitempty"`
	GameId            string `protobuf:"bytes,3,opt,name=game_id,json=gameId" json:"game_id,omitempty"`
}

func (m *Session) Reset()                    { *m = Session{} }
func (m *Session) String() string            { return proto1.CompactTextString(m) }
func (*Session) ProtoMessage()               {}
func (*Session) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{4} }

func (m *Session) GetSessionId() string {
	if m != nil {
		return m.SessionId
	}
	return ""
}

func (m *Session) GetAllocatedServerId() string {
	if m != nil {
		return m.AllocatedServerId
	}
	return ""
}

func (m *Session) GetGameId() string {
	if m != nil {
		return m.GameId
	}
	return ""
}

type GetGameMetadataRequest struct {
	GameId string `protobuf:"bytes,1,opt,name=game_id,json=gameId" json:"game_id,omitempty"`
}

func (m *GetGameMetadataRequest) Reset()                    { *m = GetGameMetadataRequest{} }
func (m *GetGameMetadataRequest) String() string            { return proto1.CompactTextString(m) }
func (*GetGameMetadataRequest) ProtoMessage()               {}
func (*GetGameMetadataRequest) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{5} }

func (m *GetGameMetadataRequest) GetGameId() string {
	if m != nil {
		return m.GameId
	}
	return ""
}

type GetGameMetadataResponse struct {
	GameMetadata *GameMetadata `protobuf:"bytes,1,opt,name=game_metadata,json=gameMetadata" json:"game_metadata,omitempty"`
}

func (m *GetGameMetadataResponse) Reset()                    { *m = GetGameMetadataResponse{} }
func (m *GetGameMetadataResponse) String() string            { return proto1.CompactTextString(m) }
func (*GetGameMetadataResponse) ProtoMessage()               {}
func (*GetGameMetadataResponse) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{6} }

func (m *GetGameMetadataResponse) GetGameMetadata() *GameMetadata {
	if m != nil {
		return m.GameMetadata
	}
	return nil
}

type GameMetadata struct {
	Body string `protobuf:"bytes,1,opt,name=body" json:"body,omitempty"`
}

func (m *GameMetadata) Reset()                    { *m = GameMetadata{} }
func (m *GameMetadata) String() string            { return proto1.CompactTextString(m) }
func (*GameMetadata) ProtoMessage()               {}
func (*GameMetadata) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{7} }

func (m *GameMetadata) GetBody() string {
	if m != nil {
		return m.Body
	}
	return ""
}

func init() {
	proto1.RegisterType((*WatchSessionRequest)(nil), "WatchSessionRequest")
	proto1.RegisterType((*WatchSessionResponse)(nil), "WatchSessionResponse")
	proto1.RegisterType((*DeleteSessionRequest)(nil), "DeleteSessionRequest")
	proto1.RegisterType((*DeleteSessionResponse)(nil), "DeleteSessionResponse")
	proto1.RegisterType((*Session)(nil), "Session")
	proto1.RegisterType((*GetGameMetadataRequest)(nil), "GetGameMetadataRequest")
	proto1.RegisterType((*GetGameMetadataResponse)(nil), "GetGameMetadataResponse")
	proto1.RegisterType((*GameMetadata)(nil), "GameMetadata")
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// Client API for Broker service

type BrokerClient interface {
	WatchSession(ctx context.Context, in *WatchSessionRequest, opts ...grpc.CallOption) (Broker_WatchSessionClient, error)
	DeleteSession(ctx context.Context, in *DeleteSessionRequest, opts ...grpc.CallOption) (*DeleteSessionResponse, error)
	GetGameMetadata(ctx context.Context, in *GetGameMetadataRequest, opts ...grpc.CallOption) (*GetGameMetadataResponse, error)
}

type brokerClient struct {
	cc *grpc.ClientConn
}

func NewBrokerClient(cc *grpc.ClientConn) BrokerClient {
	return &brokerClient{cc}
}

func (c *brokerClient) WatchSession(ctx context.Context, in *WatchSessionRequest, opts ...grpc.CallOption) (Broker_WatchSessionClient, error) {
	stream, err := grpc.NewClientStream(ctx, &_Broker_serviceDesc.Streams[0], c.cc, "/Broker/WatchSession", opts...)
	if err != nil {
		return nil, err
	}
	x := &brokerWatchSessionClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type Broker_WatchSessionClient interface {
	Recv() (*WatchSessionResponse, error)
	grpc.ClientStream
}

type brokerWatchSessionClient struct {
	grpc.ClientStream
}

func (x *brokerWatchSessionClient) Recv() (*WatchSessionResponse, error) {
	m := new(WatchSessionResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *brokerClient) DeleteSession(ctx context.Context, in *DeleteSessionRequest, opts ...grpc.CallOption) (*DeleteSessionResponse, error) {
	out := new(DeleteSessionResponse)
	err := grpc.Invoke(ctx, "/Broker/DeleteSession", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *brokerClient) GetGameMetadata(ctx context.Context, in *GetGameMetadataRequest, opts ...grpc.CallOption) (*GetGameMetadataResponse, error) {
	out := new(GetGameMetadataResponse)
	err := grpc.Invoke(ctx, "/Broker/GetGameMetadata", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Server API for Broker service

type BrokerServer interface {
	WatchSession(*WatchSessionRequest, Broker_WatchSessionServer) error
	DeleteSession(context.Context, *DeleteSessionRequest) (*DeleteSessionResponse, error)
	GetGameMetadata(context.Context, *GetGameMetadataRequest) (*GetGameMetadataResponse, error)
}

func RegisterBrokerServer(s *grpc.Server, srv BrokerServer) {
	s.RegisterService(&_Broker_serviceDesc, srv)
}

func _Broker_WatchSession_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(WatchSessionRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(BrokerServer).WatchSession(m, &brokerWatchSessionServer{stream})
}

type Broker_WatchSessionServer interface {
	Send(*WatchSessionResponse) error
	grpc.ServerStream
}

type brokerWatchSessionServer struct {
	grpc.ServerStream
}

func (x *brokerWatchSessionServer) Send(m *WatchSessionResponse) error {
	return x.ServerStream.SendMsg(m)
}

func _Broker_DeleteSession_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DeleteSessionRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BrokerServer).DeleteSession(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/Broker/DeleteSession",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BrokerServer).DeleteSession(ctx, req.(*DeleteSessionRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Broker_GetGameMetadata_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetGameMetadataRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BrokerServer).GetGameMetadata(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/Broker/GetGameMetadata",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BrokerServer).GetGameMetadata(ctx, req.(*GetGameMetadataRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _Broker_serviceDesc = grpc.ServiceDesc{
	ServiceName: "Broker",
	HandlerType: (*BrokerServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "DeleteSession",
			Handler:    _Broker_DeleteSession_Handler,
		},
		{
			MethodName: "GetGameMetadata",
			Handler:    _Broker_GetGameMetadata_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "WatchSession",
			Handler:       _Broker_WatchSession_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "proto/broker.proto",
}

func init() { proto1.RegisterFile("proto/broker.proto", fileDescriptor0) }

var fileDescriptor0 = []byte{
	// 350 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xac, 0x53, 0x41, 0x4f, 0xf2, 0x40,
	0x10, 0xa5, 0x7c, 0x1f, 0x14, 0x06, 0xc8, 0x97, 0x6f, 0x28, 0x94, 0x90, 0x98, 0x90, 0x3d, 0x71,
	0x5a, 0xb5, 0xfe, 0x00, 0x0d, 0x51, 0x09, 0x07, 0x12, 0x03, 0x07, 0x13, 0x2f, 0x64, 0x61, 0x47,
	0x24, 0x02, 0x0b, 0xdd, 0xc5, 0xc4, 0xdf, 0xe9, 0x1f, 0x32, 0x6c, 0x8b, 0xb6, 0x58, 0x6e, 0x9e,
	0xda, 0xd9, 0xf7, 0x5e, 0xe7, 0xf5, 0xcd, 0x2c, 0xe0, 0x26, 0x54, 0x46, 0x9d, 0x4f, 0x43, 0xf5,
	0x4a, 0x21, 0xb7, 0x05, 0xbb, 0x83, 0xfa, 0xa3, 0x30, 0xb3, 0x97, 0x31, 0x69, 0xbd, 0x50, 0xeb,
	0x11, 0x6d, 0x77, 0xa4, 0x0d, 0x72, 0xa8, 0x8b, 0xe5, 0x52, 0xcd, 0x84, 0x21, 0x39, 0xd1, 0x14,
	0xbe, 0x51, 0x38, 0x59, 0xc8, 0x96, 0xd3, 0x71, 0xba, 0xe5, 0xd1, 0xff, 0x2f, 0x68, 0x6c, 0x91,
	0x81, 0x64, 0x0f, 0xe0, 0xa5, 0x3f, 0xa3, 0x37, 0x6a, 0xad, 0x09, 0x3d, 0x28, 0x3c, 0xab, 0xdd,
	0x3a, 0x52, 0x96, 0x46, 0x51, 0x81, 0x0c, 0x5c, 0x1d, 0x11, 0x5b, 0xf9, 0x8e, 0xd3, 0xad, 0x04,
	0x25, 0x7e, 0x10, 0x1e, 0x00, 0x46, 0xe0, 0xdd, 0xd2, 0x92, 0x0c, 0x1d, 0x39, 0x3b, 0x03, 0x88,
	0x29, 0xdf, 0x86, 0xca, 0xf1, 0xc9, 0x40, 0x9e, 0x32, 0x9e, 0x3f, 0x65, 0xdc, 0x87, 0xc6, 0x51,
	0x9b, 0xc8, 0x39, 0xdb, 0x82, 0x1b, 0x1f, 0xfd, 0x72, 0x4b, 0xf4, 0xc1, 0x9d, 0x8b, 0x15, 0xed,
	0x39, 0x7f, 0x2c, 0xa7, 0xb8, 0x2f, 0x07, 0x92, 0x5d, 0x42, 0xb3, 0x4f, 0xa6, 0x2f, 0x56, 0x34,
	0x24, 0x23, 0xa4, 0x30, 0xe2, 0xf0, 0xd3, 0x09, 0x89, 0x93, 0x92, 0x0c, 0xc1, 0xff, 0x21, 0x89,
	0xa3, 0x0f, 0xa0, 0x66, 0x35, 0xab, 0x18, 0xb0, 0xca, 0x4a, 0x50, 0xe3, 0x29, 0x76, 0x75, 0x9e,
	0xa8, 0x18, 0x83, 0x6a, 0x12, 0x45, 0x84, 0xbf, 0x53, 0x25, 0xdf, 0xe3, 0xa6, 0xf6, 0x3d, 0xf8,
	0x70, 0xa0, 0xd8, 0xb3, 0x2b, 0x84, 0xd7, 0x50, 0x4d, 0x4e, 0x1d, 0x3d, 0x9e, 0xb1, 0x4b, 0xed,
	0x06, 0xcf, 0x5a, 0x0d, 0x96, 0xbb, 0x70, 0xf0, 0x06, 0x6a, 0xa9, 0xf4, 0xb1, 0xc1, 0xb3, 0x86,
	0xde, 0x6e, 0xf2, 0xec, 0x21, 0xe5, 0xf0, 0x1e, 0xfe, 0x1d, 0x05, 0x80, 0x3e, 0xcf, 0x4e, 0xb1,
	0xdd, 0xe2, 0x27, 0xb2, 0x62, 0xb9, 0x9e, 0xfb, 0x54, 0xb0, 0x17, 0x62, 0x5a, 0xb4, 0x8f, 0xab,
	0xcf, 0x00, 0x00, 0x00, 0xff, 0xff, 0x26, 0xec, 0xcd, 0x4a, 0x2d, 0x03, 0x00, 0x00,
}
