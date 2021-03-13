package codec

import (
	"bufio"
	"encoding/gob"
	"io"
	"log"
)

/**
 * @Description: 这里GobCodec 回去实现 Codec接口需要的方法;之后GobCodec就可以作为Codec类型
 */
type GobCodec struct {
	conn io.ReadWriteCloser //TCP connection/socket
	buf *bufio.Writer //带缓冲的Writer
	dec *gob.Decoder // 解码器
	enc *gob.Encoder // 编码器
}

/**
 * @Description: 虽然下面实现了 Codec 接口,但是,也可以通过这种方式,确保接口被实现,
 * 使得IDE在编译器可以进行检查,但是,似乎Goland已经实现了
 * @param *GobCodec
 * @return nil
 */
var _ Codec = (*GobCodec)(nil)

/**
 * @Description: Codec 接口需要的函数
 * @receiver c
 * @param h
 * @return error
 */
func (c *GobCodec) ReadHeader(h *Header) error{
	return c.dec.Decode(h)
}

/**
 * @Description: Codec 接口需要的函数
 * @receiver c
 * @param body
 * @return error
 */
func (c *GobCodec) ReadBody(body interface{}) error{
	return c.dec.Decode(body)
}

/**
 * @Description: Codec 接口需要的函数
 * @receiver c
 * @param h
 * @param body
 * @return err
 */
func (c *GobCodec) Write(h *Header,body interface{}) (err error){
	defer func() {
		_ = c.buf.Flush() //在结束前,刷新缓冲区
		if err !=nil{
			_ = c.Close() //如果报错,关闭连接
		}
	}()
	//对header进行编码
	if err = c.enc.Encode(h);err !=nil{
		log.Println("rpc:gob error encoding header:",err)
		return err
	}
	//对body进行编码
	if err = c.enc.Encode(body); err != nil {
		log.Println("rpc:gob error encoding body:",err)
		return err
	}
	return nil
}


/**
 * @Description: 关闭连接
 * @receiver c
 * @return error
 */
func (c *GobCodec) Close() error {
	return c.conn.Close()
}

/**
 * @Description: 初始化GobCodec,传入connection,是接口形函数的实现方式之一
 * @param conn
 * @return *Codec
 */
func NewGobCodec(conn io.ReadWriteCloser) Codec {
	buf := bufio.NewWriter(conn)
	return &GobCodec{
		conn: conn,
		buf: buf,
		dec: gob.NewDecoder(conn),
		enc: gob.NewEncoder(buf),
	}
}
//var _ Codec = (*GobCodec)(nil)

