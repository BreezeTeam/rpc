package rpc

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"io"
	"log"
	"net"
	"reflect"
	"rpc/codec"
	"strings"
	"sync"
	"time"
)

/**
 * @Description: option部分
 */
const MagicNumber = 0x3bef5c

/*
type Header struct {
	ServiceMethod string //服务名和方法名
	Seq uint64 //请求序号
	Error string //客户端为空,服务端如果发生错误,会把错误信息放到Error中
}

*/

/**

Register(rcvr interface{}) error
RegisterName(name string, rcvr interface{}) error
register(rcvr interface{}, name string, useName bool) error
sendResponse(sending *sync.Mutex, req *Request, reply interface{}, codec ServerCodec, errmsg string)
ServeConn(conn io.ReadWriteCloser)
ServeCodec(codec ServerCodec)
ServeRequest(codec ServerCodec) error
getRequest() *Request
freeRequest(req *Request)
getResponse() *Response
freeResponse(resp *Response)
readRequest(codec ServerCodec) (service *service, mtype *methodType, req *Request, argv reflect.Value, replyv reflect.Value, keepReading bool, err error)
readRequestHeader(codec ServerCodec) (svc *service, mtype *methodType, req *Request, keepReading bool, err error)
Accept(lis net.Listener)
ServeHTTP(w http.ResponseWriter, req *http.Request)
HandleHTTP(rpcPath string, debugPath string)
*/

var DefaultOption = &Protocol{
	MagicNumber:    MagicNumber,
	CodecType:      codec.JsonType,
	ConnectTimeout: time.Second * 10,
}

/**
 * @Description: 服务端的实现部分
 */
type Server struct {
	serviceMap sync.Map // map[string]*service
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
 * @Description: 协议层 处理每一个连接的具体的函数
 * @receiver server
 * @param conn
 */
func (server *Server) ServeConn(conn io.ReadWriteCloser) {
	defer conn.Close() //必须要这样才能关闭链接
	var opt Protocol
	// 将链接 通过json反序列化得到Option实例，
	if err := json.NewDecoder(conn).Decode(&opt); err != nil {
		log.Println("rpc server:option decode error:", err)
		return
	}
	buf := bufio.NewWriter(conn)

	//检查魔数
	if opt.MagicNumber != MagicNumber {
		log.Printf("rpc server:invalid magic number %x", opt.MagicNumber)
		return
	}
	//拿到构造函数
	codecFunc := codec.NewCodecFuncMap[opt.CodecType]
	if codecFunc == nil {
		log.Printf("rpc server:invalid codec type %s", opt.CodecType)
		return
	}
	//对每一个请求进行解码
	server.ServeConnCodec(codecFunc(conn), &opt)
}

/**
 * @Description: 一个reponse中 body的占位符,当请求发生错误时使用,或者无法得到body时使用
 */
var invalidRequest = struct{}{}

type ServerCodec interface {
	ReadRequestHeader(*Request) error //通过
	ReadRequestBody(interface{}) error
	WriteResponse(*Response, interface{}) error
	io.Closer
}

/**
 * @Description: 一个链接的解码函数,一个链接会有很多请求,主要流程是读取请求,处理请求,回复请求
 * @receiver server
 * @param codecFunc
 */
func (server *Server) ServeConnCodec(codec ServerCodec) {
	//最后需要记得回收链接
	defer codec.Close()
	replyMutex := new(sync.Mutex) // 发送的锁
	handleGroup := new(sync.WaitGroup)
	for {
		//读取请求
		req, err := server.readRequest(codec) // 读取请求  service, mtype, req, argv, replyv, keepReading, err := server.readRequest(codec)
		if err != nil {
			//			if debugLog && err != io.EOF {
			//				log.Println("rpc:", err)
			//			}

			//if !keepReading {
			//	break
			//}

			//请求为空时,说明,传输过程丢包了,或者怎么样,关闭连接吧
			if req != nil {
				//发送请求
				//server.sendResponse(sending, req, invalidRequest, codec, err.Error())
				//释放请求
				//server.freeRequest(req)

				req.header.Error = err.Error()
				//处理请求可以并发,但是回复请求必须逐个发出,并发会导致多个回复报文在一起,客户端无法解析,这里使用一个互斥锁
				server.sendResponse(codec, req.header, invalidRequest, replyMutex)
			}
			continue
		}
		handleGroup.Add(1)
		//使用协程来并发处理请求
		go server.call(codec, req, replyMutex, handleGroup, opt.HandleTimeout) //对应rpc call
	}
	//等待该次链接的所有请求处理完毕
	handleGroup.Wait()
}

/**
 * @Description: 这个结构体会存储依次请求中的所有信息
 */
type request struct {
	header       *codec.Header //请求中的header
	argv, replyv reflect.Value //请求中的参数和返回值
	mtype        *methodType
	serv         *service
}

/**
 * @Description: 读取请求中的header
 * @receiver server
 * @param codecFunc
 * @return *codec.Header
 * @return error
 */
func (server *Server) readRequestHeader(codec ServerCodec) (*codec.Header, error) {
	//var header codec.Header
	if err := codec.ReadRequestHeader(NewResponse(), &header); err != nil {
		if err != io.EOF && err != io.ErrUnexpectedEOF {
			log.Println("rpc server:read header error:", err)
		}
		return nil, err
	}
	return &header, nil
}

/**
 * @Description: 读取请求
 * @receiver server
 * @param codecFunc
 * @return *request
 * @return error
 */
func (server *Server) readRequest(codec ServerCodec) (*request, error) {
	//读取请求中的header
	h, err := server.readRequestHeader(codec)
	if err != nil {
		return nil, err
	}
	//构造一个包含了请求中全部信息的数据结构,然后读取请求中的请求体
	req := &request{header: h}
	req.serv, req.mtype, err = server.findService(h.ServiceMethod)
	if err != nil {
		return req, err
	}
	//创建两个入参
	req.argv = req.mtype.newArgv()
	req.replyv = req.mtype.newReplyv()
	iargv := req.argv.Interface()
	//判断是否是指针类型
	if req.argv.Type().Kind() != reflect.Ptr {
		iargv = req.argv.Addr().Interface()
	}
	//将请求反序列化为第一个入参
	if err = codec.ReadRequestBody(iargv); err != nil {
		log.Println("rpc server: read body err:", err)
		return req, err
	}
	//最后把这个包含了请求全部信息的对象返回
	return req, nil
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
	if err := codecFunc.Write(header, body); err != nil {
		log.Println("rpc server:write  response error:", err)
	}
}

/*
func (s *service) call(server *Server, sending *sync.Mutex, wg *sync.WaitGroup, mtype *methodType, req *Request, argv, replyv reflect.Value, codec ServerCodec) {
	if wg != nil {
		defer wg.Done()
	}
	mtype.Lock()
	mtype.numCalls++
	mtype.Unlock()
	function := mtype.method.Func
	// Invoke the method, providing a new value for the reply.
	returnValues := function.Call([]reflect.Value{s.rcvr, argv, replyv})
	// The return value for the method is an error.
	errInter := returnValues[0].Interface()
	errmsg := ""
	if errInter != nil {
		errmsg = errInter.(error).Error()
	}
	server.sendResponse(sending, req, replyv.Interface(), codec, errmsg)
	server.freeRequest(req)
}
*/

//called 信道接收到消息，代表处理没有超时，继续执行 sendResponse
//time.After() 先于 called 接收到消息，说明处理已经超时，called 和 sent 都将被阻塞。在 case <-time.After(timeout) 处调用 sendResponse
func (server *Server) call(codecFunc codec.Codec, req *request, replyMutex *sync.Mutex, handleGroup *sync.WaitGroup, timeout time.Duration) {
	defer handleGroup.Done()
	//TODO 携程泄露
	called := make(chan struct{})
	sent := make(chan struct{})
	go func() {
		err := req.serv.call(req.mtype, req.argv, req.replyv)
		called <- struct{}{}
		if err != nil {
			req.header.Error = err.Error()
			server.sendResponse(codecFunc, req.header, invalidRequest, replyMutex)
			sent <- struct{}{}
			return
		}
		//回复请求
		server.sendResponse(codecFunc, req.header, req.replyv.Interface(), replyMutex)
		sent <- struct{}{}
	}()
	if timeout == 0 {
		<-called
		<-sent
		return
	}
	select {
	case <-time.After(timeout):
		req.header.Error = fmt.Sprintf("rpc server: request handle timeout: expect within %s", timeout)
		server.sendResponse(codecFunc, req.header, invalidRequest, replyMutex)
	case <-called:
		<-sent
	}

}

/**
 * @Description: 链接层 默认的 和net/rpc 包的操作一致 并发处理每一个连接
 * @receiver server
 * @param list
 */
func (server *Server) Accept(list net.Listener) {
	for { //循环等待socket连接建立
		conn, err := list.Accept()
		if err != nil {
			log.Println("rpc.Serve: accept:", err.Error())
			return
		}
		// 当链接建立，就交给子协成进行处理
		go server.ServeConn(conn)
	}
}

/**
 * @Description: 服务注册
 * @receiver server
 * @param list
 */
func (server *Server) Register(rcvr interface{}) error {
	s := newService(rcvr)
	if _, dup := server.serviceMap.LoadOrStore(s.name, s); dup {
		return errors.New("rpc: service already defind: " + s.name)
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
func Accept(listener net.Listener) {
	DefaultServer.Accept(listener)
}

/**
 * @Description: 默认的 Register 方法
 * @param listener
 */
func Register(rcvr interface{}) error {
	return DefaultServer.Register(rcvr)
}
