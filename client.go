package rpc

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"rpc/codec"
	"sync"
)

/**
 * @Description: 一次rpc调用需要的信息
 */
type Call struct {
	Seq uint64
	ServiceMethod string // format <service>.<method>
	Args interface{} //method args
	Reply interface{} //method reply
	Error error //method error
	Done chan *Call
}
// 为了支持异步调用,会有一个Done字段,当调用结束时,会调用call.done() 会通知调用方
func (call *Call) done() {
	call.Done<-call
}

/**
 * @Description: Client 实现部分
 */
type Client struct {
	codecFunc codec.Codec //消息编解码器
	opt *Option
	calling sync.Mutex //保证客户端请求的有序发送
	header codec.Header //请求的消息头,header在请求发送时才需要,由于请求发送是互斥的,因此可以只要一个
	mu sync.Mutex
	seq uint64 //请求的唯一编号
	pending map[uint64]*Call //存储没有处理完的请求,key:seq ,value:call
	closing bool //用户调用Close(),为true时,Client不可用
	shutdown bool //有错误时shutdown,为true时,Client不可用
}
//强制转换
var _ io.Closer = (*Client)(nil)
var ErrShutdown = errors.New("connection is shut down")

/**
 * @Description: 关闭链接
 * @receiver client
 * @return error
 */
func (client *Client) Close() error {
	client.mu.Lock()
	defer client.mu.Unlock()
	if client.closing {
		return ErrShutdown
	}
	client.closing = true
	return client.codecFunc.Close()
}

/**
 * @Description: client 正常工作返回true
 * @receiver client
 * @return bool
 */
func (client *Client) IsAvaliable() bool {
	client.mu.Lock()
	defer client.mu.Unlock()
	return !client.shutdown && !client.closing
}

/**
 * @Description: 创建CLient实例,构造方法
 * @param conn
 * @param opt
 * @return *Client
 * @return error
 */
func NewClient(conn net.Conn,opt *Option)(*Client,error)  {

	//根据配置创建编码器
	codecFunc := codec.NewCodecFuncMap[opt.CodecType]
	if codecFunc == nil{
		err := fmt.Errorf("invalid codec type %s",opt.CodecType)
		log.Println("rpc client: codec error:",err)
		return nil, err
	}
	//发送option信息到服务端
	if err:=json.NewEncoder(conn).Encode(opt); err!=nil{
		log.Println("rpc client: options error:",err)
		conn.Close()
		return nil, err
	}
	return newClientCodec(codecFunc(conn),opt),nil
}

/**
 * @Description: 根据编解码器创建client对象,并且启动协程调用receive()方法接受响应
 * @param codecFunc
 * @param opt
 * @return *Client
 */
func newClientCodec(codecFunc codec.Codec, opt *Option) *Client {
	client := &Client{
		seq:1 ,//seq 从 1开始
		codecFunc: codecFunc,
		opt: opt,
		pending: make(map[uint64]*Call),
	}
	go client.receive()
	return client
}

/**
 * @Description: 和请求相关的三个私有方法
 * @receiver client
 * @param call
 * @return uint64
 * @return error
 */

/**
 * @Description: 注册请求,将请求call添加到client.pending中,并且更新client.seq
 * @receiver client
 * @param call
 * @return uint64
 * @return error
 */
func (client *Client) registerCall(call *Call)(uint64,error){
	client.mu.Lock()
	defer client.mu.Unlock()

	if client.closing || client.shutdown{
		return 0,ErrShutdown
	}
	//请求编号
	call.Seq = client.seq
	client.pending[call.Seq] = call
	client.seq++
	return call.Seq,nil
}
/**
 * @Description: 从未发送的请求中删除请求,根据seq,从client.pending中移除对应的call,并返回
 * @receiver client
 * @param seq 请求编号
 * @return *Call
 */
func (client *Client) removeCall(seq uint64) *Call{
	client.mu.Lock()
	defer client.mu.Unlock()
	call := client.pending[seq]
	delete(client.pending,seq)
	return call
}

/**
 * @Description: 从未请求列表中,主动结束请求,当服务端或者客户端发生错误时,把错误信息通知所有pending状态的请求
 * @receiver client
 * @param err
 */
func (client *Client) terminateCalls(err error){
	client.calling.Lock()
	defer client.calling.Unlock()
	client.mu.Lock()
	defer client.mu.Unlock()
	client.shutdown = true
	for _, call := range client.pending {
		call.Error = err
		call.done()
	}
}


/**
 * @Description: 客户端功能方法
 * @receiver client
 */

/**
 * @Description: 接收响应
 * @receiver client
 */
func (client *Client) receive(){
	var err error
	for err == nil{
		var header codec.Header
		//解码获取header
		if err = client.codecFunc.ReadHeader(&header);err!=nil{
			break
		}
		//根据header获取call
		call:=client.removeCall(header.Seq)

		switch{
		case call == nil:
			//请求不存在
			err = client.codecFunc.ReadBody(nil)
		case header.Error != "":
			//服务端处理出错!header.Error不为空
			call.Error = fmt.Errorf(header.Error)
			err = client.codecFunc.ReadBody(nil)
			call.done()
		default:
			//服务端处理正常,从reply中读取值
			err = client.codecFunc.ReadBody(call.Reply)
			if err != nil{
				call.Error = errors.New("rpc client: reading body "+err.Error())
			}
			call.done()
		}
	}
	//当发生错误时,就通知所有pending中的call
	client.terminateCalls(err)
}



func (client *Client) send(call *Call){
	client.calling.Lock()
	defer client.calling.Unlock()

	//注册请求
	seq,err := client.registerCall(call)
	if err !=nil{
		call.Error = err
		call.done()
		return
	}

	//准备请求头
	client.header.ServiceMethod = call.ServiceMethod
	client.header.Seq = seq
	client.header.Error = ""

	//编码,并且发送请求
	if err := client.codecFunc.Write(&client.header,call.Args);err!=nil{
		//发生错误后,把他从pending状态移除
		call := client.removeCall(seq)
		//当call为nil,一般是该请求已经处理过了
		if call!=nil{
			call.Error = err
			call.done()
		}
	}
}


/**
 * @Description: 简化用户使用的函数
 */

/**
 * @Description: 异步调用的send的包装函数,返回call实例,异步接口,需要自己call.Done()进行阻塞
 * @receiver client
 * @param serviceMethod
 * @param args
 * @param reply
 * @param done
 * @return *Call
 */
func (client *Client) Go(serviceMethod string,args,reply interface{},done chan *Call) *Call{
	if done == nil{
		done = make(chan *Call,10)
	}else if cap(done) == 0{
		log.Panic("rpc client: done channel is unbuffered")
	}
	call := &Call{
		ServiceMethod: serviceMethod,
		Args:args,
		Reply: reply,
		Done: done,
	}
	//
	client.send(call)
	return call
}
/**
 * @Description: 阻塞call.Done,等待响应返回,同步接口
 * @receiver client
 * @param serviceMethod
 * @param args
 * @param reply
 * @return error
 */
func (client *Client) Call(serviceMethod string, args, reply interface{}) error {
	call := <-client.Go(serviceMethod, args, reply, make(chan *Call, 1)).Done
	return call.Error
}

/**
 * @Description: 正确的解析配置以后,通过地址,拨号连接到指定的rpc服务器
 * @param network
 * @param address
 * @param opts
 * @return client
 * @return err
 */
func Dial(network,address string,opts ...*Option)(client *Client,err error) {
	opt ,err :=parseOptions(opts...)
	if err != nil{
		return nil, err
	}
	conn,err := net.Dial(network,address)
	if err != nil{
		return nil, err
	}
	defer func() {
		if err != nil{
			conn.Close()
		}
	}()
	return NewClient(conn,opt)
}

/**
 * @Description: 解析配置
 * @param opts
 * @return *Option
 * @return error
 */
func parseOptions(opts ...*Option) (*Option, error) {

	//如果没有传配置,或者瞎传的配置,就用默认配置
	if len(opts) ==0 || opts[0] == nil{
		return DefaultOption, nil
	}
	if len(opts) != 1{
		return nil,errors.New("number of options is more than 1")
	}
	// todo:为啥只取第一个?
	opt := opts[0]
	//配置项的魔数用默认配置中定义的魔数
	opt.MagicNumber = DefaultOption.MagicNumber
	//如果配置中的编码类型没有传,就用默认配置的编码类型
	if opt.CodecType == ""{
		opt.CodecType = DefaultOption.CodecType
	}
	return opt,nil
}


