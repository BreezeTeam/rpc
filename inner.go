package rpc

//内部对象 请求
type Request struct {
	ServiceMethod string
	Seq           uint64
	next          *Request
}

// 内部对象 返回
type Response struct {
	ServiceMethod string
	Seq           uint64
	Error         string
	next          *Response
}
