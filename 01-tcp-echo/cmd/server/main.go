package main

import (
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/GenkiSugiyama/go-foundation-labs/01-tcp-echo/internal/echo"
)

func main() {
	// アプリケーション層の下層のトランスポート層のプロトコルであるTCPを使用して、ポート8080で待ち受けるリスナーを作成する
	// アプリケーションが通信を行うための土台を提供する
	// リスナーはトランスポート層の窓口
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()

	log.Println("tcp echo server is listening on :8080")

	quit := make(chan os.Signal, 1)
	// Ctrl + c が押下されるとプロセスにSIGINTが送られる
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		// signal.Notify()でSIGINTシグナルを受け取ると listenerClose()が発火する
		<-quit
		log.Println("shutting down server...")
		listener.Close()
	}()

	for {
		// listener.Close()が発火するとlistener.Accept()がエラーでもどりサーバーが終了する
		conn, err := listener.Accept()
		if err != nil {
			log.Println("accept stopped:", err)
			return
		}

		go handleConn(conn)
	}
}

func handleConn(conn net.Conn) {
	defer conn.Close()

	log.Println("client connected:", conn.RemoteAddr())
	defer log.Println("client disconnected:", conn.RemoteAddr())

	err := echo.Handle(conn, conn)

	// Scan()でエラーが発生した場合はErr()でエラーを取得することができる
	if err != nil {
		log.Println("read error:", err)
	}
}
