package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
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

	// クライアントからのリクエストがきたらその接続のコネクションを返す
	// コネクションはクライアントとの実際の接続
	// 複数クライアントからのリクエストを処理するためにAccept()をループで呼び出す
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("accept error:", err)
			continue
		}

		// それぞれの接続をゴルーチンで処理することでメインループがブロックされるのを防ぐ
		go handleConn(conn)
	}
}

func handleConn(conn net.Conn) {
	defer conn.Close()

	log.Println("client connected:", conn.RemoteAddr())
	defer log.Println("client disconnected:", conn.RemoteAddr())

	// クライアントからのリクエストを読み取るためのスキャナーを作成する
	// クライアントとのコネクションをReaderとしてスキャナーに渡すことで、クライアントからのリクエストを読み取ることができる
	scanner := bufio.NewScanner(conn)
	// Scan()でトークンごとにクライアントからのリクエストを読み取る
	// クライアントからのテキストはtcp層では単なるバイトストリームなので、アプリケーション層でトークンの単位を決めてどこからどこまでを1つのメッセージ（リクエスト）とみなすかを定義する必要がある
	// 明示的にトークンの単位を設定していないのでデフォルトの改行コードで区切られた行単位で読み取る
	// Scan()は接続と閉じたときや読み取りエラー等が発生したときにfalseを返すのでループを抜ける
	for scanner.Scan() {
		// Scan()で読み取ったトークンを文字列として取得
		text := scanner.Text()
		// コネクションをWriterとしてFprintln()に渡すことでクライアントにテキストを送信する
		fmt.Fprintln(conn, text)
	}

	// Scan()でエラーが発生した場合はErr()でエラーを取得することができる
	if err := scanner.Err(); err != nil {
		log.Println("read error:", err)
	}
}
