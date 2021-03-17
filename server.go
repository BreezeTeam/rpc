package rpc

import (
	"encoding/json"
	"github.com/pkg/errors"
	"io"
	"log"
	"net"
	"reflect"
	"rpc/codec"
	"strings"
	"sync"
)

/**
 * @Description: option部分
 */
const MagicNumber = 0x3bef5c
type Option struct {
	MagicNumber int //魔数,标识服务端的系统
	CodecType codec.Type
}

var DefaultOption = &Option{
	MagicNumber:MagicNumber,
	CodecType:codec.GobType,
}


/**
 * @Description: 服务端的实现部分
 */
type Server struct {
	serviceMap sync.Map
}
/**
 * @Description: 构造一个new Server
 * @return *Server
 */
func NewServer() *Server {
	return &Server{}
}

/**
 * @Description: 服务端的方法区
 * @receiver server
 * @param conn
 */

/**
 * @Description: 处理每一个连接的具体的函数
 * @receiver server
 * @param conn
 */
func (server *Server)HandleConnection(conn io.ReadWriteCloser){
	defer conn.Close() //必须要这样才能关闭链接
	var opt Option
	//反序列化得到Option实例
	if err:=json.NewDecoder(conn).Decode(&opt);err!=nil {
		log.Println("rpc server:option decode error:",err)
		return
	}
	//检查魔数
	if opt.MagicNumber != MagicNumber{
		log.Printf("rpc server:invalid magic number %x",opt.MagicNumber)
		return
	}
	//拿到构造函数
	codecFunc:=codec.NewCodecFuncMap[opt.CodecType]
	if codecFunc == nil{
		log.Printf("rpc server:invalid codec type %s",opt.CodecType)
		return
	}
	//对每一个请求进行解码
	server.connCodec(codecFunc(conn))
}

/**
 * @Description: 一个reponse中 body的占位符,当请求发生错误时使用,或者无法得到body时使用
 */
var invalidRequest = struct{}{}
/**
 * @Description: 一个链接的解码函数,一个链接会有很多请求,主要流程是读取请求,处理请求,回复请求
 * @receiver server
 * @param codecFunc
 */
func (server *Server) connCodec(codecFunc codec.Codec){
	defer codecFunc.Close() //最后的是否,关闭链接
	handleGroup:= new(sync.WaitGroup)
	replyMutex := new(sync.Mutex)
	//一次
	for {
		//读取请求
		req,err := server.readRequest(codecFunc)
		if err !=nil{
			//请求为空时,说明,传输过程丢包了,或者怎么样,关闭连接吧
			if req == nil{
				break
			}
			req.header.Error = err.Error()
			//处理请求可以并发,但是回复请求必须逐个发出,并发会导致多个回复报文在一起,客户端无法解析,这里使用一个互斥锁
			server.sendResponse(codecFunc,req.header,invalidRequest,replyMutex)
			continue
		}
		handleGroup.Add(1)
		//使用协程来并发处理请求
		go server.handleRequest(codecFunc,req,replyMutex,handleGroup)
	}
	//等待该次链接的所有请求处理完毕
	handleGroup.Wait()
}

/**
 * @Description: 这个结构体会存储依次请求中的所有信息
 */
type request struct {
	header *codec.Header	//请求中的header
	argv,replyv reflect.Value //请求中的参数和返回值
	mtype *methodType
	serv *service
}

/**
 * @Description: 读取请求中的header
 * @receiver server
 * @param codecFunc
 * @return *codec.Header
 * @return error
 */
func (server *Server) readRequestHeader(codecFunc  codec.Codec)(*codec.Header,error)  {
	var header codec.Header
	if err := codecFunc.ReadHeader(&header);err !=nil{
		if err != io.EOF && err !=io.ErrUnexpectedEOF{
			log.Println("rpc server:read header error:",err)
		}
		return nil, err
	}
	return &header,nil
}
/**
 * @Description: 读取请求
 * @receiver server
 * @param codecFunc
 * @return *request
 * @return error
 */
func (server *Server) readRequest(codecFunc codec.Codec)(*request ,error){
	//读取请求中的header
	h,err := server.readRequestHeader(codecFunc)
	if err != nil{
		return nil,err
	}
	//构造一个包含了请求中全部信息的数据结构,然后读取请求中的请求体
	req:=&request{header:h}
	req.serv ,req.mtype,err  = server.findService(h.ServiceMethod)
	if err !=nil{
		return req,err
	}
	//创建两个入参
	req.argv = req.mtype.newArgv()
	req.replyv = req.mtype.newReplyv()
	iargv:=req.argv.Interface()
	//判断是否是指针类型
	if req.argv.Type().Kind() != reflect.Ptr{
		iargv = req.argv.Addr().Interface()
	}
	//将请求反序列化为第一个入参
	if err = codecFunc.ReadBody(iargv);err != nil{
		log.Println("rpc server: read body err:",err)
		return req,err
	}
	//最后把这个包含了请求全部信息的对象返回
	return req,nil
}

/**
 * @Description: 回复报文,必须加锁
 * @receiver server
 * @param codecFunc
 * @param header
 * @param body
 * @param mu
 */
func (server *Server) sendResponse(codecFunc codec.Codec, header *codec.Header, body interface{}, mu *sync.Mutex) {
	mu.Lock()
	defer mu.Unlock()
	if err :=codecFunc.Write(header,body);err !=nil {
		log.Println("rpc server:write  response error:",err)
	}
}

func (server *Server) handleRequest(codecFunc codec.Codec, req *request, replyMutex *sync.Mutex, handleGroup *sync.WaitGroup) {
	defer handleGroup.Done()

	err := req.serv.call(req.mtype,req.argv,req.replyv)
	if err !=nil{
		req.header.Error = err.Error()
		server.sendResponse(codecFunc,req.header,invalidRequest,replyMutex)
		return
	}
	//回复请求
	server.sendResponse(codecFunc,req.header,req.replyv.Interface(),replyMutex)
}
/**
 * @Description: 并发处理每一个连接
 * @receiver server
 * @param list
 */
func (server *Server) Accept(list net.Listener) {
	for  {
		conn,err :=list.Accept()
		if err!=nil{
			log.Println("rpc server:accept error:",err)
		}
		go server.HandleConnection(conn)

	}
}

/**
 * @Description: 服务注册
 * @receiver server
 * @param list
 */
func (server *Server) Register(rcvr interface{}) error {
	s := newService(rcvr)
	if _,dup := server.serviceMap.LoadOrStore(s.name,s);dup{
		return errors.New("rpc: service already defind: "+s.name)
	}
	return nil
}


func (server *Server) findService(serviceMethod string) (svc *service, mtype *methodType, err error) {
	dot := strings.LastIndex(serviceMethod, ".")
	if dot < 0 {
		err = errors.New("rpc server: service/method request ill-formed: " + serviceMethod)
		return
	}
	serviceName, methodName := serviceMethod[:dot], serviceMethod[dot+1:]
	svci, ok := server.serviceMap.Load(serviceName)
	if !ok {
		err = errors.New("rpc server: can't find service " + serviceName)
		return
	}
	svc = svci.(*service)
	mtype = svc.method[methodName]
	if mtype == nil {
		err = errors.New("rpc server: can't find method " + methodName)
	}
	return
}



/**
 * @Description: 服务端的快捷函数,简化使用
 */


/**
 * @Description: 默认的Server实例
 */
var DefaultServer = NewServer()

/**
 * @Description: 默认的 Accept 方法,net.Listener 为参数,服务器是默认的服务器
 * @param listener
 */
func Accept(listener net.Listener){
	DefaultServer.Accept(listener)
}

/**
 * @Description: 默认的 Register 方法
 * @param listener
 */
func Register(rcvr interface{})  error{
	return DefaultServer.Register(rcvr)
}