// Code generated by protoc-gen-go. DO NOT EDIT.
// source: proto/encoder.proto

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

type StartEncodingRequest struct {
	// Pipelines with the same ID will not run at the same time.
	// If you make a request to a running pipeline ID, the running pipeline will stop and a new pipeline will start.
	PipelineId  string `protobuf:"bytes,1,opt,name=pipeline_id,json=pipelineId" json:"pipeline_id,omitempty"`
	GstPipeline string `protobuf:"bytes,2,opt,name=gst_pipeline,json=gstPipeline" json:"gst_pipeline,omitempty"`
	// Use 0 to allocate random port
	Port int32 `protobuf:"varint,3,opt,name=port" json:"port,omitempty"`
}

func (m *StartEncodingRequest) Reset()                    { *m = StartEncodingRequest{} }
func (m *StartEncodingRequest) String() string            { return proto1.CompactTextString(m) }
func (*StartEncodingRequest) ProtoMessage()               {}
func (*StartEncodingRequest) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{0} }

func (m *StartEncodingRequest) GetPipelineId() string {
	if m != nil {
		return m.PipelineId
	}
	return ""
}

func (m *StartEncodingRequest) GetGstPipeline() string {
	if m != nil {
		return m.GstPipeline
	}
	return ""
}

func (m *StartEncodingRequest) GetPort() int32 {
	if m != nil {
		return m.Port
	}
	return 0
}

type StartEncodingResponse struct {
	ListenPort uint32 `protobuf:"varint,1,opt,name=listen_port,json=listenPort" json:"listen_port,omitempty"`
}

func (m *StartEncodingResponse) Reset()                    { *m = StartEncodingResponse{} }
func (m *StartEncodingResponse) String() string            { return proto1.CompactTextString(m) }
func (*StartEncodingResponse) ProtoMessage()               {}
func (*StartEncodingResponse) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{1} }

func (m *StartEncodingResponse) GetListenPort() uint32 {
	if m != nil {
		return m.ListenPort
	}
	return 0
}

func init() {
	proto1.RegisterType((*StartEncodingRequest)(nil), "StartEncodingRequest")
	proto1.RegisterType((*StartEncodingResponse)(nil), "StartEncodingResponse")
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// Client API for Encoder service

type EncoderClient interface {
	StartEncoding(ctx context.Context, in *StartEncodingRequest, opts ...grpc.CallOption) (*StartEncodingResponse, error)
}

type encoderClient struct {
	cc *grpc.ClientConn
}

func NewEncoderClient(cc *grpc.ClientConn) EncoderClient {
	return &encoderClient{cc}
}

func (c *encoderClient) StartEncoding(ctx context.Context, in *StartEncodingRequest, opts ...grpc.CallOption) (*StartEncodingResponse, error) {
	out := new(StartEncodingResponse)
	err := grpc.Invoke(ctx, "/Encoder/StartEncoding", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Server API for Encoder service

type EncoderServer interface {
	StartEncoding(context.Context, *StartEncodingRequest) (*StartEncodingResponse, error)
}

func RegisterEncoderServer(s *grpc.Server, srv EncoderServer) {
	s.RegisterService(&_Encoder_serviceDesc, srv)
}

func _Encoder_StartEncoding_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(StartEncodingRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(EncoderServer).StartEncoding(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/Encoder/StartEncoding",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(EncoderServer).StartEncoding(ctx, req.(*StartEncodingRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _Encoder_serviceDesc = grpc.ServiceDesc{
	ServiceName: "Encoder",
	HandlerType: (*EncoderServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "StartEncoding",
			Handler:    _Encoder_StartEncoding_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "proto/encoder.proto",
}

func init() { proto1.RegisterFile("proto/encoder.proto", fileDescriptor1) }

var fileDescriptor1 = []byte{
	// 200 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0x12, 0x2e, 0x28, 0xca, 0x2f,
	0xc9, 0xd7, 0x4f, 0xcd, 0x4b, 0xce, 0x4f, 0x49, 0x2d, 0xd2, 0x03, 0xf3, 0x94, 0xf2, 0xb8, 0x44,
	0x82, 0x4b, 0x12, 0x8b, 0x4a, 0x5c, 0x41, 0xa2, 0x99, 0x79, 0xe9, 0x41, 0xa9, 0x85, 0xa5, 0xa9,
	0xc5, 0x25, 0x42, 0xf2, 0x5c, 0xdc, 0x05, 0x99, 0x05, 0xa9, 0x39, 0x99, 0x79, 0xa9, 0xf1, 0x99,
	0x29, 0x12, 0x8c, 0x0a, 0x8c, 0x1a, 0x9c, 0x41, 0x5c, 0x30, 0x21, 0xcf, 0x14, 0x21, 0x45, 0x2e,
	0x9e, 0xf4, 0xe2, 0x92, 0x78, 0x98, 0x88, 0x04, 0x13, 0x58, 0x05, 0x77, 0x7a, 0x71, 0x49, 0x00,
	0x54, 0x48, 0x48, 0x88, 0x8b, 0xa5, 0x20, 0xbf, 0xa8, 0x44, 0x82, 0x59, 0x81, 0x51, 0x83, 0x35,
	0x08, 0xcc, 0x56, 0xb2, 0xe0, 0x12, 0x45, 0xb3, 0xaf, 0xb8, 0x20, 0x3f, 0xaf, 0x38, 0x15, 0x64,
	0x61, 0x4e, 0x66, 0x71, 0x49, 0x6a, 0x5e, 0x3c, 0x58, 0x0f, 0xc8, 0x42, 0xde, 0x20, 0x2e, 0x88,
	0x50, 0x40, 0x7e, 0x51, 0x89, 0x91, 0x37, 0x17, 0xbb, 0x2b, 0xc4, 0xe9, 0x42, 0x0e, 0x5c, 0xbc,
	0x28, 0x86, 0x08, 0x89, 0xea, 0x61, 0xf3, 0x84, 0x94, 0x98, 0x1e, 0x56, 0xbb, 0x94, 0x18, 0x9c,
	0xd8, 0xa3, 0x58, 0xc1, 0xfe, 0x4f, 0x62, 0x03, 0x53, 0xc6, 0x80, 0x00, 0x00, 0x00, 0xff, 0xff,
	0x14, 0x93, 0x49, 0x0a, 0x1d, 0x01, 0x00, 0x00,
}
