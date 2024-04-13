package main

import (
	"example/hello/internal/connect"
	"log"
	"net"
)

func main() {
	InitServer()
}

func InitServer() {
	listener, err := net.Listen("tcp", ":1935")
	if err != nil {
		log.Printf("Error starting RTMP server %s", err.Error())
		panic(err)
	}
	log.Println("RTMP Server started at port 1935")

	ctx := initStreamContext()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Unable to accept connection %s", err.Error())
			continue
		}
		rtmpConnection := connect.NewConnection(conn, ctx)

		go rtmpConnection.Serve()
	}
}

func initStreamContext() (ctx *connect.StreamContext) {
	ctx = &connect.StreamContext{}
	ctx.Sessions = make(map[string]*connect.Connection)
	ctx.Preview = make(chan string)
	return
}

// Connect 성공 메시지를 클라이언트로 보내는 함수
func sendConnectSuccessMessage(conn net.Conn) error {
	// "NetConnection.Connect.Success" 메시지를 생성하여 클라이언트로 보냄
	_, err := conn.Write([]byte("NetConnection.Connect.Success"))
	if err != nil {
		return err
	}

	return nil
}
