package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"rpc"
	"rpc/codec"
	"time"
)
func startServer(addr chan string){
	listener,err := net.Listen("tcp",":0")
	if err != nil{
		log.Fatal("network error: ",err)
	}
	log.Println("start rpc client on ",listener.Addr())
	addr <-listener.Addr().String()
	rpc.Accept(listener)
}

func main() {
	log.SetFlags(0)

	//通过信道来向函数发送addr
	addr := make(chan string)
	//坚听信道中发送的addr,并且将listener对象返回到信道
	go startServer(addr)

	conn, _ := net.Dial("tcp",<-addr)
	defer conn.Close()

	time.Sleep(time.Second)

	_ = json.NewEncoder(conn).Encode(rpc.DefaultOption)
	codecFunc := codec.NewGobCodec(conn)
	for i := 0; i < 5; i++ {
		h := &codec.Header{
			ServiceMethod: "Foo.Sum",
			Seq:           uint64(i),
		}
		_ = codecFunc.Write(h, fmt.Sprintf("rpc req %d", h.Seq))
		_ = codecFunc.ReadHeader(h)
		var reply string
		_ = codecFunc.ReadBody(&reply)
		log.Println("reply:", reply)
	}
}
