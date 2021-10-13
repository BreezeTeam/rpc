package codec

import (
	"io"
	"rpc/metadata"
)

/**
 * @Description: 编解码器的接口定义
 */
type Codec interface {
	Reader
	Writer
	io.Closer
	String() string
}

type Reader interface {
	ReadHeader(*metadata.Message, metadata.MessageType) error
	ReadBody(interface{}) error
}

type Writer interface {
	Write(*metadata.Message, interface{}) error
}

//抽象出构造函数,入参数是connection ,返回值是一个Codec对象
//定义一个接口形函数
type NewCodec func(io.ReadWriteCloser) Codec


/**
 * @Description: 序列化和反序列化器接口定义
 */
type Marshaler interface {
	Marshal(interface{}) ([]byte, error)
	Unmarshal([]byte, interface{}) error
	String() string
}