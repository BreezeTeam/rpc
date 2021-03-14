package main

import (
	"fmt"
	"log"
	"net"
	"rpc"
	"sync"
	"time"
)
func startServer(addr chan string){
	listener,err := net.Listen("tcp",":0")
	if err != nil{
		log.Fatal("network error: ",err)
	}
	log.Println("start rpc client on ",listener.Addr())
	//监听成功,将坚挺的地址返回到信道
	addr <-listener.Addr().String()
	rpc.Accept(listener)
}

func main() {
	//log.SetFlags(0)
	////创建信道,用于返回listener对象
	//addr := make(chan string)
	////异步申请地址, 并且将listener对象返回到信道
	//go startServer(addr)
	////阻塞,直到信道返回到信道
	//conn, _ := net.Dial("tcp",<-addr)
	//defer conn.Close()
	//
	//
	//	time.Sleep(time.Second)
	////将Option,编码到conn中,格式化为json
	////_ = json.NewEncoder(conn).Encode(rpc.DefaultOption)
	//_ = json.NewEncoder(conn).Encode(&rpc.Option{
	//	MagicNumber:rpc.MagicNumber,
	//	CodecType:codec.JsonType,
	//})
	//codecFunc := codec.NewJsonCodec(conn)
	//
	//for i := 0; i < 5; i++ {
	//	h := &codec.Header{
	//		ServiceMethod: "Foo.Sum",
	//		Seq:           uint64(i),
	//	}
	//	_ = codecFunc.Write(h, fmt.Sprintf("rpc req %d", h.Seq))
	//	_ = codecFunc.ReadHeader(h)
	//	var reply string
	//	_ = codecFunc.ReadBody(&reply)
	//	log.Println("reply:", reply)
	//}

	log.SetFlags(0)
	//创建信道,用于返回listener对象
	addr := make(chan string)
	//异步申请地址, 并且将listener对象返回到信道
	go startServer(addr)
	//阻塞,直到信道返回到信道
	client, _ := rpc.Dial("tcp",<-addr)
	defer client.Close()
	time.Sleep(time.Second)
	//将Option,编码到conn中,格式化为json

	var wg sync.WaitGroup
	//i 就是请求编号
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int){
			defer wg.Done()
			args:=fmt.Sprintf("rpc req %d", i)
			var reply string
			if err:= client.Call("Foo.Sum",args,&reply);err !=nil{
				log.Fatal("call Foo.Sum error:",err)
			}
			log.Println("reply:", reply)

		}(i)
	}
	wg.Wait()
}
