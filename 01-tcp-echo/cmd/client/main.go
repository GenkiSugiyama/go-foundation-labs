package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
)

func main() {
	// net.Dial()は指定されたプロトコルで指定されたアドレスに接続するための関数
	conn, err := net.Dial("tcp", "localhost:8080")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	// サーバーへのコネクションをWriterとしてFprintln()に渡すことでサーバーにテキストを送信する
	fmt.Fprintln(conn, "Hello from go client")

	// コネクションをReaderとしてスキャナーに渡すことで、サーバーからのレスポンスを読み取ることができる
	scanner := bufio.NewScanner(conn)
	// サーバーはクライアントからのリクエストを読み取ってそのテキストをそのままクライアントに返している
	// クライアントはサーバーからのレスポンスを読み取ったらそれを表示して終了する
	if scanner.Scan() {
		fmt.Println("response:", scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		log.Println("read error:", err)
	}

	os.Exit(0)
}
