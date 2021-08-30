### 编码解码器
interface：
```
type Codec interface {
	io.Closer //实现了 Close() 方法的就是Closer
	ReadHeader(*Header) error
	ReadBody(interface{}) error
	Write(*Header,interface{}) error
}
```
通过编码解码器，调用段和服务端都可以根据需要来该pkg下找需要的编码解码器来进行序列化和反序列化
