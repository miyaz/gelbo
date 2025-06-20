// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.6
// 	protoc        v5.29.3
// source: gelbo.proto

package __

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
	unsafe "unsafe"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type GelboRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Cpu           string                 `protobuf:"bytes,1,opt,name=cpu,proto3" json:"cpu,omitempty"`
	Memory        string                 `protobuf:"bytes,2,opt,name=memory,proto3" json:"memory,omitempty"`
	Sleep         string                 `protobuf:"bytes,3,opt,name=sleep,proto3" json:"sleep,omitempty"`
	Size          string                 `protobuf:"bytes,4,opt,name=size,proto3" json:"size,omitempty"`
	Code          string                 `protobuf:"bytes,5,opt,name=code,proto3" json:"code,omitempty"`
	Addheader     string                 `protobuf:"bytes,6,opt,name=addheader,proto3" json:"addheader,omitempty"`
	Delheader     string                 `protobuf:"bytes,7,opt,name=delheader,proto3" json:"delheader,omitempty"`
	Addtrailer    string                 `protobuf:"bytes,8,opt,name=addtrailer,proto3" json:"addtrailer,omitempty"`
	Deltrailer    string                 `protobuf:"bytes,9,opt,name=deltrailer,proto3" json:"deltrailer,omitempty"`
	Stdout        string                 `protobuf:"bytes,10,opt,name=stdout,proto3" json:"stdout,omitempty"`
	Stderr        string                 `protobuf:"bytes,11,opt,name=stderr,proto3" json:"stderr,omitempty"`
	Repeat        string                 `protobuf:"bytes,12,opt,name=repeat,proto3" json:"repeat,omitempty"`
	Dataonly      string                 `protobuf:"bytes,13,opt,name=dataonly,proto3" json:"dataonly,omitempty"`
	Noop          string                 `protobuf:"bytes,14,opt,name=noop,proto3" json:"noop,omitempty"`
	Ifclientip    string                 `protobuf:"bytes,15,opt,name=ifclientip,proto3" json:"ifclientip,omitempty"`
	Ifproxy1Ip    string                 `protobuf:"bytes,16,opt,name=ifproxy1ip,proto3" json:"ifproxy1ip,omitempty"`
	Ifproxy2Ip    string                 `protobuf:"bytes,17,opt,name=ifproxy2ip,proto3" json:"ifproxy2ip,omitempty"`
	Ifproxy3Ip    string                 `protobuf:"bytes,18,opt,name=ifproxy3ip,proto3" json:"ifproxy3ip,omitempty"`
	Iflasthopip   string                 `protobuf:"bytes,19,opt,name=iflasthopip,proto3" json:"iflasthopip,omitempty"`
	Iftargetip    string                 `protobuf:"bytes,20,opt,name=iftargetip,proto3" json:"iftargetip,omitempty"`
	Ifhostip      string                 `protobuf:"bytes,21,opt,name=ifhostip,proto3" json:"ifhostip,omitempty"`
	Ifhost        string                 `protobuf:"bytes,22,opt,name=ifhost,proto3" json:"ifhost,omitempty"`
	Ifaz          string                 `protobuf:"bytes,23,opt,name=ifaz,proto3" json:"ifaz,omitempty"`
	Iftype        string                 `protobuf:"bytes,24,opt,name=iftype,proto3" json:"iftype,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *GelboRequest) Reset() {
	*x = GelboRequest{}
	mi := &file_gelbo_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GelboRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GelboRequest) ProtoMessage() {}

func (x *GelboRequest) ProtoReflect() protoreflect.Message {
	mi := &file_gelbo_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GelboRequest.ProtoReflect.Descriptor instead.
func (*GelboRequest) Descriptor() ([]byte, []int) {
	return file_gelbo_proto_rawDescGZIP(), []int{0}
}

func (x *GelboRequest) GetCpu() string {
	if x != nil {
		return x.Cpu
	}
	return ""
}

func (x *GelboRequest) GetMemory() string {
	if x != nil {
		return x.Memory
	}
	return ""
}

func (x *GelboRequest) GetSleep() string {
	if x != nil {
		return x.Sleep
	}
	return ""
}

func (x *GelboRequest) GetSize() string {
	if x != nil {
		return x.Size
	}
	return ""
}

func (x *GelboRequest) GetCode() string {
	if x != nil {
		return x.Code
	}
	return ""
}

func (x *GelboRequest) GetAddheader() string {
	if x != nil {
		return x.Addheader
	}
	return ""
}

func (x *GelboRequest) GetDelheader() string {
	if x != nil {
		return x.Delheader
	}
	return ""
}

func (x *GelboRequest) GetAddtrailer() string {
	if x != nil {
		return x.Addtrailer
	}
	return ""
}

func (x *GelboRequest) GetDeltrailer() string {
	if x != nil {
		return x.Deltrailer
	}
	return ""
}

func (x *GelboRequest) GetStdout() string {
	if x != nil {
		return x.Stdout
	}
	return ""
}

func (x *GelboRequest) GetStderr() string {
	if x != nil {
		return x.Stderr
	}
	return ""
}

func (x *GelboRequest) GetRepeat() string {
	if x != nil {
		return x.Repeat
	}
	return ""
}

func (x *GelboRequest) GetDataonly() string {
	if x != nil {
		return x.Dataonly
	}
	return ""
}

func (x *GelboRequest) GetNoop() string {
	if x != nil {
		return x.Noop
	}
	return ""
}

func (x *GelboRequest) GetIfclientip() string {
	if x != nil {
		return x.Ifclientip
	}
	return ""
}

func (x *GelboRequest) GetIfproxy1Ip() string {
	if x != nil {
		return x.Ifproxy1Ip
	}
	return ""
}

func (x *GelboRequest) GetIfproxy2Ip() string {
	if x != nil {
		return x.Ifproxy2Ip
	}
	return ""
}

func (x *GelboRequest) GetIfproxy3Ip() string {
	if x != nil {
		return x.Ifproxy3Ip
	}
	return ""
}

func (x *GelboRequest) GetIflasthopip() string {
	if x != nil {
		return x.Iflasthopip
	}
	return ""
}

func (x *GelboRequest) GetIftargetip() string {
	if x != nil {
		return x.Iftargetip
	}
	return ""
}

func (x *GelboRequest) GetIfhostip() string {
	if x != nil {
		return x.Ifhostip
	}
	return ""
}

func (x *GelboRequest) GetIfhost() string {
	if x != nil {
		return x.Ifhost
	}
	return ""
}

func (x *GelboRequest) GetIfaz() string {
	if x != nil {
		return x.Ifaz
	}
	return ""
}

func (x *GelboRequest) GetIftype() string {
	if x != nil {
		return x.Iftype
	}
	return ""
}

type GelboResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Host          *HostInfo              `protobuf:"bytes,1,opt,name=host,proto3" json:"host,omitempty"`
	Resource      *ResourceInfo          `protobuf:"bytes,2,opt,name=resource,proto3" json:"resource,omitempty"`
	Request       *RequestInfo           `protobuf:"bytes,3,opt,name=request,proto3" json:"request,omitempty"`
	Direction     *Direction             `protobuf:"bytes,4,opt,name=direction,proto3" json:"direction,omitempty"`
	Data          string                 `protobuf:"bytes,5,opt,name=data,proto3" json:"data,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *GelboResponse) Reset() {
	*x = GelboResponse{}
	mi := &file_gelbo_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GelboResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GelboResponse) ProtoMessage() {}

func (x *GelboResponse) ProtoReflect() protoreflect.Message {
	mi := &file_gelbo_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GelboResponse.ProtoReflect.Descriptor instead.
func (*GelboResponse) Descriptor() ([]byte, []int) {
	return file_gelbo_proto_rawDescGZIP(), []int{1}
}

func (x *GelboResponse) GetHost() *HostInfo {
	if x != nil {
		return x.Host
	}
	return nil
}

func (x *GelboResponse) GetResource() *ResourceInfo {
	if x != nil {
		return x.Resource
	}
	return nil
}

func (x *GelboResponse) GetRequest() *RequestInfo {
	if x != nil {
		return x.Request
	}
	return nil
}

func (x *GelboResponse) GetDirection() *Direction {
	if x != nil {
		return x.Direction
	}
	return nil
}

func (x *GelboResponse) GetData() string {
	if x != nil {
		return x.Data
	}
	return ""
}

type HostInfo struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Name          string                 `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	Ip            string                 `protobuf:"bytes,2,opt,name=ip,proto3" json:"ip,omitempty"`
	Az            string                 `protobuf:"bytes,3,opt,name=az,proto3" json:"az,omitempty"`
	Type          string                 `protobuf:"bytes,4,opt,name=type,proto3" json:"type,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *HostInfo) Reset() {
	*x = HostInfo{}
	mi := &file_gelbo_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *HostInfo) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*HostInfo) ProtoMessage() {}

func (x *HostInfo) ProtoReflect() protoreflect.Message {
	mi := &file_gelbo_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use HostInfo.ProtoReflect.Descriptor instead.
func (*HostInfo) Descriptor() ([]byte, []int) {
	return file_gelbo_proto_rawDescGZIP(), []int{2}
}

func (x *HostInfo) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *HostInfo) GetIp() string {
	if x != nil {
		return x.Ip
	}
	return ""
}

func (x *HostInfo) GetAz() string {
	if x != nil {
		return x.Az
	}
	return ""
}

func (x *HostInfo) GetType() string {
	if x != nil {
		return x.Type
	}
	return ""
}

type ResourceUsage struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Target        float64                `protobuf:"fixed64,1,opt,name=target,proto3" json:"target,omitempty"`
	Current       float64                `protobuf:"fixed64,2,opt,name=current,proto3" json:"current,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *ResourceUsage) Reset() {
	*x = ResourceUsage{}
	mi := &file_gelbo_proto_msgTypes[3]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ResourceUsage) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ResourceUsage) ProtoMessage() {}

func (x *ResourceUsage) ProtoReflect() protoreflect.Message {
	mi := &file_gelbo_proto_msgTypes[3]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ResourceUsage.ProtoReflect.Descriptor instead.
func (*ResourceUsage) Descriptor() ([]byte, []int) {
	return file_gelbo_proto_rawDescGZIP(), []int{3}
}

func (x *ResourceUsage) GetTarget() float64 {
	if x != nil {
		return x.Target
	}
	return 0
}

func (x *ResourceUsage) GetCurrent() float64 {
	if x != nil {
		return x.Current
	}
	return 0
}

type ResourceInfo struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Cpu           *ResourceUsage         `protobuf:"bytes,1,opt,name=cpu,proto3" json:"cpu,omitempty"`
	Memory        *ResourceUsage         `protobuf:"bytes,2,opt,name=memory,proto3" json:"memory,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *ResourceInfo) Reset() {
	*x = ResourceInfo{}
	mi := &file_gelbo_proto_msgTypes[4]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ResourceInfo) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ResourceInfo) ProtoMessage() {}

func (x *ResourceInfo) ProtoReflect() protoreflect.Message {
	mi := &file_gelbo_proto_msgTypes[4]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ResourceInfo.ProtoReflect.Descriptor instead.
func (*ResourceInfo) Descriptor() ([]byte, []int) {
	return file_gelbo_proto_rawDescGZIP(), []int{4}
}

func (x *ResourceInfo) GetCpu() *ResourceUsage {
	if x != nil {
		return x.Cpu
	}
	return nil
}

func (x *ResourceInfo) GetMemory() *ResourceUsage {
	if x != nil {
		return x.Memory
	}
	return nil
}

type RequestInfo struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Protocol      string                 `protobuf:"bytes,1,opt,name=protocol,proto3" json:"protocol,omitempty"`
	Method        string                 `protobuf:"bytes,2,opt,name=method,proto3" json:"method,omitempty"`
	Header        []string               `protobuf:"bytes,3,rep,name=header,proto3" json:"header,omitempty"`
	Clientip      string                 `protobuf:"bytes,4,opt,name=clientip,proto3" json:"clientip,omitempty"`
	Proxy1Ip      string                 `protobuf:"bytes,5,opt,name=proxy1ip,proto3" json:"proxy1ip,omitempty"`
	Proxy2Ip      string                 `protobuf:"bytes,6,opt,name=proxy2ip,proto3" json:"proxy2ip,omitempty"`
	Proxy3Ip      string                 `protobuf:"bytes,7,opt,name=proxy3ip,proto3" json:"proxy3ip,omitempty"`
	Lasthopip     string                 `protobuf:"bytes,8,opt,name=lasthopip,proto3" json:"lasthopip,omitempty"`
	Targetip      string                 `protobuf:"bytes,9,opt,name=targetip,proto3" json:"targetip,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RequestInfo) Reset() {
	*x = RequestInfo{}
	mi := &file_gelbo_proto_msgTypes[5]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RequestInfo) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RequestInfo) ProtoMessage() {}

func (x *RequestInfo) ProtoReflect() protoreflect.Message {
	mi := &file_gelbo_proto_msgTypes[5]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RequestInfo.ProtoReflect.Descriptor instead.
func (*RequestInfo) Descriptor() ([]byte, []int) {
	return file_gelbo_proto_rawDescGZIP(), []int{5}
}

func (x *RequestInfo) GetProtocol() string {
	if x != nil {
		return x.Protocol
	}
	return ""
}

func (x *RequestInfo) GetMethod() string {
	if x != nil {
		return x.Method
	}
	return ""
}

func (x *RequestInfo) GetHeader() []string {
	if x != nil {
		return x.Header
	}
	return nil
}

func (x *RequestInfo) GetClientip() string {
	if x != nil {
		return x.Clientip
	}
	return ""
}

func (x *RequestInfo) GetProxy1Ip() string {
	if x != nil {
		return x.Proxy1Ip
	}
	return ""
}

func (x *RequestInfo) GetProxy2Ip() string {
	if x != nil {
		return x.Proxy2Ip
	}
	return ""
}

func (x *RequestInfo) GetProxy3Ip() string {
	if x != nil {
		return x.Proxy3Ip
	}
	return ""
}

func (x *RequestInfo) GetLasthopip() string {
	if x != nil {
		return x.Lasthopip
	}
	return ""
}

func (x *RequestInfo) GetTargetip() string {
	if x != nil {
		return x.Targetip
	}
	return ""
}

type Direction struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Input         []string               `protobuf:"bytes,1,rep,name=input,proto3" json:"input,omitempty"`
	Result        []string               `protobuf:"bytes,2,rep,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Direction) Reset() {
	*x = Direction{}
	mi := &file_gelbo_proto_msgTypes[6]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Direction) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Direction) ProtoMessage() {}

func (x *Direction) ProtoReflect() protoreflect.Message {
	mi := &file_gelbo_proto_msgTypes[6]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Direction.ProtoReflect.Descriptor instead.
func (*Direction) Descriptor() ([]byte, []int) {
	return file_gelbo_proto_rawDescGZIP(), []int{6}
}

func (x *Direction) GetInput() []string {
	if x != nil {
		return x.Input
	}
	return nil
}

func (x *Direction) GetResult() []string {
	if x != nil {
		return x.Result
	}
	return nil
}

var File_gelbo_proto protoreflect.FileDescriptor

const file_gelbo_proto_rawDesc = "" +
	"\n" +
	"\vgelbo.proto\x12\aelbgrpc\"\x8c\x05\n" +
	"\fGelboRequest\x12\x10\n" +
	"\x03cpu\x18\x01 \x01(\tR\x03cpu\x12\x16\n" +
	"\x06memory\x18\x02 \x01(\tR\x06memory\x12\x14\n" +
	"\x05sleep\x18\x03 \x01(\tR\x05sleep\x12\x12\n" +
	"\x04size\x18\x04 \x01(\tR\x04size\x12\x12\n" +
	"\x04code\x18\x05 \x01(\tR\x04code\x12\x1c\n" +
	"\taddheader\x18\x06 \x01(\tR\taddheader\x12\x1c\n" +
	"\tdelheader\x18\a \x01(\tR\tdelheader\x12\x1e\n" +
	"\n" +
	"addtrailer\x18\b \x01(\tR\n" +
	"addtrailer\x12\x1e\n" +
	"\n" +
	"deltrailer\x18\t \x01(\tR\n" +
	"deltrailer\x12\x16\n" +
	"\x06stdout\x18\n" +
	" \x01(\tR\x06stdout\x12\x16\n" +
	"\x06stderr\x18\v \x01(\tR\x06stderr\x12\x16\n" +
	"\x06repeat\x18\f \x01(\tR\x06repeat\x12\x1a\n" +
	"\bdataonly\x18\r \x01(\tR\bdataonly\x12\x12\n" +
	"\x04noop\x18\x0e \x01(\tR\x04noop\x12\x1e\n" +
	"\n" +
	"ifclientip\x18\x0f \x01(\tR\n" +
	"ifclientip\x12\x1e\n" +
	"\n" +
	"ifproxy1ip\x18\x10 \x01(\tR\n" +
	"ifproxy1ip\x12\x1e\n" +
	"\n" +
	"ifproxy2ip\x18\x11 \x01(\tR\n" +
	"ifproxy2ip\x12\x1e\n" +
	"\n" +
	"ifproxy3ip\x18\x12 \x01(\tR\n" +
	"ifproxy3ip\x12 \n" +
	"\viflasthopip\x18\x13 \x01(\tR\viflasthopip\x12\x1e\n" +
	"\n" +
	"iftargetip\x18\x14 \x01(\tR\n" +
	"iftargetip\x12\x1a\n" +
	"\bifhostip\x18\x15 \x01(\tR\bifhostip\x12\x16\n" +
	"\x06ifhost\x18\x16 \x01(\tR\x06ifhost\x12\x12\n" +
	"\x04ifaz\x18\x17 \x01(\tR\x04ifaz\x12\x16\n" +
	"\x06iftype\x18\x18 \x01(\tR\x06iftype\"\xdf\x01\n" +
	"\rGelboResponse\x12%\n" +
	"\x04host\x18\x01 \x01(\v2\x11.elbgrpc.HostInfoR\x04host\x121\n" +
	"\bresource\x18\x02 \x01(\v2\x15.elbgrpc.ResourceInfoR\bresource\x12.\n" +
	"\arequest\x18\x03 \x01(\v2\x14.elbgrpc.RequestInfoR\arequest\x120\n" +
	"\tdirection\x18\x04 \x01(\v2\x12.elbgrpc.DirectionR\tdirection\x12\x12\n" +
	"\x04data\x18\x05 \x01(\tR\x04data\"R\n" +
	"\bHostInfo\x12\x12\n" +
	"\x04name\x18\x01 \x01(\tR\x04name\x12\x0e\n" +
	"\x02ip\x18\x02 \x01(\tR\x02ip\x12\x0e\n" +
	"\x02az\x18\x03 \x01(\tR\x02az\x12\x12\n" +
	"\x04type\x18\x04 \x01(\tR\x04type\"A\n" +
	"\rResourceUsage\x12\x16\n" +
	"\x06target\x18\x01 \x01(\x01R\x06target\x12\x18\n" +
	"\acurrent\x18\x02 \x01(\x01R\acurrent\"h\n" +
	"\fResourceInfo\x12(\n" +
	"\x03cpu\x18\x01 \x01(\v2\x16.elbgrpc.ResourceUsageR\x03cpu\x12.\n" +
	"\x06memory\x18\x02 \x01(\v2\x16.elbgrpc.ResourceUsageR\x06memory\"\x83\x02\n" +
	"\vRequestInfo\x12\x1a\n" +
	"\bprotocol\x18\x01 \x01(\tR\bprotocol\x12\x16\n" +
	"\x06method\x18\x02 \x01(\tR\x06method\x12\x16\n" +
	"\x06header\x18\x03 \x03(\tR\x06header\x12\x1a\n" +
	"\bclientip\x18\x04 \x01(\tR\bclientip\x12\x1a\n" +
	"\bproxy1ip\x18\x05 \x01(\tR\bproxy1ip\x12\x1a\n" +
	"\bproxy2ip\x18\x06 \x01(\tR\bproxy2ip\x12\x1a\n" +
	"\bproxy3ip\x18\a \x01(\tR\bproxy3ip\x12\x1c\n" +
	"\tlasthopip\x18\b \x01(\tR\tlasthopip\x12\x1a\n" +
	"\btargetip\x18\t \x01(\tR\btargetip\"9\n" +
	"\tDirection\x12\x14\n" +
	"\x05input\x18\x01 \x03(\tR\x05input\x12\x16\n" +
	"\x06result\x18\x02 \x03(\tR\x06result2\x89\x02\n" +
	"\fGelboService\x126\n" +
	"\x05Unary\x12\x15.elbgrpc.GelboRequest\x1a\x16.elbgrpc.GelboResponse\x12?\n" +
	"\fServerStream\x12\x15.elbgrpc.GelboRequest\x1a\x16.elbgrpc.GelboResponse0\x01\x12?\n" +
	"\fClientStream\x12\x15.elbgrpc.GelboRequest\x1a\x16.elbgrpc.GelboResponse(\x01\x12?\n" +
	"\n" +
	"BidiStream\x12\x15.elbgrpc.GelboRequest\x1a\x16.elbgrpc.GelboResponse(\x010\x01B\x04Z\x02./b\x06proto3"

var (
	file_gelbo_proto_rawDescOnce sync.Once
	file_gelbo_proto_rawDescData []byte
)

func file_gelbo_proto_rawDescGZIP() []byte {
	file_gelbo_proto_rawDescOnce.Do(func() {
		file_gelbo_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_gelbo_proto_rawDesc), len(file_gelbo_proto_rawDesc)))
	})
	return file_gelbo_proto_rawDescData
}

var file_gelbo_proto_msgTypes = make([]protoimpl.MessageInfo, 7)
var file_gelbo_proto_goTypes = []any{
	(*GelboRequest)(nil),  // 0: elbgrpc.GelboRequest
	(*GelboResponse)(nil), // 1: elbgrpc.GelboResponse
	(*HostInfo)(nil),      // 2: elbgrpc.HostInfo
	(*ResourceUsage)(nil), // 3: elbgrpc.ResourceUsage
	(*ResourceInfo)(nil),  // 4: elbgrpc.ResourceInfo
	(*RequestInfo)(nil),   // 5: elbgrpc.RequestInfo
	(*Direction)(nil),     // 6: elbgrpc.Direction
}
var file_gelbo_proto_depIdxs = []int32{
	2,  // 0: elbgrpc.GelboResponse.host:type_name -> elbgrpc.HostInfo
	4,  // 1: elbgrpc.GelboResponse.resource:type_name -> elbgrpc.ResourceInfo
	5,  // 2: elbgrpc.GelboResponse.request:type_name -> elbgrpc.RequestInfo
	6,  // 3: elbgrpc.GelboResponse.direction:type_name -> elbgrpc.Direction
	3,  // 4: elbgrpc.ResourceInfo.cpu:type_name -> elbgrpc.ResourceUsage
	3,  // 5: elbgrpc.ResourceInfo.memory:type_name -> elbgrpc.ResourceUsage
	0,  // 6: elbgrpc.GelboService.Unary:input_type -> elbgrpc.GelboRequest
	0,  // 7: elbgrpc.GelboService.ServerStream:input_type -> elbgrpc.GelboRequest
	0,  // 8: elbgrpc.GelboService.ClientStream:input_type -> elbgrpc.GelboRequest
	0,  // 9: elbgrpc.GelboService.BidiStream:input_type -> elbgrpc.GelboRequest
	1,  // 10: elbgrpc.GelboService.Unary:output_type -> elbgrpc.GelboResponse
	1,  // 11: elbgrpc.GelboService.ServerStream:output_type -> elbgrpc.GelboResponse
	1,  // 12: elbgrpc.GelboService.ClientStream:output_type -> elbgrpc.GelboResponse
	1,  // 13: elbgrpc.GelboService.BidiStream:output_type -> elbgrpc.GelboResponse
	10, // [10:14] is the sub-list for method output_type
	6,  // [6:10] is the sub-list for method input_type
	6,  // [6:6] is the sub-list for extension type_name
	6,  // [6:6] is the sub-list for extension extendee
	0,  // [0:6] is the sub-list for field type_name
}

func init() { file_gelbo_proto_init() }
func file_gelbo_proto_init() {
	if File_gelbo_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_gelbo_proto_rawDesc), len(file_gelbo_proto_rawDesc)),
			NumEnums:      0,
			NumMessages:   7,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_gelbo_proto_goTypes,
		DependencyIndexes: file_gelbo_proto_depIdxs,
		MessageInfos:      file_gelbo_proto_msgTypes,
	}.Build()
	File_gelbo_proto = out.File
	file_gelbo_proto_goTypes = nil
	file_gelbo_proto_depIdxs = nil
}
