package codec

import (
	"bufio"
	"encoding/json"
	"io"
	"log"
)

/**
 * @Description: 这里 JsonCodec 回去实现 Codec接口需要的方法;之后 JsonCodec 就可以作为Codec类型
 */
type JsonCodec struct {
	conn io.ReadWriteCloser //TCP connection/socket
	buf *bufio.Writer //带缓冲的Writer
	dec *json.Decoder // 解码器
	enc *json.Encoder // 编码器
}

/**
 * @Description: Codec 接口需要的函数
 * @receiver c
 * @param h
 * @return error
 */
func (c *JsonCodec) ReadHeader(h *Header) error{
	return c.dec.Decode(h)
}

/**
 * @Description: Codec 接口需要的函数
 * @receiver c
 * @param body
 * @return error
 */
func (c *JsonCodec) ReadBody(body interface{}) error{
	return c.dec.Decode(body)
}

/**
 * @Description: Codec 接口需要的函数
 * @receiver c
 * @param h
 * @param body
 * @return err
 */
func (c *JsonCodec) Write(h *Header,body interface{}) (err error){
	defer func() {
		_ = c.buf.Flush() //在结束前,刷新缓冲区
		if err !=nil{
			_ = c.Close() //如果报错,关闭连接
		}
	}()
	//对header进行编码
	if err = c.enc.Encode(h);err !=nil{
		log.Println("rpc:json error encoding header:",err)
		return err
	}
	//对body进行编码
	if err = c.enc.Encode(body); err != nil {
		log.Println("rpc:json error encoding body:",err)
		return err
	}
	return nil
}


/**
 * @Description: 关闭连接
 * @receiver c
 * @return error
 */
func (c *JsonCodec) Close() error {
	return c.conn.Close()
}

/**
 * @Description: 初始化 JsonCodec,传入connection
 * @param conn
 * @return *Codec
 */
func NewJsonCodec(conn io.ReadWriteCloser) Codec {
	buf := bufio.NewWriter(conn)
	return &JsonCodec{
		conn: conn,
		buf: buf,
		dec: json.NewDecoder(conn),
		enc: json.NewEncoder(buf),
	}
}
//var _ Codec = (*JsonCodec)(nil)

