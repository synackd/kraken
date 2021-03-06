// Code generated by protoc-gen-go. DO NOT EDIT.
// source: RFAggregatorServer.proto

package proto

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

type RFAggregatorServer struct {
	RfAggregator         string   `protobuf:"bytes,1,opt,name=rf_aggregator,json=rfAggregator,proto3" json:"rf_aggregator,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *RFAggregatorServer) Reset()         { *m = RFAggregatorServer{} }
func (m *RFAggregatorServer) String() string { return proto.CompactTextString(m) }
func (*RFAggregatorServer) ProtoMessage()    {}
func (*RFAggregatorServer) Descriptor() ([]byte, []int) {
	return fileDescriptor_RFAggregatorServer_ae2e47f1e111a540, []int{0}
}
func (m *RFAggregatorServer) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_RFAggregatorServer.Unmarshal(m, b)
}
func (m *RFAggregatorServer) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_RFAggregatorServer.Marshal(b, m, deterministic)
}
func (dst *RFAggregatorServer) XXX_Merge(src proto.Message) {
	xxx_messageInfo_RFAggregatorServer.Merge(dst, src)
}
func (m *RFAggregatorServer) XXX_Size() int {
	return xxx_messageInfo_RFAggregatorServer.Size(m)
}
func (m *RFAggregatorServer) XXX_DiscardUnknown() {
	xxx_messageInfo_RFAggregatorServer.DiscardUnknown(m)
}

var xxx_messageInfo_RFAggregatorServer proto.InternalMessageInfo

func (m *RFAggregatorServer) GetRfAggregator() string {
	if m != nil {
		return m.RfAggregator
	}
	return ""
}

func init() {
	proto.RegisterType((*RFAggregatorServer)(nil), "proto.RFAggregatorServer")
}

func init() {
	proto.RegisterFile("RFAggregatorServer.proto", fileDescriptor_RFAggregatorServer_ae2e47f1e111a540)
}

var fileDescriptor_RFAggregatorServer_ae2e47f1e111a540 = []byte{
	// 87 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0x92, 0x08, 0x72, 0x73, 0x4c,
	0x4f, 0x2f, 0x4a, 0x4d, 0x4f, 0x2c, 0xc9, 0x2f, 0x0a, 0x4e, 0x2d, 0x2a, 0x4b, 0x2d, 0xd2, 0x2b,
	0x28, 0xca, 0x2f, 0xc9, 0x17, 0x62, 0x05, 0x53, 0x4a, 0x96, 0x5c, 0x42, 0x98, 0x4a, 0x84, 0x94,
	0xb9, 0x78, 0x8b, 0xd2, 0xe2, 0x13, 0xe1, 0xc2, 0x12, 0x8c, 0x0a, 0x8c, 0x1a, 0x9c, 0x41, 0x3c,
	0x45, 0x69, 0x08, 0xa5, 0x49, 0x6c, 0x60, 0x13, 0x8c, 0x01, 0x01, 0x00, 0x00, 0xff, 0xff, 0x62,
	0x6e, 0x11, 0x55, 0x64, 0x00, 0x00, 0x00,
}
