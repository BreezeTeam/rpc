package rpc

import (
	"rpc/codec"
	"time"
)

//[dubbo 网络传输协议](https://zhuanlan.zhihu.com/p/98562180)
//响应状态
const (
	OK = iota
	CLIENT_TIMEOUT
	SERVER_TIMEOUT
	BAD_REQUEST
	BAD_RESPONSE
	SERVICE_NOT_FOUND
	SERVICE_ERROR
	SERVER_ERROR
	CLIENT_ERROR
	SERVER_THREADPOOL_EXHAUSTED_ERROR
)

//1. 将固定部分进行json编码
//2. 通过固定部分解码为

//固定部分协议
type Protocol struct {
	MagicNumber          int           //魔数,标识RPC传输协议的版本号 16
	Sequence             uint64        //标识唯一请求
	ProtocolLength       int           //协议体长度
	ProtocolHeaderLength int           //协议头长度
	CodecType            codec.Type    //编码类型
	ConnectTimeout       time.Duration //链接超时时间
	HandleTimeout        time.Duration //处理超时时间

	//扩展字段
	Req    int //是否是请求，请求1,0为响应
	Event  int //是否为事件信息
	Status int //标识响应状态
}
