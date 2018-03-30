// Code generated by protoc-gen-go. DO NOT EDIT.
// source: common/trie/pb/trie.proto

/*
Package triepb is a generated protocol buffer package.

It is generated from these files:
	common/trie/pb/trie.proto

It has these top-level messages:
	Node
*/
package triepb

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

type Node struct {
	Type uint32   `protobuf:"varint,1,opt,name=type" json:"type,omitempty"`
	Val  [][]byte `protobuf:"bytes,2,rep,name=val,proto3" json:"val,omitempty"`
}

func (m *Node) Reset()                    { *m = Node{} }
func (m *Node) String() string            { return proto.CompactTextString(m) }
func (*Node) ProtoMessage()               {}
func (*Node) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

func (m *Node) GetType() uint32 {
	if m != nil {
		return m.Type
	}
	return 0
}

func (m *Node) GetVal() [][]byte {
	if m != nil {
		return m.Val
	}
	return nil
}

func init() {
	proto.RegisterType((*Node)(nil), "triepb.Node")
}

func init() { proto.RegisterFile("common/trie/pb/trie.proto", fileDescriptor0) }

var fileDescriptor0 = []byte{
	// 99 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0x92, 0x4c, 0xce, 0xcf, 0xcd,
	0xcd, 0xcf, 0xd3, 0x2f, 0x29, 0xca, 0x4c, 0xd5, 0x2f, 0x48, 0x02, 0xd3, 0x7a, 0x05, 0x45, 0xf9,
	0x25, 0xf9, 0x42, 0x6c, 0x20, 0x76, 0x41, 0x92, 0x92, 0x0e, 0x17, 0x8b, 0x5f, 0x7e, 0x4a, 0xaa,
	0x90, 0x10, 0x17, 0x4b, 0x49, 0x65, 0x41, 0xaa, 0x04, 0xa3, 0x02, 0xa3, 0x06, 0x6f, 0x10, 0x98,
	0x2d, 0x24, 0xc0, 0xc5, 0x5c, 0x96, 0x98, 0x23, 0xc1, 0xa4, 0xc0, 0xac, 0xc1, 0x13, 0x04, 0x62,
	0x26, 0xb1, 0x81, 0x35, 0x1b, 0x03, 0x02, 0x00, 0x00, 0xff, 0xff, 0x89, 0xd4, 0x6f, 0x84, 0x59,
	0x00, 0x00, 0x00,
}