// Code generated by protoc-gen-go. DO NOT EDIT.
// source: protocol_message.proto

package encryption

import (
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
	math "math"
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

type SignedPreKey struct {
	SignedPreKey         []byte   `protobuf:"bytes,1,opt,name=signed_pre_key,json=signedPreKey,proto3" json:"signed_pre_key,omitempty"`
	Version              uint32   `protobuf:"varint,2,opt,name=version,proto3" json:"version,omitempty"`
	ProtocolVersion      uint32   `protobuf:"varint,3,opt,name=protocol_version,json=protocolVersion,proto3" json:"protocol_version,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *SignedPreKey) Reset()         { *m = SignedPreKey{} }
func (m *SignedPreKey) String() string { return proto.CompactTextString(m) }
func (*SignedPreKey) ProtoMessage()    {}
func (*SignedPreKey) Descriptor() ([]byte, []int) {
	return fileDescriptor_4e37b52004a72e16, []int{0}
}

func (m *SignedPreKey) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_SignedPreKey.Unmarshal(m, b)
}
func (m *SignedPreKey) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_SignedPreKey.Marshal(b, m, deterministic)
}
func (m *SignedPreKey) XXX_Merge(src proto.Message) {
	xxx_messageInfo_SignedPreKey.Merge(m, src)
}
func (m *SignedPreKey) XXX_Size() int {
	return xxx_messageInfo_SignedPreKey.Size(m)
}
func (m *SignedPreKey) XXX_DiscardUnknown() {
	xxx_messageInfo_SignedPreKey.DiscardUnknown(m)
}

var xxx_messageInfo_SignedPreKey proto.InternalMessageInfo

func (m *SignedPreKey) GetSignedPreKey() []byte {
	if m != nil {
		return m.SignedPreKey
	}
	return nil
}

func (m *SignedPreKey) GetVersion() uint32 {
	if m != nil {
		return m.Version
	}
	return 0
}

func (m *SignedPreKey) GetProtocolVersion() uint32 {
	if m != nil {
		return m.ProtocolVersion
	}
	return 0
}

// X3DH prekey bundle
type Bundle struct {
	// Identity key
	Identity []byte `protobuf:"bytes,1,opt,name=identity,proto3" json:"identity,omitempty"`
	// Installation id
	SignedPreKeys map[string]*SignedPreKey `protobuf:"bytes,2,rep,name=signed_pre_keys,json=signedPreKeys,proto3" json:"signed_pre_keys,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	// Prekey signature
	Signature []byte `protobuf:"bytes,4,opt,name=signature,proto3" json:"signature,omitempty"`
	// When the bundle was created locally
	Timestamp            int64    `protobuf:"varint,5,opt,name=timestamp,proto3" json:"timestamp,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Bundle) Reset()         { *m = Bundle{} }
func (m *Bundle) String() string { return proto.CompactTextString(m) }
func (*Bundle) ProtoMessage()    {}
func (*Bundle) Descriptor() ([]byte, []int) {
	return fileDescriptor_4e37b52004a72e16, []int{1}
}

func (m *Bundle) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Bundle.Unmarshal(m, b)
}
func (m *Bundle) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Bundle.Marshal(b, m, deterministic)
}
func (m *Bundle) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Bundle.Merge(m, src)
}
func (m *Bundle) XXX_Size() int {
	return xxx_messageInfo_Bundle.Size(m)
}
func (m *Bundle) XXX_DiscardUnknown() {
	xxx_messageInfo_Bundle.DiscardUnknown(m)
}

var xxx_messageInfo_Bundle proto.InternalMessageInfo

func (m *Bundle) GetIdentity() []byte {
	if m != nil {
		return m.Identity
	}
	return nil
}

func (m *Bundle) GetSignedPreKeys() map[string]*SignedPreKey {
	if m != nil {
		return m.SignedPreKeys
	}
	return nil
}

func (m *Bundle) GetSignature() []byte {
	if m != nil {
		return m.Signature
	}
	return nil
}

func (m *Bundle) GetTimestamp() int64 {
	if m != nil {
		return m.Timestamp
	}
	return 0
}

type BundleContainer struct {
	// X3DH prekey bundle
	Bundle *Bundle `protobuf:"bytes,1,opt,name=bundle,proto3" json:"bundle,omitempty"`
	// Private signed prekey
	PrivateSignedPreKey  []byte   `protobuf:"bytes,2,opt,name=private_signed_pre_key,json=privateSignedPreKey,proto3" json:"private_signed_pre_key,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *BundleContainer) Reset()         { *m = BundleContainer{} }
func (m *BundleContainer) String() string { return proto.CompactTextString(m) }
func (*BundleContainer) ProtoMessage()    {}
func (*BundleContainer) Descriptor() ([]byte, []int) {
	return fileDescriptor_4e37b52004a72e16, []int{2}
}

func (m *BundleContainer) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_BundleContainer.Unmarshal(m, b)
}
func (m *BundleContainer) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_BundleContainer.Marshal(b, m, deterministic)
}
func (m *BundleContainer) XXX_Merge(src proto.Message) {
	xxx_messageInfo_BundleContainer.Merge(m, src)
}
func (m *BundleContainer) XXX_Size() int {
	return xxx_messageInfo_BundleContainer.Size(m)
}
func (m *BundleContainer) XXX_DiscardUnknown() {
	xxx_messageInfo_BundleContainer.DiscardUnknown(m)
}

var xxx_messageInfo_BundleContainer proto.InternalMessageInfo

func (m *BundleContainer) GetBundle() *Bundle {
	if m != nil {
		return m.Bundle
	}
	return nil
}

func (m *BundleContainer) GetPrivateSignedPreKey() []byte {
	if m != nil {
		return m.PrivateSignedPreKey
	}
	return nil
}

type DRHeader struct {
	// Current ratchet public key
	Key []byte `protobuf:"bytes,1,opt,name=key,proto3" json:"key,omitempty"`
	// Number of the message in the sending chain
	N uint32 `protobuf:"varint,2,opt,name=n,proto3" json:"n,omitempty"`
	// Length of the previous sending chain
	Pn uint32 `protobuf:"varint,3,opt,name=pn,proto3" json:"pn,omitempty"`
	// Bundle ID
	Id                   []byte   `protobuf:"bytes,4,opt,name=id,proto3" json:"id,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *DRHeader) Reset()         { *m = DRHeader{} }
func (m *DRHeader) String() string { return proto.CompactTextString(m) }
func (*DRHeader) ProtoMessage()    {}
func (*DRHeader) Descriptor() ([]byte, []int) {
	return fileDescriptor_4e37b52004a72e16, []int{3}
}

func (m *DRHeader) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_DRHeader.Unmarshal(m, b)
}
func (m *DRHeader) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_DRHeader.Marshal(b, m, deterministic)
}
func (m *DRHeader) XXX_Merge(src proto.Message) {
	xxx_messageInfo_DRHeader.Merge(m, src)
}
func (m *DRHeader) XXX_Size() int {
	return xxx_messageInfo_DRHeader.Size(m)
}
func (m *DRHeader) XXX_DiscardUnknown() {
	xxx_messageInfo_DRHeader.DiscardUnknown(m)
}

var xxx_messageInfo_DRHeader proto.InternalMessageInfo

func (m *DRHeader) GetKey() []byte {
	if m != nil {
		return m.Key
	}
	return nil
}

func (m *DRHeader) GetN() uint32 {
	if m != nil {
		return m.N
	}
	return 0
}

func (m *DRHeader) GetPn() uint32 {
	if m != nil {
		return m.Pn
	}
	return 0
}

func (m *DRHeader) GetId() []byte {
	if m != nil {
		return m.Id
	}
	return nil
}

type DHHeader struct {
	// Compressed ephemeral public key
	Key                  []byte   `protobuf:"bytes,1,opt,name=key,proto3" json:"key,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *DHHeader) Reset()         { *m = DHHeader{} }
func (m *DHHeader) String() string { return proto.CompactTextString(m) }
func (*DHHeader) ProtoMessage()    {}
func (*DHHeader) Descriptor() ([]byte, []int) {
	return fileDescriptor_4e37b52004a72e16, []int{4}
}

func (m *DHHeader) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_DHHeader.Unmarshal(m, b)
}
func (m *DHHeader) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_DHHeader.Marshal(b, m, deterministic)
}
func (m *DHHeader) XXX_Merge(src proto.Message) {
	xxx_messageInfo_DHHeader.Merge(m, src)
}
func (m *DHHeader) XXX_Size() int {
	return xxx_messageInfo_DHHeader.Size(m)
}
func (m *DHHeader) XXX_DiscardUnknown() {
	xxx_messageInfo_DHHeader.DiscardUnknown(m)
}

var xxx_messageInfo_DHHeader proto.InternalMessageInfo

func (m *DHHeader) GetKey() []byte {
	if m != nil {
		return m.Key
	}
	return nil
}

type X3DHHeader struct {
	// Ephemeral key used
	Key []byte `protobuf:"bytes,1,opt,name=key,proto3" json:"key,omitempty"`
	// Used bundle's signed prekey
	Id                   []byte   `protobuf:"bytes,4,opt,name=id,proto3" json:"id,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *X3DHHeader) Reset()         { *m = X3DHHeader{} }
func (m *X3DHHeader) String() string { return proto.CompactTextString(m) }
func (*X3DHHeader) ProtoMessage()    {}
func (*X3DHHeader) Descriptor() ([]byte, []int) {
	return fileDescriptor_4e37b52004a72e16, []int{5}
}

func (m *X3DHHeader) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_X3DHHeader.Unmarshal(m, b)
}
func (m *X3DHHeader) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_X3DHHeader.Marshal(b, m, deterministic)
}
func (m *X3DHHeader) XXX_Merge(src proto.Message) {
	xxx_messageInfo_X3DHHeader.Merge(m, src)
}
func (m *X3DHHeader) XXX_Size() int {
	return xxx_messageInfo_X3DHHeader.Size(m)
}
func (m *X3DHHeader) XXX_DiscardUnknown() {
	xxx_messageInfo_X3DHHeader.DiscardUnknown(m)
}

var xxx_messageInfo_X3DHHeader proto.InternalMessageInfo

func (m *X3DHHeader) GetKey() []byte {
	if m != nil {
		return m.Key
	}
	return nil
}

func (m *X3DHHeader) GetId() []byte {
	if m != nil {
		return m.Id
	}
	return nil
}

// Direct message value
type DirectMessageProtocol struct {
	X3DHHeader *X3DHHeader `protobuf:"bytes,1,opt,name=X3DH_header,json=X3DHHeader,proto3" json:"X3DH_header,omitempty"`
	DRHeader   *DRHeader   `protobuf:"bytes,2,opt,name=DR_header,json=DRHeader,proto3" json:"DR_header,omitempty"`
	DHHeader   *DHHeader   `protobuf:"bytes,101,opt,name=DH_header,json=DHHeader,proto3" json:"DH_header,omitempty"`
	// Encrypted payload
	Payload              []byte   `protobuf:"bytes,3,opt,name=payload,proto3" json:"payload,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *DirectMessageProtocol) Reset()         { *m = DirectMessageProtocol{} }
func (m *DirectMessageProtocol) String() string { return proto.CompactTextString(m) }
func (*DirectMessageProtocol) ProtoMessage()    {}
func (*DirectMessageProtocol) Descriptor() ([]byte, []int) {
	return fileDescriptor_4e37b52004a72e16, []int{6}
}

func (m *DirectMessageProtocol) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_DirectMessageProtocol.Unmarshal(m, b)
}
func (m *DirectMessageProtocol) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_DirectMessageProtocol.Marshal(b, m, deterministic)
}
func (m *DirectMessageProtocol) XXX_Merge(src proto.Message) {
	xxx_messageInfo_DirectMessageProtocol.Merge(m, src)
}
func (m *DirectMessageProtocol) XXX_Size() int {
	return xxx_messageInfo_DirectMessageProtocol.Size(m)
}
func (m *DirectMessageProtocol) XXX_DiscardUnknown() {
	xxx_messageInfo_DirectMessageProtocol.DiscardUnknown(m)
}

var xxx_messageInfo_DirectMessageProtocol proto.InternalMessageInfo

func (m *DirectMessageProtocol) GetX3DHHeader() *X3DHHeader {
	if m != nil {
		return m.X3DHHeader
	}
	return nil
}

func (m *DirectMessageProtocol) GetDRHeader() *DRHeader {
	if m != nil {
		return m.DRHeader
	}
	return nil
}

func (m *DirectMessageProtocol) GetDHHeader() *DHHeader {
	if m != nil {
		return m.DHHeader
	}
	return nil
}

func (m *DirectMessageProtocol) GetPayload() []byte {
	if m != nil {
		return m.Payload
	}
	return nil
}

// Top-level protocol message
type ProtocolMessage struct {
	// The device id of the sender
	InstallationId string `protobuf:"bytes,2,opt,name=installation_id,json=installationId,proto3" json:"installation_id,omitempty"`
	// List of bundles
	Bundles []*Bundle `protobuf:"bytes,3,rep,name=bundles,proto3" json:"bundles,omitempty"`
	// One to one message, encrypted, indexed by installation_id
	DirectMessage map[string]*DirectMessageProtocol `protobuf:"bytes,101,rep,name=direct_message,json=directMessage,proto3" json:"direct_message,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	// Public chats, not encrypted
	PublicMessage        []byte   `protobuf:"bytes,102,opt,name=public_message,json=publicMessage,proto3" json:"public_message,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *ProtocolMessage) Reset()         { *m = ProtocolMessage{} }
func (m *ProtocolMessage) String() string { return proto.CompactTextString(m) }
func (*ProtocolMessage) ProtoMessage()    {}
func (*ProtocolMessage) Descriptor() ([]byte, []int) {
	return fileDescriptor_4e37b52004a72e16, []int{7}
}

func (m *ProtocolMessage) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ProtocolMessage.Unmarshal(m, b)
}
func (m *ProtocolMessage) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ProtocolMessage.Marshal(b, m, deterministic)
}
func (m *ProtocolMessage) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ProtocolMessage.Merge(m, src)
}
func (m *ProtocolMessage) XXX_Size() int {
	return xxx_messageInfo_ProtocolMessage.Size(m)
}
func (m *ProtocolMessage) XXX_DiscardUnknown() {
	xxx_messageInfo_ProtocolMessage.DiscardUnknown(m)
}

var xxx_messageInfo_ProtocolMessage proto.InternalMessageInfo

func (m *ProtocolMessage) GetInstallationId() string {
	if m != nil {
		return m.InstallationId
	}
	return ""
}

func (m *ProtocolMessage) GetBundles() []*Bundle {
	if m != nil {
		return m.Bundles
	}
	return nil
}

func (m *ProtocolMessage) GetDirectMessage() map[string]*DirectMessageProtocol {
	if m != nil {
		return m.DirectMessage
	}
	return nil
}

func (m *ProtocolMessage) GetPublicMessage() []byte {
	if m != nil {
		return m.PublicMessage
	}
	return nil
}

func init() {
	proto.RegisterType((*SignedPreKey)(nil), "encryption.SignedPreKey")
	proto.RegisterType((*Bundle)(nil), "encryption.Bundle")
	proto.RegisterMapType((map[string]*SignedPreKey)(nil), "encryption.Bundle.SignedPreKeysEntry")
	proto.RegisterType((*BundleContainer)(nil), "encryption.BundleContainer")
	proto.RegisterType((*DRHeader)(nil), "encryption.DRHeader")
	proto.RegisterType((*DHHeader)(nil), "encryption.DHHeader")
	proto.RegisterType((*X3DHHeader)(nil), "encryption.X3DHHeader")
	proto.RegisterType((*DirectMessageProtocol)(nil), "encryption.DirectMessageProtocol")
	proto.RegisterType((*ProtocolMessage)(nil), "encryption.ProtocolMessage")
	proto.RegisterMapType((map[string]*DirectMessageProtocol)(nil), "encryption.ProtocolMessage.DirectMessageEntry")
}

func init() {
	proto.RegisterFile("protocol_message.proto", fileDescriptor_4e37b52004a72e16)
}

var fileDescriptor_4e37b52004a72e16 = []byte{
	// 563 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x74, 0x54, 0xdd, 0x8e, 0xd3, 0x3c,
	0x10, 0x55, 0x92, 0xdd, 0xfe, 0x4c, 0xd3, 0x1f, 0xf9, 0xfb, 0x58, 0x45, 0xd5, 0x5e, 0x94, 0x88,
	0x15, 0x05, 0xa1, 0x48, 0xb4, 0x48, 0x8b, 0xb8, 0x84, 0x22, 0x95, 0x45, 0x2b, 0xad, 0x8c, 0x40,
	0x88, 0x9b, 0xc8, 0xad, 0xcd, 0x62, 0x91, 0x3a, 0x51, 0xec, 0x56, 0xe4, 0x19, 0x78, 0x34, 0x6e,
	0x78, 0x24, 0x14, 0x27, 0x6e, 0xdd, 0x9f, 0xbd, 0x8b, 0x8f, 0x67, 0xce, 0xcc, 0x39, 0xe3, 0x09,
	0x5c, 0x64, 0x79, 0xaa, 0xd2, 0x65, 0x9a, 0xc4, 0x2b, 0x26, 0x25, 0xb9, 0x67, 0x91, 0x06, 0x10,
	0x30, 0xb1, 0xcc, 0x8b, 0x4c, 0xf1, 0x54, 0x84, 0x05, 0xf8, 0x9f, 0xf8, 0xbd, 0x60, 0xf4, 0x2e,
	0x67, 0x1f, 0x59, 0x81, 0x9e, 0x40, 0x4f, 0xea, 0x73, 0x9c, 0xe5, 0x2c, 0xfe, 0xc9, 0x8a, 0xc0,
	0x19, 0x39, 0x63, 0x1f, 0xfb, 0xd2, 0x8e, 0x0a, 0xa0, 0xb9, 0x61, 0xb9, 0xe4, 0xa9, 0x08, 0xdc,
	0x91, 0x33, 0xee, 0x62, 0x73, 0x44, 0xcf, 0x60, 0xb0, 0xad, 0x6a, 0x42, 0x3c, 0x1d, 0xd2, 0x37,
	0xf8, 0x97, 0x0a, 0x0e, 0x7f, 0xbb, 0xd0, 0x78, 0xbb, 0x16, 0x34, 0x61, 0x68, 0x08, 0x2d, 0x4e,
	0x99, 0x50, 0x5c, 0x99, 0x7a, 0xdb, 0x33, 0xba, 0x85, 0xfe, 0x7e, 0x47, 0x32, 0x70, 0x47, 0xde,
	0xb8, 0x33, 0xb9, 0x8a, 0x76, 0x3a, 0xa2, 0x8a, 0x28, 0xb2, 0xb5, 0xc8, 0xf7, 0x42, 0xe5, 0x05,
	0xee, 0xda, 0x9d, 0x4b, 0x74, 0x09, 0xed, 0x12, 0x20, 0x6a, 0x9d, 0xb3, 0xe0, 0x4c, 0xd7, 0xda,
	0x01, 0xe5, 0xad, 0xe2, 0x2b, 0x26, 0x15, 0x59, 0x65, 0xc1, 0xf9, 0xc8, 0x19, 0x7b, 0x78, 0x07,
	0x0c, 0xbf, 0x01, 0x3a, 0x2e, 0x80, 0x06, 0xe0, 0x19, 0x9f, 0xda, 0xb8, 0xfc, 0x44, 0x11, 0x9c,
	0x6f, 0x48, 0xb2, 0x66, 0xda, 0x9c, 0xce, 0x24, 0xb0, 0x1b, 0xb5, 0x09, 0x70, 0x15, 0xf6, 0xc6,
	0x7d, 0xed, 0x84, 0xbf, 0xa0, 0x5f, 0x69, 0x78, 0x97, 0x0a, 0x45, 0xb8, 0x60, 0x39, 0x7a, 0x0e,
	0x8d, 0x85, 0x86, 0x34, 0x77, 0x67, 0x82, 0x8e, 0x05, 0xe3, 0x3a, 0x02, 0x4d, 0xcb, 0x69, 0xf3,
	0x0d, 0x51, 0x2c, 0x3e, 0x98, 0x9f, 0xab, 0x35, 0xfe, 0x57, 0xdf, 0xda, 0xe5, 0x6f, 0xce, 0x5a,
	0xde, 0xe0, 0x2c, 0xbc, 0x81, 0xd6, 0x0c, 0xcf, 0x19, 0xa1, 0x2c, 0xb7, 0xb5, 0xf8, 0x95, 0x16,
	0x1f, 0x1c, 0x33, 0x64, 0x47, 0xa0, 0x1e, 0xb8, 0x99, 0x19, 0xa8, 0x9b, 0xe9, 0x33, 0xa7, 0xb5,
	0x8d, 0x2e, 0xa7, 0xe1, 0x25, 0xb4, 0x66, 0xf3, 0x87, 0xb8, 0xc2, 0x57, 0x00, 0x5f, 0xa7, 0x0f,
	0xdf, 0x1f, 0xb2, 0xd5, 0xfd, 0xfd, 0x75, 0xe0, 0xd1, 0x8c, 0xe7, 0x6c, 0xa9, 0x6e, 0xab, 0x67,
	0x7c, 0x57, 0x3f, 0x24, 0x74, 0x0d, 0x9d, 0x92, 0x2f, 0xfe, 0xa1, 0x09, 0x6b, 0x97, 0x2e, 0x6c,
	0x97, 0x76, 0xe5, 0xb0, 0x5d, 0xfa, 0x25, 0xb4, 0x67, 0xd8, 0xa4, 0x55, 0x43, 0xfa, 0xdf, 0x4e,
	0x33, 0x7e, 0xe0, 0x9d, 0x33, 0x65, 0xca, 0xb6, 0x12, 0x3b, 0x91, 0x32, 0xdf, 0xa6, 0x98, 0x2a,
	0x01, 0x34, 0x33, 0x52, 0x24, 0x29, 0xa1, 0xda, 0x31, 0x1f, 0x9b, 0x63, 0xf8, 0xc7, 0x85, 0xbe,
	0x51, 0x51, 0x8b, 0x42, 0x4f, 0xa1, 0xcf, 0x85, 0x54, 0x24, 0x49, 0x48, 0x49, 0x18, 0x73, 0xaa,
	0x3b, 0x6b, 0xe3, 0x9e, 0x0d, 0x7f, 0xa0, 0xe8, 0x05, 0x34, 0xab, 0xa1, 0xcb, 0xc0, 0xd3, 0x8b,
	0x70, 0xea, 0x5d, 0x98, 0x10, 0xf4, 0x19, 0x7a, 0x54, 0x9b, 0x67, 0x7e, 0x02, 0x01, 0xd3, 0x49,
	0x91, 0x9d, 0x74, 0xd0, 0x4b, 0xb4, 0x67, 0x77, 0xbd, 0x46, 0xd4, 0xc6, 0xd0, 0x15, 0xf4, 0xb2,
	0xf5, 0x22, 0xe1, 0xcb, 0x2d, 0xed, 0x77, 0x2d, 0xb1, 0x5b, 0xa1, 0x75, 0xd8, 0x70, 0x09, 0xe8,
	0x98, 0xeb, 0xc4, 0xc6, 0x5c, 0xef, 0x6f, 0xcc, 0xe3, 0x3d, 0x67, 0x4f, 0xcd, 0xde, 0x5a, 0x9d,
	0x45, 0x43, 0xff, 0x59, 0xa6, 0xff, 0x02, 0x00, 0x00, 0xff, 0xff, 0xa3, 0x5b, 0x5f, 0x6b, 0xf0,
	0x04, 0x00, 0x00,
}
