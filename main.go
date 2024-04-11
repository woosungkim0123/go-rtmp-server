package main

import (
	"example/hello/internal"
	"fmt"
	"log"
	"net"
)

func main() {
	tcpAddr, err := net.ResolveTCPAddr("tcp", ":1935")
	if err != nil {
		log.Panicf("Address resolution failed: %+v", err)
	}

	listener, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		log.Panicf("Listener creation failed: %+v", err)
	}

	defer listener.Close()
	fmt.Println("Listening on", listener.Addr().String())

	for {
		rwc, err := listener.Accept()
		if err != nil {

			log.Printf("Failed to accept connection: %+v", err)
			continue
		} else {
			fmt.Println("Connection accepted from", rwc.RemoteAddr().String())
		}
		conn := internal.NewConn(rwc)
		go internal.Serve(conn)

	}
}

// Connect 요청 처리 함수
func handleConnectRequest(conn net.Conn) error {
	// Connect 요청 처리 로직
	// 여기에 필요한 처리 코드를 추가하세요.

	// Connect 요청이 성공한 경우, 성공 메시지를 클라이언트로 보냄
	if err := sendConnectSuccessMessage(conn); err != nil {
		return err
	}

	return nil
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
