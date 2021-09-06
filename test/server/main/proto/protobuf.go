//通讯协议处理，主要处理封包和解包的过程
package proto

import (
	"bytes"
	"encoding/binary"
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

//字段长度
const (
	MagicNumber               = 0x6aed56e8
	ConstMagicNumber          = 4
	ConstProtocolLength       = 4
	ConstProtocolHeaderLength = 4
	ConstCodecType            = 4
	ConstConnectTimeout       = 8
	ConstHandleTimeout        = 8
	ConstSequence             = 16

	ConstHeaderLength = 48
	ConstDataLength   = 4
)

//扩展字段
type Extend map[string]interface{}

var extend = Extend{
	"req":    1,  //是否是请求，请求1,0为响应 1位
	"event":  0,  //是否为事件信息 1位
	"status": OK, //标识响应状态 4位
}

//固定部分协议
type Protocol struct {
	MagicNumber          int32          //魔数,标识RPC传输协议的版本号,4位int类型，得到之后，将其转为16进制 4位
	ProtocolLength       int32          //协议体长度 4位
	ProtocolHeaderLength int32          //协议头长度 4位
	CodecType            codec.TypeCode //编码类型 int32 4位
	ConnectTimeout       time.Duration  //链接超时时间 8位
	HandleTimeout        time.Duration  //处理超时时间 8位
	Sequence             uint64         //标识唯一请求 16位
}

func PacketHeader(codeType codec.TypeCode, connectTimeout time.Duration, handlerTimeout time.Duration,
	extend []byte) []byte {

	header := make([]byte,0)

	protocolHeaderLength := ConstHeaderLength
	protocolLength := ConstHeaderLength + len(extend)

	header = append(header, IntToInt32ToBytes(MagicNumber)...)
	header = append(header, IntToInt32ToBytes(protocolLength)...)
	header = append(header, IntToInt32ToBytes(protocolHeaderLength)...)
	header = append(header, Int32ToBytes(int32(codeType))...)
	header = append(header, Int64ToBytes(int64(connectTimeout))...)
	header = append(header, Int64ToBytes(int64(handlerTimeout))...)
	header = append(header, extend...)
	return header
}
func UnpackHeader(buffer []byte, readerChannel chan []byte) []byte {
	length := len(buffer)

	var i int
	if length < i+ConstHeaderLength {
		return make([]byte, 0)
	}
	if BytesToInt32ToInt(buffer[i:i+ConstMagicNumber]) == MagicNumber {
		println(string(buffer))
	}
	return make([]byte, 0)

}

////封包
//func Packet(message []byte) []byte {
//	return append(append([]byte(ConstHeader), IntToBytes(len(message))...), message...)
//}
//
////解包
//func Unpack(buffer []byte, readerChannel chan []byte) []byte {
//	length := len(buffer)
//
//	var i int
//	for i = 0; i < length; i = i + 1 {
//		if length < i+ConstHeaderLength+ConstSaveDataLength {
//			break
//		}
//		if string(buffer[i:i+ConstHeaderLength]) == ConstHeader {
//			messageLength := BytesToInt(buffer[i+ConstHeaderLength : i+ConstHeaderLength+ConstSaveDataLength])
//			if length < i+ConstHeaderLength+ConstSaveDataLength+messageLength {
//				break
//			}
//			data := buffer[i+ConstHeaderLength+ConstSaveDataLength : i+ConstHeaderLength+ConstSaveDataLength+messageLength]
//			readerChannel <- data
//
//			i += ConstHeaderLength + ConstSaveDataLength + messageLength - 1
//		}
//	}
//
//	if i == length {
//		return make([]byte, 0)
//	}
//	return buffer[i:]
//}

func IntToInt32ToBytes(n int) []byte {
	x := int32(n)
	bytesBuffer := bytes.NewBuffer([]byte{})
	binary.Write(bytesBuffer, binary.BigEndian, x)
	return bytesBuffer.Bytes()
}

func BytesToInt32ToInt(b []byte) (x int) {
	var m int32
	bytesBuffer := bytes.NewBuffer(b)
	binary.Read(bytesBuffer, binary.BigEndian, &m)
	return int(m)
}

func Int32ToBytes(n int32) []byte {
	bytesBuffer := bytes.NewBuffer([]byte{})
	binary.Write(bytesBuffer, binary.BigEndian, n)
	return bytesBuffer.Bytes()
}

func BytesToInt32(b []byte) (x int32) {
	bytesBuffer := bytes.NewBuffer(b)
	binary.Read(bytesBuffer, binary.BigEndian, &x)
	return x
}

func Int64ToBytes(n int64) []byte {
	bytesBuffer := bytes.NewBuffer([]byte{})
	binary.Write(bytesBuffer, binary.BigEndian, n)
	return bytesBuffer.Bytes()
}

func BytesToInt64(b []byte) (x int64) {
	bytesBuffer := bytes.NewBuffer(b)
	binary.Read(bytesBuffer, binary.BigEndian, &x)
	return x
}
func Uint64ToBytes(n uint64) []byte {
	bytesBuffer := bytes.NewBuffer([]byte{})
	binary.Write(bytesBuffer, binary.BigEndian, n)
	return bytesBuffer.Bytes()
}

func BytesToUint64(b []byte) (x uint64) {
	bytesBuffer := bytes.NewBuffer(b)
	binary.Read(bytesBuffer, binary.BigEndian, &x)
	return x
}
func Int8ToBytes(n int8) []byte {
	bytesBuffer := bytes.NewBuffer([]byte{})
	binary.Write(bytesBuffer, binary.BigEndian, n)
	return bytesBuffer.Bytes()
}

func BytesToInt8(b []byte) (x int8) {
	bytesBuffer := bytes.NewBuffer(b)
	binary.Read(bytesBuffer, binary.BigEndian, &x)
	return x
}
