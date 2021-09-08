package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"os"
	"rpc/codec"
	"sync"
	"time"
)

func sender(conn net.Conn) {
	header := DefaultProtocolHeader(codec.GobTypeCode, 1, 2, DefaultExtend())
	packedHeader, err := header.PacketHeader()
	if err != nil {
		fmt.Println("send failed: ", err)
	}
	conn.Write(packedHeader)
	p := NewProtocolDataFactory()
	for i := 0; i < 100; i++ {
		data := "{\"Id\":1,\"Name\":\"golang\",\"Message\":\"message\"}"
		conn.Write(p.ProductProtocolData([]byte(data)).PacketData())
	}
	fmt.Println("send over")
}

func main() {
	server := "127.0.0.1:9988"
	tcpAddr, err := net.ResolveTCPAddr("tcp4", server)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s", err.Error())
		os.Exit(1)
	}

	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s", err.Error())
		os.Exit(1)
	}

	defer conn.Close()
	fmt.Println("connect success")
	go sender(conn)
	for {
		time.Sleep(1 * 1e9)
	}
}

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

	ConstSequence = 16

	ConstHeaderLength = 32
	ConstDataLength   = 4
)

//固定部分协议
type ProtocolHeader struct {
	MagicNumber          int32          //魔数,标识RPC传输协议的版本号,4位int类型，得到之后，将其转为16进制 4位
	ProtocolLength       int32          //协议体长度 4位
	ProtocolHeaderLength int32          //协议头长度 4位
	CodecType            codec.TypeCode //编码类型 int32 4位
	ConnectTimeout       time.Duration  //链接超时时间 8位
	HandleTimeout        time.Duration  //处理超时时间 8位
	Extend               *Extend        //扩展字段
}

//默认协议构造器
func DefaultProtocolHeader(codeType codec.TypeCode, connectTimeout time.Duration, handlerTimeout time.Duration, extend *Extend) *ProtocolHeader {
	return &ProtocolHeader{
		MagicNumber:          MagicNumber,
		ProtocolLength:       ConstHeaderLength,
		ProtocolHeaderLength: ConstHeaderLength,
		CodecType:            codeType,
		ConnectTimeout:       connectTimeout,
		HandleTimeout:        handlerTimeout,
		Extend:               extend,
	}
}

//包装协议
func (protocolHeader *ProtocolHeader) PacketHeader() ([]byte, error) {

	extend, err := protocolHeader.Extend.ExtendEncode()
	if err != nil {
		return nil, err
	}
	header := make([]byte, 0)
	protocolHeader.ProtocolLength = int32(len(extend)) + protocolHeader.ProtocolLength

	header = append(header, Int32ToBytes(protocolHeader.MagicNumber)...)
	header = append(header, Int32ToBytes(protocolHeader.ProtocolLength)...)
	header = append(header, Int32ToBytes(protocolHeader.ProtocolHeaderLength)...)
	header = append(header, Int32ToBytes(int32(protocolHeader.CodecType))...)
	header = append(header, Int64ToBytes(int64(protocolHeader.ConnectTimeout))...)
	header = append(header, Int64ToBytes(int64(protocolHeader.HandleTimeout))...)
	header = append(header, extend...)
	return header, nil
}

//解包
func UnpackHeader(buffer []byte, readerChannel chan []byte) []byte {

	length := len(buffer)
	if length < ConstHeaderLength {
		return make([]byte, 0)
	}

	for i := 0; i < length; i++ {
		magicNumber := BytesToInt32ToInt(buffer[i : i+ConstMagicNumber])
		if magicNumber == MagicNumber {
			println(magicNumber)
			protocolLength := BytesToInt32ToInt(buffer[i+ConstMagicNumber : i+ConstMagicNumber+ConstProtocolLength])
			println(protocolLength)
			if length < i+protocolLength {
				break
			}
			protocolHeaderLength := BytesToInt32ToInt(buffer[i+ConstMagicNumber+ConstProtocolLength : i+ConstMagicNumber+ConstProtocolLength+ConstProtocolHeaderLength])
			println(protocolHeaderLength)
			codecType := BytesToInt32ToInt(buffer[i+ConstMagicNumber+ConstProtocolLength+ConstProtocolHeaderLength : i+ConstMagicNumber+ConstProtocolLength+ConstProtocolHeaderLength+ConstCodecType])
			println(codecType)
			connectTimeout := BytesToInt32ToInt(buffer[i+ConstMagicNumber+ConstProtocolLength+ConstProtocolHeaderLength+ConstCodecType : i+ConstMagicNumber+ConstProtocolLength+ConstProtocolHeaderLength+ConstCodecType+ConstConnectTimeout])
			println(connectTimeout)
			handlerTimeout := BytesToInt32ToInt(buffer[i+ConstMagicNumber+ConstProtocolLength+ConstProtocolHeaderLength+ConstCodecType+ConstConnectTimeout : i+ConstMagicNumber+ConstProtocolLength+ConstProtocolHeaderLength+ConstCodecType+ConstConnectTimeout+ConstHandleTimeout])
			println(handlerTimeout)
			extend := buffer[i+ConstMagicNumber+ConstProtocolLength+ConstProtocolHeaderLength+ConstCodecType+ConstConnectTimeout+ConstHandleTimeout : i+ConstMagicNumber+ConstProtocolLength+ConstProtocolHeaderLength+ConstCodecType+ConstConnectTimeout+ConstHandleTimeout+(protocolLength-protocolHeaderLength)]
			println(string(extend))
			readerChannel <- extend
			i += ConstMagicNumber + ConstProtocolLength + ConstProtocolHeaderLength + ConstCodecType + ConstConnectTimeout + ConstHandleTimeout + (protocolLength - protocolHeaderLength)
		}

	}
	return make([]byte, 0)
}

//扩展字段
type Extend map[string]interface{}

func DefaultExtend() *Extend {
	return &Extend{
		"req":    1,  //是否是请求，请求1,0为响应 1位
		"event":  0,  //是否为事件信息 1位
		"status": OK, //标识响应状态 4位
	}
}
func (extend *Extend) ExtendEncode() ([]byte, error) {
	return json.Marshal(extend)
}

//协议数据
type ProtocolData struct {
	DataLength int32  //数据长度 4位
	Sequence   uint64 //标识唯一请求 16位
	Data       []byte //数据
}

type ProtocolDataFactory struct {
	InitSequence      uint64      //本次工厂生产的初始化序列号
	mutex             *sync.Mutex //锁
	ProtocolDataCount int32       //本次工厂生产的货物数量
}

func NewProtocolDataFactory() *ProtocolDataFactory {
	return &ProtocolDataFactory{
		InitSequence:      uint64(rand.Intn((1 << 32) - 1)),
		mutex:             new(sync.Mutex),
		ProtocolDataCount: 0,
	}
}
func (p *ProtocolDataFactory) ProductProtocolData(data []byte) *ProtocolData {
	p.mutex.Lock()
	defer func() {
		p.ProtocolDataCount += 1
		p.mutex.Unlock()
	}()
	return &ProtocolData{
		DataLength: int32(len(data)),
		Data:       data,
		Sequence:   p.InitSequence + uint64(p.ProtocolDataCount),
	}
}

//包装协议
func (p *ProtocolData) PacketData() []byte {
	data := make([]byte, 0)
	data = append(data, Int64ToBytes(int64(p.DataLength))...)
	data = append(data, Int64ToBytes(int64(p.Sequence))...)
	data = append(data, p.Data...)
	return data
}

//解包
func UnpackData(buffer []byte, readerChannel chan []byte) []byte {
	length := len(buffer)

	var i int
	if length < i+ConstHeaderLength {
		return make([]byte, 0)
	}
	if BytesToInt32ToInt(buffer[i:i+ConstMagicNumber]) == MagicNumber {
		println(string(buffer))
	}
	return make([]byte, 0)

	//func Unpack(buffer []byte, readerChannel chan []byte) []byte {
	//length := len(buffer)
	//
	//var i int
	//for i = 0; i < length; i = i + 1 {
	//	if length < i+ConstHeaderLength+ConstSaveDataLength {
	//		break
	//	}
	//	if string(buffer[i:i+ConstHeaderLength]) == ConstHeader {
	//		messageLength := BytesToInt(buffer[i+ConstHeaderLength : i+ConstHeaderLength+ConstSaveDataLength])
	//		if length < i+ConstHeaderLength+ConstSaveDataLength+messageLength {
	//			break
	//		}
	//		data := buffer[i+ConstHeaderLength+ConstSaveDataLength : i+ConstHeaderLength+ConstSaveDataLength+messageLength]
	//		readerChannel <- data
	//
	//		i += ConstHeaderLength + ConstSaveDataLength + messageLength - 1
	//	}
	//}
	//
	//if i == length {
	//	return make([]byte, 0)
	//}
	//return buffer[i:]
	//}

}

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
