package main

import (
	"example/hello/internal"
	"log"
	"net"
)

func main() {
	InitServer()
}

// InitServer RTMP 서버를 초기화하고 시작하는 함수입니다.
// RTMP 서버는 TCP 포트 1935에서 리스닝을 시작합니다.
func InitServer() {
	listener, err := net.Listen("tcp", ":1935")
	if err != nil {
		log.Printf("Error starting RTMP server %s", err.Error())
		panic(err)
	}
	log.Println("RTMP Server started at port 1935")

	ctx := initStreamContext()

	go internal.InitPreviewServer(ctx)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Unable to accept connection %s", err.Error())
			continue
		}
		connection := internal.NewConnection(conn, ctx)

		go connection.Serve()
	}
}

// initStreamContext 스트리밍에 필요한 전역 상태를 관리하는 컨텍스트를 초기화합니다.
// 세션 관리를 위한 map, 미리보기 채널을 초기화합니다.
func initStreamContext() (ctx *internal.StreamContext) {
	ctx = &internal.StreamContext{}
	ctx.Sessions = make(map[string]*internal.Connection)
	ctx.Preview = make(chan string)
	return
}
