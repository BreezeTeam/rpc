# rpc
rpc by go

[gRPC原理学习概述](./gRPC学习.md)
[RPC原理学习](./rpc原理学习.md)



## 实现篇





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