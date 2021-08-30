# rpc
rpc by go

[gRPC原理学习概述](./gRPC学习.md)
[RPC原理学习](./rpc原理学习.md)
## 实现篇

### net/rpc包学习

#### 链接处理：
1. 循环等待socket连接建立，并且开启子协程处理每一个链接
2. 在ServeConn中，参数是一个链接，该方法
```go
//Accept 接受侦听器上的连接并提供请求
// 对于每个传入连接。 接受块直到侦听器
// 返回一个非零错误。 调用者通常在一个
// 去语句。
func (server *Server) Accept(lis net.Listener) {
	for {
		conn, err := lis.Accept()
		if err != nil {
			log.Print("rpc.Serve: accept:", err.Error())
			return
		}
		go server.ServeConn(conn)
	}
}
// ServeConn 在单个连接上运行服务器。
// ServeConn 阻塞，服务连接直到客户端挂断。
// 调用者通常在 go 语句中调用 ServeConn。
// ServeConn 使用 gob 线格式（见包 gob）
// 使用通用编解码器，请使用 ServeCodec。
// 有关并发访问的信息，请参阅 NewClient 的注释。
func (server *Server) ServeConn(conn io.ReadWriteCloser) {
	buf := bufio.NewWriter(conn)
	srv := &gobServerCodec{
		rwc:    conn,
		dec:    gob.NewDecoder(conn),
		enc:    gob.NewEncoder(buf),
		encBuf: buf,
	}
	server.ServeCodec(srv)
}
//传入执行的解码器，进行解码
// 
func (server *Server) ServeCodec(codec ServerCodec) {
	sending := new(sync.Mutex)
	wg := new(sync.WaitGroup)
	for {
		service, mtype, req, argv, replyv, keepReading, err := server.readRequest(codec)
		if err != nil {
			if debugLog && err != io.EOF {
				log.Println("rpc:", err)
			}
			if !keepReading {
				break
			}
			// send a response if we actually managed to read a header.
			if req != nil {
				server.sendResponse(sending, req, invalidRequest, codec, err.Error())
				server.freeRequest(req)
			}
			continue
		}
		wg.Add(1)
		go service.call(server, sending, wg, mtype, req, argv, replyv, codec)
	}
	// We've seen that there are no more requests.
	// Wait for responses to be sent before closing codec.
	wg.Wait()
	codec.Close()
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
