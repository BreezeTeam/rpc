package metadata

//消息类型
type MessageType int

const (
	Error    MessageType = iota //错误
	Request                     //请求
	Response                    //响应
	Event                       //事件
)

//消息状态
type StatusType int

const (
	OK                                = iota //OK
	CLIENT_TIMEOUT                           //客户端超时
	SERVER_TIMEOUT                           //服务端超时
	BAD_REQUEST                              //错误请求
	BAD_RESPONSE                             //错误的响应
	SERVICE_NOT_FOUND                        //服务未找到
	SERVICE_ERROR                            //服务错误
	SERVER_ERROR                             //服务端错误
	CLIENT_ERROR                             //客户端错误
	SERVER_THREADPOOL_EXHAUSTED_ERROR        //服务器线程资源错误
)

//Message表示关于通信的详细信息，在出现错误的情况下，body可能为nil。
type Message struct {
	Id       string      //消息Id,应该没有递增的需求,因此UUID就可以了
	Type     MessageType //消息类型
	Target   string      //Service
	Method   string      //Method
	Endpoint string      //接入点
	Error    string      //错误信息

	// The values read from the socket
	Header map[string]string //Header KV
	Body   []byte            //body部分,放置请求体和响应体
}

//扩展header的 hdr
const (
	CONNECT_TIMEOUT = "X-ConnectTimeout" //链接超时时间
	HANDLE_TIMEOUT  = "X-HandleTimeout"  //处理超时时间
	STATUS_TYPE     = "X-StatusType"     //消息表达的状态
)
