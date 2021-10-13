# rpc
rpc by go

[gRPC原理学习概述](./gRPC学习.md)
[RPC原理学习](./rpc原理学习.md)
[net/rpc包学习](./标准包net_rpc学习.md)
[go-micro](https://www.cnblogs.com/li-peng/p/10522084.html)

## 设计和实现

### 编解码器
标准消息数据:
```
const (
	Error    MessageType = iota //错误
	Request                     //请求
	Response                    //响应
	Event                       //事件
)

//Message表示关于通信的详细信息，在出现错误的情况下，body可能为nil。
type Message struct {
    
	Id       string      //消息Id
	Type     MessageType //消息类型
	Target   string      //不知道干嘛的
	Method   string      //service method name   format: "Service.Method"
	Endpoint string     //不知道干嘛的
	Error    string     //错误

	// The values read from the socket
	Header map[string]string //Header KV 
	Body   []byte //body部分
}

```
自定义协议数据传输数据
```
//[dubbo 网络传输协议](https://zhuanlan.zhihu.com/p/98562180)
//响应状态
const (
	OK = iota
	CLIENT_TIMEOUT
	SERVER_TIMEOUT
	BAD_REQUEST
	BAD_RESPONSE
	SERVICE_NOT_FOUND
	SERVICE_ERROR
	SERVER_ERROR
	CLIENT_ERROR
	SERVER_THREADPOOL_EXHAUSTED_ERROR
)

//字段长度
const (
	MagicNumber               = 0x6aed56e8
	ConstMagicNumber          = 4
	ConstProtocolLength       = 4
	ConstProtocolHeaderLength = 4
	ConstCodecType            = 4
	ConstConnectTimeout       = 8
	ConstHandleTimeout        = 8

	ConstSequence = 16

	ConstHeaderLength = 32
	ConstDataLength   = 4
)

//固定部分协议
type ProtocolHeader struct {
	MagicNumber          int32          //魔数,标识RPC传输协议的版本号,4位int类型，得到之后，将其转为16进制 4位
	ProtocolLength       int32          //协议体长度 4位
	ProtocolHeaderLength int32          //协议头长度 4位
	CodecType            codec.TypeCode //编码类型 int32 4位
	ConnectTimeout       time.Duration  //链接超时时间 8位
	HandleTimeout        time.Duration  //处理超时时间 8位
	Extend               *Extend        //扩展字段
}
```

综合的数据
```
const (
	Error    MessageType = iota //错误
	Request                     //请求
	Response                    //响应
	Event                       //事件
)
type StatusType int 
const (
	OK StatusType = iota //OK
	CLIENT_TIMEOUT         //客户端超时
	SERVER_TIMEOUT          //服务端超时
	BAD_REQUEST             //错误请求
	BAD_RESPONSE            //错误的响应
	SERVICE_NOT_FOUND       //服务未找到
	SERVICE_ERROR   //服务错误
	SERVER_ERROR    //服务端错误
	CLIENT_ERROR    //客户端错误
	SERVER_THREADPOOL_EXHAUSTED_ERROR//服务器线程资源错误
)
//Message表示关于通信的详细信息，在出现错误的情况下，body可能为nil。
type Message struct {
    MagicNumber


	Id       string      //消息Id
	Type     MessageType //消息类型

	Target   string      //不知道干嘛的
	Method   string      //service method name   format: "Service.Method"
	Endpoint string     // 接入点,默认和method相同
	Error    string     //错误

	// The values read from the socket
    ConnectTimeout       time.Duration  //链接超时时间 8位 放在Header中
    HandleTimeout        time.Duration  //处理超时时间 8位 放在Header中
    Status          //,可以放在Header中
	Header map[string]string 
	Body   []byte
}

```


**golang 实现rpc序列化**

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

|   固定 JSON 编码   |  编码方式由 CodeType 决定    |
| ---- | ---- |
|  Option{MagicNumber: xxx, CodecType: xxx}  |  Header{ServiceMethod ...},Body interface{}   |
|      |      |
