package echo

import (
	"bufio"
	"fmt"
	"io"
)

func Handle(r io.Reader, w io.Writer) error {
	// クライアントからのリクエストを読み取るためのスキャナーを作成する
	// クライアントとのコネクションをReaderとしてスキャナーに渡すことで、クライアントからのリクエストを読み取ることができる
	scanner := bufio.NewScanner(r)
	// Scan()でトークンごとにクライアントからのリクエストを読み取る
	// クライアントからのテキストはtcp層では単なるバイトストリームなので、アプリケーション層でトークンの単位を決めてどこからどこまでを1つのメッセージ（リクエスト）とみなすかを定義する必要がある
	// 明示的にトークンの単位を設定していないのでデフォルトの改行コードで区切られた行単位で読み取る
	// Scan()は接続と閉じたときや読み取りエラー等が発生したときにfalseを返すのでループを抜ける
	for scanner.Scan() {
		fmt.Fprintln(w, scanner.Text())
	}

	return scanner.Err()
}
