// Code generated by protoc-gen-go. DO NOT EDIT.
// source: evm_event.proto

package types

import (
	fmt "fmt"
	math "math"

	proto "github.com/golang/protobuf/proto"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion3 // please upgrade the proto package

// 一条evm event log数据
type EVMLog struct {
	Topic                [][]byte `protobuf:"bytes,1,rep,name=topic,proto3" json:"topic,omitempty"`
	Data                 []byte   `protobuf:"bytes,2,opt,name=data,proto3" json:"data,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *EVMLog) Reset()         { *m = EVMLog{} }
func (m *EVMLog) String() string { return proto.CompactTextString(m) }
func (*EVMLog) ProtoMessage()    {}
func (*EVMLog) Descriptor() ([]byte, []int) {
	return fileDescriptor_00a9a715c51188e3, []int{0}
}

func (m *EVMLog) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_EVMLog.Unmarshal(m, b)
}
func (m *EVMLog) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_EVMLog.Marshal(b, m, deterministic)
}
func (m *EVMLog) XXX_Merge(src proto.Message) {
	xxx_messageInfo_EVMLog.Merge(m, src)
}
func (m *EVMLog) XXX_Size() int {
	return xxx_messageInfo_EVMLog.Size(m)
}
func (m *EVMLog) XXX_DiscardUnknown() {
	xxx_messageInfo_EVMLog.DiscardUnknown(m)
}

var xxx_messageInfo_EVMLog proto.InternalMessageInfo

func (m *EVMLog) GetTopic() [][]byte {
	if m != nil {
		return m.Topic
	}
	return nil
}

func (m *EVMLog) GetData() []byte {
	if m != nil {
		return m.Data
	}
	return nil
}

// 多条evm event log数据
type EVMLogsPerTx struct {
	Logs                 []*EVMLog `protobuf:"bytes,1,rep,name=logs,proto3" json:"logs,omitempty"`
	XXX_NoUnkeyedLiteral struct{}  `json:"-"`
	XXX_unrecognized     []byte    `json:"-"`
	XXX_sizecache        int32     `json:"-"`
}

func (m *EVMLogsPerTx) Reset()         { *m = EVMLogsPerTx{} }
func (m *EVMLogsPerTx) String() string { return proto.CompactTextString(m) }
func (*EVMLogsPerTx) ProtoMessage()    {}
func (*EVMLogsPerTx) Descriptor() ([]byte, []int) {
	return fileDescriptor_00a9a715c51188e3, []int{1}
}

func (m *EVMLogsPerTx) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_EVMLogsPerTx.Unmarshal(m, b)
}
func (m *EVMLogsPerTx) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_EVMLogsPerTx.Marshal(b, m, deterministic)
}
func (m *EVMLogsPerTx) XXX_Merge(src proto.Message) {
	xxx_messageInfo_EVMLogsPerTx.Merge(m, src)
}
func (m *EVMLogsPerTx) XXX_Size() int {
	return xxx_messageInfo_EVMLogsPerTx.Size(m)
}
func (m *EVMLogsPerTx) XXX_DiscardUnknown() {
	xxx_messageInfo_EVMLogsPerTx.DiscardUnknown(m)
}

var xxx_messageInfo_EVMLogsPerTx proto.InternalMessageInfo

func (m *EVMLogsPerTx) GetLogs() []*EVMLog {
	if m != nil {
		return m.Logs
	}
	return nil
}

type EVMTxAndLogs struct {
	Tx                   *Transaction  `protobuf:"bytes,1,opt,name=tx,proto3" json:"tx,omitempty"`
	LogsPerTx            *EVMLogsPerTx `protobuf:"bytes,2,opt,name=logsPerTx,proto3" json:"logsPerTx,omitempty"`
	XXX_NoUnkeyedLiteral struct{}      `json:"-"`
	XXX_unrecognized     []byte        `json:"-"`
	XXX_sizecache        int32         `json:"-"`
}

func (m *EVMTxAndLogs) Reset()         { *m = EVMTxAndLogs{} }
func (m *EVMTxAndLogs) String() string { return proto.CompactTextString(m) }
func (*EVMTxAndLogs) ProtoMessage()    {}
func (*EVMTxAndLogs) Descriptor() ([]byte, []int) {
	return fileDescriptor_00a9a715c51188e3, []int{2}
}

func (m *EVMTxAndLogs) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_EVMTxAndLogs.Unmarshal(m, b)
}
func (m *EVMTxAndLogs) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_EVMTxAndLogs.Marshal(b, m, deterministic)
}
func (m *EVMTxAndLogs) XXX_Merge(src proto.Message) {
	xxx_messageInfo_EVMTxAndLogs.Merge(m, src)
}
func (m *EVMTxAndLogs) XXX_Size() int {
	return xxx_messageInfo_EVMTxAndLogs.Size(m)
}
func (m *EVMTxAndLogs) XXX_DiscardUnknown() {
	xxx_messageInfo_EVMTxAndLogs.DiscardUnknown(m)
}

var xxx_messageInfo_EVMTxAndLogs proto.InternalMessageInfo

func (m *EVMTxAndLogs) GetTx() *Transaction {
	if m != nil {
		return m.Tx
	}
	return nil
}

func (m *EVMTxAndLogs) GetLogsPerTx() *EVMLogsPerTx {
	if m != nil {
		return m.LogsPerTx
	}
	return nil
}

//一个块中包含的多条evm event log数据
type EVMTxLogPerBlk struct {
	TxAndLogs            []*EVMTxAndLogs `protobuf:"bytes,1,rep,name=txAndLogs,proto3" json:"txAndLogs,omitempty"`
	Height               int64           `protobuf:"varint,2,opt,name=height,proto3" json:"height,omitempty"`
	BlockHash            []byte          `protobuf:"bytes,3,opt,name=blockHash,proto3" json:"blockHash,omitempty"`
	ParentHash           []byte          `protobuf:"bytes,4,opt,name=parentHash,proto3" json:"parentHash,omitempty"`
	PreviousHash         []byte          `protobuf:"bytes,5,opt,name=previousHash,proto3" json:"previousHash,omitempty"`
	AddDelType           int32           `protobuf:"varint,6,opt,name=addDelType,proto3" json:"addDelType,omitempty"`
	SeqNum               int64           `protobuf:"varint,7,opt,name=seqNum,proto3" json:"seqNum,omitempty"`
	XXX_NoUnkeyedLiteral struct{}        `json:"-"`
	XXX_unrecognized     []byte          `json:"-"`
	XXX_sizecache        int32           `json:"-"`
}

func (m *EVMTxLogPerBlk) Reset()         { *m = EVMTxLogPerBlk{} }
func (m *EVMTxLogPerBlk) String() string { return proto.CompactTextString(m) }
func (*EVMTxLogPerBlk) ProtoMessage()    {}
func (*EVMTxLogPerBlk) Descriptor() ([]byte, []int) {
	return fileDescriptor_00a9a715c51188e3, []int{3}
}

func (m *EVMTxLogPerBlk) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_EVMTxLogPerBlk.Unmarshal(m, b)
}
func (m *EVMTxLogPerBlk) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_EVMTxLogPerBlk.Marshal(b, m, deterministic)
}
func (m *EVMTxLogPerBlk) XXX_Merge(src proto.Message) {
	xxx_messageInfo_EVMTxLogPerBlk.Merge(m, src)
}
func (m *EVMTxLogPerBlk) XXX_Size() int {
	return xxx_messageInfo_EVMTxLogPerBlk.Size(m)
}
func (m *EVMTxLogPerBlk) XXX_DiscardUnknown() {
	xxx_messageInfo_EVMTxLogPerBlk.DiscardUnknown(m)
}

var xxx_messageInfo_EVMTxLogPerBlk proto.InternalMessageInfo

func (m *EVMTxLogPerBlk) GetTxAndLogs() []*EVMTxAndLogs {
	if m != nil {
		return m.TxAndLogs
	}
	return nil
}

func (m *EVMTxLogPerBlk) GetHeight() int64 {
	if m != nil {
		return m.Height
	}
	return 0
}

func (m *EVMTxLogPerBlk) GetBlockHash() []byte {
	if m != nil {
		return m.BlockHash
	}
	return nil
}

func (m *EVMTxLogPerBlk) GetParentHash() []byte {
	if m != nil {
		return m.ParentHash
	}
	return nil
}

func (m *EVMTxLogPerBlk) GetPreviousHash() []byte {
	if m != nil {
		return m.PreviousHash
	}
	return nil
}

func (m *EVMTxLogPerBlk) GetAddDelType() int32 {
	if m != nil {
		return m.AddDelType
	}
	return 0
}

func (m *EVMTxLogPerBlk) GetSeqNum() int64 {
	if m != nil {
		return m.SeqNum
	}
	return 0
}

//多个块中包含的多条evm event log数据
type EVMTxLogsInBlks struct {
	Logs4EVMPerBlk       []*EVMTxLogPerBlk `protobuf:"bytes,1,rep,name=logs4EVMPerBlk,proto3" json:"logs4EVMPerBlk,omitempty"`
	XXX_NoUnkeyedLiteral struct{}          `json:"-"`
	XXX_unrecognized     []byte            `json:"-"`
	XXX_sizecache        int32             `json:"-"`
}

func (m *EVMTxLogsInBlks) Reset()         { *m = EVMTxLogsInBlks{} }
func (m *EVMTxLogsInBlks) String() string { return proto.CompactTextString(m) }
func (*EVMTxLogsInBlks) ProtoMessage()    {}
func (*EVMTxLogsInBlks) Descriptor() ([]byte, []int) {
	return fileDescriptor_00a9a715c51188e3, []int{4}
}

func (m *EVMTxLogsInBlks) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_EVMTxLogsInBlks.Unmarshal(m, b)
}
func (m *EVMTxLogsInBlks) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_EVMTxLogsInBlks.Marshal(b, m, deterministic)
}
func (m *EVMTxLogsInBlks) XXX_Merge(src proto.Message) {
	xxx_messageInfo_EVMTxLogsInBlks.Merge(m, src)
}
func (m *EVMTxLogsInBlks) XXX_Size() int {
	return xxx_messageInfo_EVMTxLogsInBlks.Size(m)
}
func (m *EVMTxLogsInBlks) XXX_DiscardUnknown() {
	xxx_messageInfo_EVMTxLogsInBlks.DiscardUnknown(m)
}

var xxx_messageInfo_EVMTxLogsInBlks proto.InternalMessageInfo

func (m *EVMTxLogsInBlks) GetLogs4EVMPerBlk() []*EVMTxLogPerBlk {
	if m != nil {
		return m.Logs4EVMPerBlk
	}
	return nil
}

func init() {
	proto.RegisterType((*EVMLog)(nil), "types.EVMLog")
	proto.RegisterType((*EVMLogsPerTx)(nil), "types.EVMLogsPerTx")
	proto.RegisterType((*EVMTxAndLogs)(nil), "types.EVMTxAndLogs")
	proto.RegisterType((*EVMTxLogPerBlk)(nil), "types.EVMTxLogPerBlk")
	proto.RegisterType((*EVMTxLogsInBlks)(nil), "types.EVMTxLogsInBlks")
}

func init() {
	proto.RegisterFile("evm_event.proto", fileDescriptor_00a9a715c51188e3)
}

var fileDescriptor_00a9a715c51188e3 = []byte{
	// 378 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x5c, 0x52, 0xcd, 0xef, 0x9a, 0x40,
	0x10, 0x0d, 0x2a, 0x34, 0xbf, 0x91, 0x6a, 0xba, 0xfd, 0x08, 0x69, 0xfa, 0x41, 0x39, 0x71, 0xc2,
	0x08, 0xbd, 0xf6, 0x50, 0x53, 0x93, 0x36, 0xd1, 0xc6, 0x6c, 0x88, 0x87, 0x5e, 0x9a, 0x15, 0x36,
	0x40, 0xc4, 0x5d, 0xca, 0xae, 0x06, 0xff, 0xf0, 0xde, 0x1b, 0x76, 0x51, 0xb0, 0xb7, 0x9d, 0x79,
	0xef, 0xcd, 0x7b, 0x33, 0x59, 0x98, 0xd3, 0xcb, 0xe9, 0x37, 0xbd, 0x50, 0x26, 0x83, 0xaa, 0xe6,
	0x92, 0x23, 0x53, 0x5e, 0x2b, 0x2a, 0xde, 0xbe, 0x90, 0x35, 0x61, 0x82, 0x24, 0xb2, 0xe0, 0x4c,
	0x23, 0x5e, 0x08, 0xd6, 0x7a, 0xbf, 0xdd, 0xf0, 0x0c, 0xbd, 0x02, 0x53, 0xf2, 0xaa, 0x48, 0x1c,
	0xc3, 0x1d, 0xfb, 0x36, 0xd6, 0x05, 0x42, 0x30, 0x49, 0x89, 0x24, 0xce, 0xc8, 0x35, 0x7c, 0x1b,
	0xab, 0xb7, 0xb7, 0x04, 0x5b, 0x6b, 0xc4, 0x8e, 0xd6, 0x71, 0x83, 0x3e, 0xc1, 0xa4, 0xe4, 0x99,
	0x50, 0xc2, 0x69, 0xf8, 0x3c, 0x50, 0x66, 0x81, 0xa6, 0x60, 0x05, 0x79, 0x54, 0x49, 0xe2, 0xe6,
	0x2b, 0x4b, 0x5b, 0x1d, 0xf2, 0x60, 0x24, 0x1b, 0xc7, 0x70, 0x0d, 0x7f, 0x1a, 0xa2, 0x4e, 0x10,
	0xf7, 0xe1, 0xf0, 0x48, 0x36, 0x68, 0x09, 0x4f, 0xe5, 0xcd, 0x43, 0xf9, 0x4f, 0xc3, 0x97, 0x0f,
	0xb3, 0x35, 0x84, 0x7b, 0x96, 0xf7, 0xd7, 0x80, 0x99, 0xf2, 0xd9, 0xf0, 0x6c, 0x47, 0xeb, 0x55,
	0x79, 0x6c, 0xa7, 0xc8, 0x9b, 0x6d, 0x97, 0x70, 0x30, 0xe5, 0x9e, 0x08, 0xf7, 0x2c, 0xf4, 0x06,
	0xac, 0x9c, 0x16, 0x59, 0x2e, 0x95, 0xeb, 0x18, 0x77, 0x15, 0x7a, 0x07, 0x4f, 0x87, 0x92, 0x27,
	0xc7, 0xef, 0x44, 0xe4, 0xce, 0x58, 0x1d, 0xa4, 0x6f, 0xa0, 0x0f, 0x00, 0x15, 0xa9, 0x29, 0x93,
	0x0a, 0x9e, 0x28, 0x78, 0xd0, 0x41, 0x1e, 0xd8, 0x55, 0x4d, 0x2f, 0x05, 0x3f, 0x0b, 0xc5, 0x30,
	0x15, 0xe3, 0xa1, 0xd7, 0xce, 0x20, 0x69, 0xfa, 0x8d, 0x96, 0xf1, 0xb5, 0xa2, 0x8e, 0xe5, 0x1a,
	0xbe, 0x89, 0x07, 0x9d, 0x36, 0x99, 0xa0, 0x7f, 0x7e, 0x9e, 0x4f, 0xce, 0x33, 0x9d, 0x4c, 0x57,
	0xde, 0x0e, 0xe6, 0xb7, 0xb5, 0xc5, 0x0f, 0xb6, 0x2a, 0x8f, 0x02, 0x7d, 0x81, 0x59, 0x7b, 0x97,
	0xcf, 0xeb, 0xfd, 0x56, 0x5f, 0xa2, 0x5b, 0xfe, 0xf5, 0x70, 0xf9, 0xfb, 0x99, 0xf0, 0x7f, 0xe4,
	0xd5, 0xc7, 0x5f, 0xef, 0xb3, 0x42, 0xe6, 0xe7, 0x43, 0x90, 0xf0, 0xd3, 0x22, 0x8a, 0x12, 0xb6,
	0x48, 0x72, 0x52, 0xb0, 0x28, 0x5a, 0x28, 0xfd, 0xc1, 0x52, 0xff, 0x27, 0xfa, 0x17, 0x00, 0x00,
	0xff, 0xff, 0x4e, 0x43, 0x20, 0x06, 0x6c, 0x02, 0x00, 0x00,
}
