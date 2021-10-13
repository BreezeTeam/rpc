## net/rpc包学习

### server 学习

![net_rpc包 (1)](https://gitee.com/Euraxluo/images/raw/master/picgo/net_rpc%E5%8C%85%20(1).jpg)
#### 服务注册:
在进行rpc方法调用前,需要先进行方法注册

```go
func (server *Server) register(rcvr interface{}, name string, useName bool) error {
    //整个工作就是构造service对象,填充属性
    //最后调用`sync.Map.LoadOrStore(sname,s)`方法完成服务注册
	s := new(service)
	s.typ = reflect.TypeOf(rcvr)
	s.rcvr = reflect.ValueOf(rcvr)
	sname := reflect.Indirect(s.rcvr).Type().Name()
	if useName {
		sname = name
	}
	if sname == "" {
		s := "rpc.Register: no service name for type " + s.typ.String()
		log.Print(s)
		return errors.New(s)
	}
	if !token.IsExported(sname) && !useName {
		s := "rpc.Register: type " + sname + " is not exported"
		log.Print(s)
		return errors.New(s)
	}
	s.name = sname

	// Install the methods
	s.method = suitableMethods(s.typ, true)

	if len(s.method) == 0 {
		str := ""

		// To help the user, see if a pointer receiver would work.
		method := suitableMethods(reflect.PtrTo(s.typ), false)
		if len(method) != 0 {
			str = "rpc.Register: type " + sname + " has no exported methods of suitable type (hint: pass a pointer to value of that type)"
		} else {
			str = "rpc.Register: type " + sname + " has no exported methods of suitable type"
		}
		log.Print(str)
		return errors.New(str)
	}

	if _, dup := server.serviceMap.LoadOrStore(sname, s); dup {
		return errors.New("rpc: service already defined: " + sname)
	}
	return nil
}
```



#### 链接处理：
1. 循环等待socket连接建立，并且开启子协程处理每一个链接`go server.ServeConn(conn)`
2. 在ServeConn中，参数是一个链接，该方法首先创建了编解码器gobServerCodec,然后使用`server.ServeCodec(srv)`利用编解码器对链接进行处理
```go
// Accept 接受侦听器上的连接并提供请求
// 对于每个传入连接。 接受块直到侦听器
// 返回一个非零错误。 调用者通常在一个
// 去语句。
func (server *Server) Accept(lis net.Listener) {
	for {
		conn, err := lis.Accept()
		if err != nil {
			log.Print("rpc.Serve: accept:", err.Error())
			return
		}
		go server.ServeConn(conn)
	}
}
// ServeConn 在单个连接上运行服务器。
// ServeConn 阻塞，服务连接直到客户端挂断。
// 调用者通常在 go 语句中调用 ServeConn。
// ServeConn 使用 gob 线格式（见包 gob）
// 使用通用编解码器，请使用 ServeCodec。
// 有关并发访问的信息，请参阅 NewClient 的注释。
func (server *Server) ServeConn(conn io.ReadWriteCloser) {
	buf := bufio.NewWriter(conn)
	srv := &gobServerCodec{
		rwc:    conn,
		dec:    gob.NewDecoder(conn),
		enc:    gob.NewEncoder(buf),
		encBuf: buf,
	}
    //传入执行的解码器，进行解码
    server.ServeCodec(srv)
}

```
#### 链接数据读取,请求处理:

**编解码接口:**

```go
type ServerCodec interface {
	ReadRequestHeader(*Request) error
	ReadRequestBody(interface{}) error
	WriteResponse(*Response, interface{}) error

	// Close can be called multiple times and must be idempotent.
	Close() error
}
type gobServerCodec struct {
	rwc    io.ReadWriteCloser
	dec    *gob.Decoder
	enc    *gob.Encoder
	encBuf *bufio.Writer
	closed bool
}
```

**连接请求读取方法**

```go
func (server *Server) ServeCodec(codec ServerCodec) {
	sending := new(sync.Mutex)
	wg := new(sync.WaitGroup)
    //死循环读取请求,每从链接中读取到一个请求,就交由service.call 该接口方法进行服务调用
	for {
        //通过server.readRequest(codec)利用编解码器接口读取请求
		service, mtype, req, argv, replyv, keepReading, err := server.readRequest(codec)
		if err != nil {
			if debugLog && err != io.EOF {
				log.Println("rpc:", err)
			}
			if !keepReading {
				break
			}
			// send a response if we actually managed to read a header.
			if req != nil {
				server.sendResponse(sending, req, invalidRequest, codec, err.Error())
				server.freeRequest(req)
			}
			continue
		}
		wg.Add(1)
        
        //开启一个协程,通过请求决定调用的方法进行方法调用
        //此处通过mtype.method.Func 得到方法的函数对象
        //最后通过reflect来进行最后的函数对调用 res = function.Call([]reflect.Value{s.rcvr,argv,replyv})
        
       	//将sending互斥锁对象交给了该协程.保证了每次请求的返回不会进行冲突.
		go service.call(server, sending, wg, mtype, req, argv, replyv, codec)
	}
    //我们在推出前需要保证每一个服务请求都处理完毕.即等待所有请求的service.call 协程完成调用
    
	// We've seen that there are no more requests.
	// Wait for responses to be sent before closing codec.
	wg.Wait()
	codec.Close()
}

func (server *Server) readRequest(codec ServerCodec) (service *service, mtype *methodType, req *Request, argv, replyv reflect.Value, keepReading bool, err error) {
    //得到读取结果.顺利的话将会读取到服务名,方法名,并且会从`server.serviceMap`这个`sync.Map`结构中获取到注册的服务信息(`svci,ok:=server.ServiceMap.Load(serviceName)`).然后从服务信息中获取方法的详细信息:mtype=svci.(*service).method[methodName];如果读取发生任何错误,keepReading将会返回默认值false
	service, mtype, req, keepReading, err = server.readRequestHeader(codec)
	if err != nil {
		if !keepReading {
			return
		}
		// discard body
		codec.ReadRequestBody(nil)
		return
	}

    //通过mtype.ArgType 的判断,利用reflect.New(),得到argv这样一个指针类型的请求参数
	// Decode the argument value.
	argIsValue := false // if true, need to indirect before calling.
	if mtype.ArgType.Kind() == reflect.Ptr {
		argv = reflect.New(mtype.ArgType.Elem())
	} else {
		argv = reflect.New(mtype.ArgType)
		argIsValue = true
	}
    //此处直接使用 对应解码器的编解码方法,此处是`gob.Decoder.Decode(body)`,填充argv指针
	// argv guaranteed to be a pointer now.
	if err = codec.ReadRequestBody(argv.Interface()); err != nil {
		return
	}
	if argIsValue {
		argv = argv.Elem()
	}
	//得到replyv指针
	replyv = reflect.New(mtype.ReplyType.Elem())
	//根据replyv的类型,正确的处理指针,最后返回
	switch mtype.ReplyType.Elem().Kind() {
	case reflect.Map:
		replyv.Elem().Set(reflect.MakeMap(mtype.ReplyType.Elem()))
	case reflect.Slice:
		replyv.Elem().Set(reflect.MakeSlice(mtype.ReplyType.Elem(), 0, 0))
	}
	return
}
```
