# rpc
rpc by go


### 消息的虚拟化与反序列化
RPC客户端调用如下:
`err = client.Call("service.Method",args,&reply)`
客户端发送的请求有包含服务名,方法名,参数列表
服务端返回的响应有错误,返回值
将请求和响应中的参数和返回值抽象为body,那么剩余的信息可以抽象为一个Header
```go
type Header struct {
	ServiceMethod string //服务名和方法名
	Seq uint64 //请求序号
	Error string //客户端为空,服务端如果发生错误,会把错误信息放到Error中
}
```

### 通信
客户端与服务端通信需要协商内容,对于rpc来说,会在报文的最开始使用固定的直接协商信息,包括序列化方式,压缩方式,header长度,body长度等
对于我们的rpc来说,目前需要协商的内容是编解码方式.我们可以使用选项模式,来应对以后的变化
```go
const MagicNumber = 0x3bef5c
type Option struct {
	MagicNumber int //魔数,标识服务端的系统
	CodecType codec.Type
}

var DefaultOption = &Option{
	MagicNumber:MagicNumber,
	CodecType:codec.GobType,
}
```

我们的客户端为了便于实现,
固定采用 JSON 编码 Option，后续的 header 和 body 的编码方式由 Option 中的 CodeType 指定，
服务端首先使用 JSON 解码 Option，然后通过 Option 中指定的 CodeType 解码剩余的内容。即报文将以这样的形式发送
| Option{MagicNumber: xxx, CodecType: xxx} | Header{ServiceMethod ...} | Body interface{} |
| <------      固定 JSON 编码      ------>  | <-------   编码方式由 CodeType 决定   ------->|

### 