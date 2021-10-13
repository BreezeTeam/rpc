### Codec 编码解码器
Codec 工作在一种协议上,分为编码和解码两个部分
编码器将通信对象编码到字节序列中
解码器从字节序列中解码为对象

#### 包内主要接口说明
1. 编码解码器
```
type Codec interface {
	Reader
	Writer
	io.Closer//实现了 Close() 方法的就是Closer
	String() string
}

type Reader interface {
	ReadHeader(*metadata.Message, metadata.MessageType) error
	ReadBody(interface{}) error
}

type Writer interface {
	Write(*metadata.Message, interface{}) error
}

```
通过编码解码器，调用端和服务端都可以根据需要来该pkg下找需要的编码解码器来进行编码和解码

2. 序列化和反序列化接口
```
type Marshaler interface {
	Marshal(interface{}) ([]byte, error)
	Unmarshal([]byte, interface{}) error
	String() string
}
```

#### customer
该包内是一种基于自定义协议的编解码器
该协议

