package codec

import "io"

type Header struct {
	ServiceMethod string //服务名和方法名
	Seq uint64 //请求序号
	Error string //客户端为空,服务端如果发生错误,会把错误信息放到Error中
}

/**
 * @Description: 编解码器的接口定义
 */
type Codec interface {
	io.Closer
	ReadHeader(*Header) error
	ReadBody(interface{}) error
	Write(*Header,interface{}) error
}

//抽象出构造函数,入参数是connection ,返回值是一个Codec对象
//定义一个接口形函数
type NewCodecFunc func(io.ReadWriteCloser) Codec


/**
 * @Description: 两种编码器的的Context-Type 常量
 */
type Type string
const (
	//两种编解码器
	GobType Type = "application/gob"
	JsonType Type = "application/json"
)
	

//根据Type 从map中国获取对应的构造函数
var NewCodecFuncMap map[Type]NewCodecFunc


//类似于工厂模式
func init(){
	NewCodecFuncMap = make(map[Type]NewCodecFunc)
	NewCodecFuncMap[GobType] = NewGobCodec
	NewCodecFuncMap[JsonType] = NewJsonCodec
}
