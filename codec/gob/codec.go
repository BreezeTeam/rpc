package gob

import (
	"bufio"
	"encoding/gob"
	"io"
	"rpc/codec"
	"rpc/metadata"
)

/**
 * @Description: 这里 gobCodec 回去实现 Codec接口需要的方法;之后 gobCodec 就可以作为Codec类型
 */
type Codec struct {
	Conn    io.ReadWriteCloser //TCP connection/socket
	Buffer  *bufio.Writer      //带缓冲的Writer
	Decoder *gob.Decoder      // 解码器
	Encoder *gob.Encoder      // 编码器
}

func (c Codec) ReadHeader(message *metadata.Message, messageType metadata.MessageType) error {
	return nil
}

func (c Codec) ReadBody(i interface{}) error {
	if i == nil {
		return nil
	}
	return c.Decoder.Decode(i)
}

//如果 写入时会有多次写入,那么就应该使用带缓冲的io,否则直接写入即可,
func (c Codec) Write(message *metadata.Message, i interface{}) error {
	if i == nil {
		return nil
	}
	return c.Encoder.Encode(i)
}

func (c Codec) Close() error {
	return c.Conn.Close()
}

func (c Codec) String() string {
	return "gob"
}

var _ codec.Codec = (*Codec)(nil)
