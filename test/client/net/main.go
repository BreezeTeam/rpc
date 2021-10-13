package main;



import (

	"fmt"

	"log"

	"net/rpc"

)



type Params struct {

	Width, Height int;

}



func main() {

	//连接远程rpc服务

	//这里使用jsonrpc.Dial

	rpc, err := rpc.Dial("tcp", "127.0.0.1:8080");

	if err != nil {

		log.Fatal(err);

	}

	ret := 0;

	//调用远程方法

	//注意第三个参数是指针类型

	err2 := rpc.Call("Rect.Area", Params{50, 100}, &ret);

	if err2 != nil {

		log.Fatal(err2);

	}

	fmt.Println(ret);

	for i:=0;i<100;i++{

		err3 := rpc.Call("Rect.Perimeter", Params{20, 100}, &ret);

		if err3 != nil {

			log.Fatal(err3);

		}
		fmt.Println(ret);
	}



}

