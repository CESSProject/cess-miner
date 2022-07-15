// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.27.1
// 	protoc        (unknown)
// source: msg.proto

package rpc

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type ReqMsg struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Version int32  `protobuf:"varint,1,opt,name=version,proto3" json:"version,omitempty"`
	Id      uint64 `protobuf:"varint,2,opt,name=id,proto3" json:"id,omitempty"`
	Method  string `protobuf:"bytes,3,opt,name=method,proto3" json:"method,omitempty"`
	Service string `protobuf:"bytes,4,opt,name=service,proto3" json:"service,omitempty"`
	Body    []byte `protobuf:"bytes,5,opt,name=body,proto3" json:"body,omitempty"`
}

func (x *ReqMsg) Reset() {
	*x = ReqMsg{}
	if protoimpl.UnsafeEnabled {
		mi := &file_msg_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ReqMsg) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ReqMsg) ProtoMessage() {}

func (x *ReqMsg) ProtoReflect() protoreflect.Message {
	mi := &file_msg_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ReqMsg.ProtoReflect.Descriptor instead.
func (*ReqMsg) Descriptor() ([]byte, []int) {
	return file_msg_proto_rawDescGZIP(), []int{0}
}

func (x *ReqMsg) GetVersion() int32 {
	if x != nil {
		return x.Version
	}
	return 0
}

func (x *ReqMsg) GetId() uint64 {
	if x != nil {
		return x.Id
	}
	return 0
}

func (x *ReqMsg) GetMethod() string {
	if x != nil {
		return x.Method
	}
	return ""
}

func (x *ReqMsg) GetService() string {
	if x != nil {
		return x.Service
	}
	return ""
}

func (x *ReqMsg) GetBody() []byte {
	if x != nil {
		return x.Body
	}
	return nil
}

type RespMsg struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id   uint64 `protobuf:"varint,1,opt,name=id,proto3" json:"id,omitempty"`
	Body []byte `protobuf:"bytes,2,opt,name=body,proto3" json:"body,omitempty"`
}

func (x *RespMsg) Reset() {
	*x = RespMsg{}
	if protoimpl.UnsafeEnabled {
		mi := &file_msg_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *RespMsg) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RespMsg) ProtoMessage() {}

func (x *RespMsg) ProtoReflect() protoreflect.Message {
	mi := &file_msg_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RespMsg.ProtoReflect.Descriptor instead.
func (*RespMsg) Descriptor() ([]byte, []int) {
	return file_msg_proto_rawDescGZIP(), []int{1}
}

func (x *RespMsg) GetId() uint64 {
	if x != nil {
		return x.Id
	}
	return 0
}

func (x *RespMsg) GetBody() []byte {
	if x != nil {
		return x.Body
	}
	return nil
}

type RespBody struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Code int32  `protobuf:"varint,1,opt,name=code,proto3" json:"code,omitempty"`
	Msg  string `protobuf:"bytes,2,opt,name=msg,proto3" json:"msg,omitempty"`
	Data []byte `protobuf:"bytes,3,opt,name=data,proto3" json:"data,omitempty"`
}

func (x *RespBody) Reset() {
	*x = RespBody{}
	if protoimpl.UnsafeEnabled {
		mi := &file_msg_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *RespBody) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RespBody) ProtoMessage() {}

func (x *RespBody) ProtoReflect() protoreflect.Message {
	mi := &file_msg_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RespBody.ProtoReflect.Descriptor instead.
func (*RespBody) Descriptor() ([]byte, []int) {
	return file_msg_proto_rawDescGZIP(), []int{2}
}

func (x *RespBody) GetCode() int32 {
	if x != nil {
		return x.Code
	}
	return 0
}

func (x *RespBody) GetMsg() string {
	if x != nil {
		return x.Msg
	}
	return ""
}

func (x *RespBody) GetData() []byte {
	if x != nil {
		return x.Data
	}
	return nil
}

type FileDownloadInfo struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	BlockTotal uint32 `protobuf:"varint,1,opt,name=block_total,json=blockTotal,proto3" json:"block_total,omitempty"`
	BlockIndex uint32 `protobuf:"varint,2,opt,name=block_index,json=blockIndex,proto3" json:"block_index,omitempty"`
	Data       []byte `protobuf:"bytes,3,opt,name=data,proto3" json:"data,omitempty"`
}

func (x *FileDownloadInfo) Reset() {
	*x = FileDownloadInfo{}
	if protoimpl.UnsafeEnabled {
		mi := &file_msg_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *FileDownloadInfo) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*FileDownloadInfo) ProtoMessage() {}

func (x *FileDownloadInfo) ProtoReflect() protoreflect.Message {
	mi := &file_msg_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use FileDownloadInfo.ProtoReflect.Descriptor instead.
func (*FileDownloadInfo) Descriptor() ([]byte, []int) {
	return file_msg_proto_rawDescGZIP(), []int{3}
}

func (x *FileDownloadInfo) GetBlockTotal() uint32 {
	if x != nil {
		return x.BlockTotal
	}
	return 0
}

func (x *FileDownloadInfo) GetBlockIndex() uint32 {
	if x != nil {
		return x.BlockIndex
	}
	return 0
}

func (x *FileDownloadInfo) GetData() []byte {
	if x != nil {
		return x.Data
	}
	return nil
}

type FileDownloadReq struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	FileId     string `protobuf:"bytes,1,opt,name=file_id,json=fileId,proto3" json:"file_id,omitempty"`
	BlockIndex uint32 `protobuf:"varint,2,opt,name=block_index,json=blockIndex,proto3" json:"block_index,omitempty"`
}

func (x *FileDownloadReq) Reset() {
	*x = FileDownloadReq{}
	if protoimpl.UnsafeEnabled {
		mi := &file_msg_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *FileDownloadReq) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*FileDownloadReq) ProtoMessage() {}

func (x *FileDownloadReq) ProtoReflect() protoreflect.Message {
	mi := &file_msg_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use FileDownloadReq.ProtoReflect.Descriptor instead.
func (*FileDownloadReq) Descriptor() ([]byte, []int) {
	return file_msg_proto_rawDescGZIP(), []int{4}
}

func (x *FileDownloadReq) GetFileId() string {
	if x != nil {
		return x.FileId
	}
	return ""
}

func (x *FileDownloadReq) GetBlockIndex() uint32 {
	if x != nil {
		return x.BlockIndex
	}
	return 0
}

//space
type SpaceReq struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Publickey []byte `protobuf:"bytes,1,opt,name=publickey,proto3" json:"publickey,omitempty"`
	Msg       []byte `protobuf:"bytes,2,opt,name=msg,proto3" json:"msg,omitempty"`
	Sign      []byte `protobuf:"bytes,3,opt,name=sign,proto3" json:"sign,omitempty"`
}

func (x *SpaceReq) Reset() {
	*x = SpaceReq{}
	if protoimpl.UnsafeEnabled {
		mi := &file_msg_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *SpaceReq) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SpaceReq) ProtoMessage() {}

func (x *SpaceReq) ProtoReflect() protoreflect.Message {
	mi := &file_msg_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use SpaceReq.ProtoReflect.Descriptor instead.
func (*SpaceReq) Descriptor() ([]byte, []int) {
	return file_msg_proto_rawDescGZIP(), []int{5}
}

func (x *SpaceReq) GetPublickey() []byte {
	if x != nil {
		return x.Publickey
	}
	return nil
}

func (x *SpaceReq) GetMsg() []byte {
	if x != nil {
		return x.Msg
	}
	return nil
}

func (x *SpaceReq) GetSign() []byte {
	if x != nil {
		return x.Sign
	}
	return nil
}

//space_file
type SpaceFileReq struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Token      string `protobuf:"bytes,1,opt,name=token,proto3" json:"token,omitempty"`
	BlockIndex uint32 `protobuf:"varint,2,opt,name=block_index,json=blockIndex,proto3" json:"block_index,omitempty"`
}

func (x *SpaceFileReq) Reset() {
	*x = SpaceFileReq{}
	if protoimpl.UnsafeEnabled {
		mi := &file_msg_proto_msgTypes[6]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *SpaceFileReq) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SpaceFileReq) ProtoMessage() {}

func (x *SpaceFileReq) ProtoReflect() protoreflect.Message {
	mi := &file_msg_proto_msgTypes[6]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use SpaceFileReq.ProtoReflect.Descriptor instead.
func (*SpaceFileReq) Descriptor() ([]byte, []int) {
	return file_msg_proto_rawDescGZIP(), []int{6}
}

func (x *SpaceFileReq) GetToken() string {
	if x != nil {
		return x.Token
	}
	return ""
}

func (x *SpaceFileReq) GetBlockIndex() uint32 {
	if x != nil {
		return x.BlockIndex
	}
	return 0
}

type ReadTagReq struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Acc    []byte `protobuf:"bytes,1,opt,name=acc,proto3" json:"acc,omitempty"`
	FileId string `protobuf:"bytes,2,opt,name=file_id,json=fileId,proto3" json:"file_id,omitempty"`
}

func (x *ReadTagReq) Reset() {
	*x = ReadTagReq{}
	if protoimpl.UnsafeEnabled {
		mi := &file_msg_proto_msgTypes[7]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ReadTagReq) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ReadTagReq) ProtoMessage() {}

func (x *ReadTagReq) ProtoReflect() protoreflect.Message {
	mi := &file_msg_proto_msgTypes[7]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ReadTagReq.ProtoReflect.Descriptor instead.
func (*ReadTagReq) Descriptor() ([]byte, []int) {
	return file_msg_proto_rawDescGZIP(), []int{7}
}

func (x *ReadTagReq) GetAcc() []byte {
	if x != nil {
		return x.Acc
	}
	return nil
}

func (x *ReadTagReq) GetFileId() string {
	if x != nil {
		return x.FileId
	}
	return ""
}

type PutFileToBucket struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	BlockTotal uint32 `protobuf:"varint,1,opt,name=block_total,json=blockTotal,proto3" json:"block_total,omitempty"`
	BlockIndex uint32 `protobuf:"varint,2,opt,name=block_index,json=blockIndex,proto3" json:"block_index,omitempty"`
	FileId     string `protobuf:"bytes,3,opt,name=fileId,proto3" json:"fileId,omitempty"`
	Publickey  []byte `protobuf:"bytes,4,opt,name=publickey,proto3" json:"publickey,omitempty"`
	BlockData  []byte `protobuf:"bytes,5,opt,name=blockData,proto3" json:"blockData,omitempty"`
}

func (x *PutFileToBucket) Reset() {
	*x = PutFileToBucket{}
	if protoimpl.UnsafeEnabled {
		mi := &file_msg_proto_msgTypes[8]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *PutFileToBucket) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*PutFileToBucket) ProtoMessage() {}

func (x *PutFileToBucket) ProtoReflect() protoreflect.Message {
	mi := &file_msg_proto_msgTypes[8]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use PutFileToBucket.ProtoReflect.Descriptor instead.
func (*PutFileToBucket) Descriptor() ([]byte, []int) {
	return file_msg_proto_rawDescGZIP(), []int{8}
}

func (x *PutFileToBucket) GetBlockTotal() uint32 {
	if x != nil {
		return x.BlockTotal
	}
	return 0
}

func (x *PutFileToBucket) GetBlockIndex() uint32 {
	if x != nil {
		return x.BlockIndex
	}
	return 0
}

func (x *PutFileToBucket) GetFileId() string {
	if x != nil {
		return x.FileId
	}
	return ""
}

func (x *PutFileToBucket) GetPublickey() []byte {
	if x != nil {
		return x.Publickey
	}
	return nil
}

func (x *PutFileToBucket) GetBlockData() []byte {
	if x != nil {
		return x.BlockData
	}
	return nil
}

type PutTagToBucket struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	FileId    string   `protobuf:"bytes,1,opt,name=fileId,proto3" json:"fileId,omitempty"`
	Name      []byte   `protobuf:"bytes,2,opt,name=name,proto3" json:"name,omitempty"`
	N         int64    `protobuf:"varint,3,opt,name=n,proto3" json:"n,omitempty"`
	U         [][]byte `protobuf:"bytes,4,rep,name=u,proto3" json:"u,omitempty"`
	Signature []byte   `protobuf:"bytes,5,opt,name=signature,proto3" json:"signature,omitempty"`
	Sigmas    [][]byte `protobuf:"bytes,6,rep,name=sigmas,proto3" json:"sigmas,omitempty"`
}

func (x *PutTagToBucket) Reset() {
	*x = PutTagToBucket{}
	if protoimpl.UnsafeEnabled {
		mi := &file_msg_proto_msgTypes[9]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *PutTagToBucket) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*PutTagToBucket) ProtoMessage() {}

func (x *PutTagToBucket) ProtoReflect() protoreflect.Message {
	mi := &file_msg_proto_msgTypes[9]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use PutTagToBucket.ProtoReflect.Descriptor instead.
func (*PutTagToBucket) Descriptor() ([]byte, []int) {
	return file_msg_proto_rawDescGZIP(), []int{9}
}

func (x *PutTagToBucket) GetFileId() string {
	if x != nil {
		return x.FileId
	}
	return ""
}

func (x *PutTagToBucket) GetName() []byte {
	if x != nil {
		return x.Name
	}
	return nil
}

func (x *PutTagToBucket) GetN() int64 {
	if x != nil {
		return x.N
	}
	return 0
}

func (x *PutTagToBucket) GetU() [][]byte {
	if x != nil {
		return x.U
	}
	return nil
}

func (x *PutTagToBucket) GetSignature() []byte {
	if x != nil {
		return x.Signature
	}
	return nil
}

func (x *PutTagToBucket) GetSigmas() [][]byte {
	if x != nil {
		return x.Sigmas
	}
	return nil
}

var File_msg_proto protoreflect.FileDescriptor

var file_msg_proto_rawDesc = []byte{
	0x0a, 0x09, 0x6d, 0x73, 0x67, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x03, 0x72, 0x70, 0x63,
	0x22, 0x78, 0x0a, 0x06, 0x52, 0x65, 0x71, 0x4d, 0x73, 0x67, 0x12, 0x18, 0x0a, 0x07, 0x76, 0x65,
	0x72, 0x73, 0x69, 0x6f, 0x6e, 0x18, 0x01, 0x20, 0x01, 0x28, 0x05, 0x52, 0x07, 0x76, 0x65, 0x72,
	0x73, 0x69, 0x6f, 0x6e, 0x12, 0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x04,
	0x52, 0x02, 0x69, 0x64, 0x12, 0x16, 0x0a, 0x06, 0x6d, 0x65, 0x74, 0x68, 0x6f, 0x64, 0x18, 0x03,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x6d, 0x65, 0x74, 0x68, 0x6f, 0x64, 0x12, 0x18, 0x0a, 0x07,
	0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x73,
	0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x12, 0x12, 0x0a, 0x04, 0x62, 0x6f, 0x64, 0x79, 0x18, 0x05,
	0x20, 0x01, 0x28, 0x0c, 0x52, 0x04, 0x62, 0x6f, 0x64, 0x79, 0x22, 0x2d, 0x0a, 0x07, 0x52, 0x65,
	0x73, 0x70, 0x4d, 0x73, 0x67, 0x12, 0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x04, 0x52, 0x02, 0x69, 0x64, 0x12, 0x12, 0x0a, 0x04, 0x62, 0x6f, 0x64, 0x79, 0x18, 0x02, 0x20,
	0x01, 0x28, 0x0c, 0x52, 0x04, 0x62, 0x6f, 0x64, 0x79, 0x22, 0x44, 0x0a, 0x08, 0x52, 0x65, 0x73,
	0x70, 0x42, 0x6f, 0x64, 0x79, 0x12, 0x12, 0x0a, 0x04, 0x63, 0x6f, 0x64, 0x65, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x05, 0x52, 0x04, 0x63, 0x6f, 0x64, 0x65, 0x12, 0x10, 0x0a, 0x03, 0x6d, 0x73, 0x67,
	0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6d, 0x73, 0x67, 0x12, 0x12, 0x0a, 0x04, 0x64,
	0x61, 0x74, 0x61, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x04, 0x64, 0x61, 0x74, 0x61, 0x22,
	0x6a, 0x0a, 0x12, 0x66, 0x69, 0x6c, 0x65, 0x5f, 0x64, 0x6f, 0x77, 0x6e, 0x6c, 0x6f, 0x61, 0x64,
	0x5f, 0x69, 0x6e, 0x66, 0x6f, 0x12, 0x1f, 0x0a, 0x0b, 0x62, 0x6c, 0x6f, 0x63, 0x6b, 0x5f, 0x74,
	0x6f, 0x74, 0x61, 0x6c, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x0a, 0x62, 0x6c, 0x6f, 0x63,
	0x6b, 0x54, 0x6f, 0x74, 0x61, 0x6c, 0x12, 0x1f, 0x0a, 0x0b, 0x62, 0x6c, 0x6f, 0x63, 0x6b, 0x5f,
	0x69, 0x6e, 0x64, 0x65, 0x78, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x0a, 0x62, 0x6c, 0x6f,
	0x63, 0x6b, 0x49, 0x6e, 0x64, 0x65, 0x78, 0x12, 0x12, 0x0a, 0x04, 0x64, 0x61, 0x74, 0x61, 0x18,
	0x03, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x04, 0x64, 0x61, 0x74, 0x61, 0x22, 0x4d, 0x0a, 0x11, 0x66,
	0x69, 0x6c, 0x65, 0x5f, 0x64, 0x6f, 0x77, 0x6e, 0x6c, 0x6f, 0x61, 0x64, 0x5f, 0x72, 0x65, 0x71,
	0x12, 0x17, 0x0a, 0x07, 0x66, 0x69, 0x6c, 0x65, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x06, 0x66, 0x69, 0x6c, 0x65, 0x49, 0x64, 0x12, 0x1f, 0x0a, 0x0b, 0x62, 0x6c, 0x6f,
	0x63, 0x6b, 0x5f, 0x69, 0x6e, 0x64, 0x65, 0x78, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x0a,
	0x62, 0x6c, 0x6f, 0x63, 0x6b, 0x49, 0x6e, 0x64, 0x65, 0x78, 0x22, 0x4f, 0x0a, 0x09, 0x73, 0x70,
	0x61, 0x63, 0x65, 0x5f, 0x72, 0x65, 0x71, 0x12, 0x1c, 0x0a, 0x09, 0x70, 0x75, 0x62, 0x6c, 0x69,
	0x63, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x09, 0x70, 0x75, 0x62, 0x6c,
	0x69, 0x63, 0x6b, 0x65, 0x79, 0x12, 0x10, 0x0a, 0x03, 0x6d, 0x73, 0x67, 0x18, 0x02, 0x20, 0x01,
	0x28, 0x0c, 0x52, 0x03, 0x6d, 0x73, 0x67, 0x12, 0x12, 0x0a, 0x04, 0x73, 0x69, 0x67, 0x6e, 0x18,
	0x03, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x04, 0x73, 0x69, 0x67, 0x6e, 0x22, 0x47, 0x0a, 0x0e, 0x73,
	0x70, 0x61, 0x63, 0x65, 0x5f, 0x66, 0x69, 0x6c, 0x65, 0x5f, 0x72, 0x65, 0x71, 0x12, 0x14, 0x0a,
	0x05, 0x74, 0x6f, 0x6b, 0x65, 0x6e, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x74, 0x6f,
	0x6b, 0x65, 0x6e, 0x12, 0x1f, 0x0a, 0x0b, 0x62, 0x6c, 0x6f, 0x63, 0x6b, 0x5f, 0x69, 0x6e, 0x64,
	0x65, 0x78, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x0a, 0x62, 0x6c, 0x6f, 0x63, 0x6b, 0x49,
	0x6e, 0x64, 0x65, 0x78, 0x22, 0x38, 0x0a, 0x0b, 0x52, 0x65, 0x61, 0x64, 0x54, 0x61, 0x67, 0x5f,
	0x72, 0x65, 0x71, 0x12, 0x10, 0x0a, 0x03, 0x61, 0x63, 0x63, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0c,
	0x52, 0x03, 0x61, 0x63, 0x63, 0x12, 0x17, 0x0a, 0x07, 0x66, 0x69, 0x6c, 0x65, 0x5f, 0x69, 0x64,
	0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x66, 0x69, 0x6c, 0x65, 0x49, 0x64, 0x22, 0xa7,
	0x01, 0x0a, 0x0f, 0x50, 0x75, 0x74, 0x46, 0x69, 0x6c, 0x65, 0x54, 0x6f, 0x42, 0x75, 0x63, 0x6b,
	0x65, 0x74, 0x12, 0x1f, 0x0a, 0x0b, 0x62, 0x6c, 0x6f, 0x63, 0x6b, 0x5f, 0x74, 0x6f, 0x74, 0x61,
	0x6c, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x0a, 0x62, 0x6c, 0x6f, 0x63, 0x6b, 0x54, 0x6f,
	0x74, 0x61, 0x6c, 0x12, 0x1f, 0x0a, 0x0b, 0x62, 0x6c, 0x6f, 0x63, 0x6b, 0x5f, 0x69, 0x6e, 0x64,
	0x65, 0x78, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x0a, 0x62, 0x6c, 0x6f, 0x63, 0x6b, 0x49,
	0x6e, 0x64, 0x65, 0x78, 0x12, 0x16, 0x0a, 0x06, 0x66, 0x69, 0x6c, 0x65, 0x49, 0x64, 0x18, 0x03,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x66, 0x69, 0x6c, 0x65, 0x49, 0x64, 0x12, 0x1c, 0x0a, 0x09,
	0x70, 0x75, 0x62, 0x6c, 0x69, 0x63, 0x6b, 0x65, 0x79, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0c, 0x52,
	0x09, 0x70, 0x75, 0x62, 0x6c, 0x69, 0x63, 0x6b, 0x65, 0x79, 0x12, 0x1c, 0x0a, 0x09, 0x62, 0x6c,
	0x6f, 0x63, 0x6b, 0x44, 0x61, 0x74, 0x61, 0x18, 0x05, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x09, 0x62,
	0x6c, 0x6f, 0x63, 0x6b, 0x44, 0x61, 0x74, 0x61, 0x22, 0x8e, 0x01, 0x0a, 0x0e, 0x50, 0x75, 0x74,
	0x54, 0x61, 0x67, 0x54, 0x6f, 0x42, 0x75, 0x63, 0x6b, 0x65, 0x74, 0x12, 0x16, 0x0a, 0x06, 0x66,
	0x69, 0x6c, 0x65, 0x49, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x66, 0x69, 0x6c,
	0x65, 0x49, 0x64, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28,
	0x0c, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x0c, 0x0a, 0x01, 0x6e, 0x18, 0x03, 0x20, 0x01,
	0x28, 0x03, 0x52, 0x01, 0x6e, 0x12, 0x0c, 0x0a, 0x01, 0x75, 0x18, 0x04, 0x20, 0x03, 0x28, 0x0c,
	0x52, 0x01, 0x75, 0x12, 0x1c, 0x0a, 0x09, 0x73, 0x69, 0x67, 0x6e, 0x61, 0x74, 0x75, 0x72, 0x65,
	0x18, 0x05, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x09, 0x73, 0x69, 0x67, 0x6e, 0x61, 0x74, 0x75, 0x72,
	0x65, 0x12, 0x16, 0x0a, 0x06, 0x73, 0x69, 0x67, 0x6d, 0x61, 0x73, 0x18, 0x06, 0x20, 0x03, 0x28,
	0x0c, 0x52, 0x06, 0x73, 0x69, 0x67, 0x6d, 0x61, 0x73, 0x42, 0x08, 0x5a, 0x06, 0x2e, 0x2f, 0x3b,
	0x72, 0x70, 0x63, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_msg_proto_rawDescOnce sync.Once
	file_msg_proto_rawDescData = file_msg_proto_rawDesc
)

func file_msg_proto_rawDescGZIP() []byte {
	file_msg_proto_rawDescOnce.Do(func() {
		file_msg_proto_rawDescData = protoimpl.X.CompressGZIP(file_msg_proto_rawDescData)
	})
	return file_msg_proto_rawDescData
}

var file_msg_proto_msgTypes = make([]protoimpl.MessageInfo, 10)
var file_msg_proto_goTypes = []interface{}{
	(*ReqMsg)(nil),           // 0: rpc.ReqMsg
	(*RespMsg)(nil),          // 1: rpc.RespMsg
	(*RespBody)(nil),         // 2: rpc.RespBody
	(*FileDownloadInfo)(nil), // 3: rpc.file_download_info
	(*FileDownloadReq)(nil),  // 4: rpc.file_download_req
	(*SpaceReq)(nil),         // 5: rpc.space_req
	(*SpaceFileReq)(nil),     // 6: rpc.space_file_req
	(*ReadTagReq)(nil),       // 7: rpc.ReadTag_req
	(*PutFileToBucket)(nil),  // 8: rpc.PutFileToBucket
	(*PutTagToBucket)(nil),   // 9: rpc.PutTagToBucket
}
var file_msg_proto_depIdxs = []int32{
	0, // [0:0] is the sub-list for method output_type
	0, // [0:0] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_msg_proto_init() }
func file_msg_proto_init() {
	if File_msg_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_msg_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ReqMsg); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_msg_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*RespMsg); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_msg_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*RespBody); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_msg_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*FileDownloadInfo); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_msg_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*FileDownloadReq); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_msg_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*SpaceReq); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_msg_proto_msgTypes[6].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*SpaceFileReq); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_msg_proto_msgTypes[7].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ReadTagReq); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_msg_proto_msgTypes[8].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*PutFileToBucket); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_msg_proto_msgTypes[9].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*PutTagToBucket); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_msg_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   10,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_msg_proto_goTypes,
		DependencyIndexes: file_msg_proto_depIdxs,
		MessageInfos:      file_msg_proto_msgTypes,
	}.Build()
	File_msg_proto = out.File
	file_msg_proto_rawDesc = nil
	file_msg_proto_goTypes = nil
	file_msg_proto_depIdxs = nil
}
