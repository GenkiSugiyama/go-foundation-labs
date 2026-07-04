package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"time"
)

func main() {
	// net.Dial()は指定されたプロトコルで指定されたアドレスに接続するための関数
	conn, err := net.DialTimeout("tcp", "local:8080", 3*time.Second)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(10 * time.Second))

	stdin := bufio.NewScanner(os.Stdin)
	server := bufio.NewScanner(conn)

	for {
		fmt.Print("> ")

		// stdin.Scan()は標準入力からの入力を読み取るための関数
		// 入力があるとtrueを返し、入力がない場合やエラーが発生した場合はfalseを返す
		if !stdin.Scan() {
			break
		}

		text := stdin.Text()
		// "exit"と入力されたらループを抜ける
		if text == "exit" {
			break
		}

		fmt.Fprintln(conn, text)

		// server.Scan()はサーバーからの応答を読み取るための関数
		// サーバーからの応答があるとtrueを返し、サーバーが接続を閉じた場合やエラーが発生した場合はfalseを返す
		if server.Scan() {
			fmt.Println("echo:", server.Text())
		}
	}

	if err := stdin.Err(); err != nil {
		log.Println("stdin error:", err)
	}

	if err := server.Err(); err != nil {
		log.Println("server error:", err)
	}
}
