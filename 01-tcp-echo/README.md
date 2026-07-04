# 01-tcp-echo

## 目的

この課題では、Goで小さなTCP echo server / clientを作りながら、TCP通信の基本を学びます。

HTTPを学ぶ前に、HTTPの下で使われることが多いTCPの動きを確認することが目的です。

## この課題で学ぶこと

- TCPはコネクションを確立して通信すること
- サーバーは特定のポートで待ち受けること
- クライアントはIPアドレスとポートを指定して接続すること
- 1つのTCP接続では、双方向にデータを送受信できること
- Goでは `net.Listen` / `net.Dial` を使ってTCP通信を書けること
- 複数クライアントを扱うには goroutine が必要になること
- TCPは「メッセージ単位」ではなく「バイトストリーム」であること

## 完成イメージ

サーバーを起動します。

```bash
go run ./cmd/server
```

別ターミナルから接続します。

```bash
nc localhost 8080
```

入力します。

```text
hello
```

サーバーから同じ内容が返ってきます。

```text
hello
```

つまり、クライアントが送った文字列をそのまま返す echo server を作ります。

## ディレクトリ構成

```text
01-tcp-echo/
├── README.md
├── go.mod
├── cmd/
│   ├── server/
│   │   └── main.go
│   └── client/
│       └── main.go
└── internal/
    └── echo/
        └── server.go
```

最初は `cmd/server/main.go` だけでも構いません。  
慣れてきたら `internal/echo/server.go` に処理を分けます。

## Step 1: Goモジュールを作成する

```bash
mkdir -p go-foundation-labs/01-tcp-echo
cd go-foundation-labs/01-tcp-echo

go mod init example.com/go-foundation-labs/01-tcp-echo
```

確認します。

```bash
ls
```

```text
go.mod
```

## Step 2: 最小のTCPサーバーを作る

まずは `cmd/server/main.go` を作成します。

```bash
mkdir -p cmd/server
touch cmd/server/main.go
```

実装します。

```go
package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
)

func main() {
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()

	log.Println("tcp echo server listening on :8080")

	conn, err := listener.Accept()
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	log.Println("client connected:", conn.RemoteAddr())

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		text := scanner.Text()
		fmt.Fprintln(conn, text)
	}

	if err := scanner.Err(); err != nil {
		log.Println("read error:", err)
	}

	log.Println("client disconnected")
}
```

## Step 3: サーバーを起動する

```bash
go run ./cmd/server
```

次のように表示されればOKです。

```text
tcp echo server listening on :8080
```

この状態で、サーバーは `8080` 番ポートでTCP接続を待ち受けています。

## Step 4: ncで接続する

別ターミナルを開いて、次を実行します。

```bash
nc localhost 8080
```

適当に入力します。

```text
hello
```

同じ文字列が返ってきます。

```text
hello
```

別の文字列も試します。

```text
go tcp
```

```text
go tcp
```

終了するには `Ctrl+C` を押します。

## Step 5: ここで確認すること

この時点で確認したいことは以下です。

### サーバーはポートで待ち受ける

```go
net.Listen("tcp", ":8080")
```

これは、TCPで `8080` 番ポートを待ち受けるという意味です。

```text
サーバー
  ↓
TCP 8080番ポートで待機
```

### クライアントはIPアドレスとポートに接続する

```bash
nc localhost 8080
```

これは、`localhost` の `8080` 番ポートに接続するという意味です。

```text
クライアント
  ↓
localhost:8080 に接続
```

### TCP接続は accept される

```go
conn, err := listener.Accept()
```

`Accept()` は、クライアントからの接続を受け付けます。

接続が来るまで、この行で待機します。

### TCPは双方向通信できる

```go
scanner := bufio.NewScanner(conn)
```

でクライアントから読み取り、

```go
fmt.Fprintln(conn, text)
```

でクライアントへ書き返しています。

同じ `conn` を使って、読み取りも書き込みもできます。

## Step 6: ssコマンドで待ち受けポートを確認する

サーバーを起動した状態で、別ターミナルから確認します。

```bash
ss -ltnp | grep 8080
```

環境によっては `-p` に権限が必要です。

```bash
sudo ss -ltnp | grep 8080
```

例:

```text
LISTEN 0 4096 *:8080 *:*
```

ここで見るポイントは、`8080` 番ポートが `LISTEN` 状態になっていることです。

```text
LISTEN = 接続待ち受け中
```

## Step 7: 1接続しか扱えない問題を確認する

今の実装は、1つのクライアントしか扱えません。

理由は、`Accept()` を1回しか呼んでいないからです。

```go
conn, err := listener.Accept()
```

つまり、1人目のクライアントを処理している間、2人目以降の接続をアプリケーション側で処理できません。

これを確認します。

1つ目のターミナルで接続します。

```bash
nc localhost 8080
```

接続したままにします。

別のターミナルで、もう一度接続します。

```bash
nc localhost 8080
```

期待通りに動かない、または応答が返らないことを確認します。

## Step 8: 複数クライアントに対応する

`Accept()` をループし、クライアントごとの処理を goroutine に分けます。

`cmd/server/main.go` を次のように修正します。

```go
package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
)

func main() {
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()

	log.Println("tcp echo server listening on :8080")

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("accept error:", err)
			continue
		}

		go handleConn(conn)
	}
}

func handleConn(conn net.Conn) {
	defer conn.Close()

	log.Println("client connected:", conn.RemoteAddr())
	defer log.Println("client disconnected:", conn.RemoteAddr())

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		text := scanner.Text()
		fmt.Fprintln(conn, text)
	}

	if err := scanner.Err(); err != nil {
		log.Println("read error:", err)
	}
}
```

再起動します。

```bash
go run ./cmd/server
```

複数ターミナルから接続します。

```bash
nc localhost 8080
```

別ターミナルでも接続します。

```bash
nc localhost 8080
```

どちらでもechoが返ればOKです。

## Step 9: ここで学ぶgoroutineの意味

```go
go handleConn(conn)
```

この行により、接続ごとの処理を別goroutineで実行します。

もし `go` を付けずに、

```go
handleConn(conn)
```

と書くと、1つの接続を処理している間、次の `Accept()` に戻れません。

つまり、複数クライアントを同時に扱いにくくなります。

```text
goなし:
Accept
↓
1クライアントを処理
↓
切断されるまで次のAcceptに戻れない

goあり:
Accept
↓
処理はgoroutineに任せる
↓
すぐ次のAcceptに戻る
```

## Step 10: TCPはメッセージ単位ではなくバイトストリームであることを確認する

今の実装では `bufio.Scanner` を使っています。

```go
scanner := bufio.NewScanner(conn)
```

`Scanner` はデフォルトで改行までを1つの単位として読みます。

そのため、`nc` で1行入力すると、その1行が返ってきます。

ただし、TCP自体が「1行ごとのメッセージ」を持っているわけではありません。  
TCPは単なるバイト列の流れです。

```text
TCP = バイトストリーム
HTTP = その上にリクエスト/レスポンスという形式を定義している
```

ここは重要です。

アプリケーション側が、

```text
どこからどこまでを1つのメッセージとみなすか
```

を決める必要があります。

今回のecho serverでは、改行をメッセージの区切りとして扱っています。

## Step 11: Go製TCPクライアントを作る

`nc` だけでなく、Goでクライアントも作ります。

```bash
mkdir -p cmd/client
touch cmd/client/main.go
```

`cmd/client/main.go`:

```go
package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
)

func main() {
	conn, err := net.Dial("tcp", "localhost:8080")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	fmt.Fprintln(conn, "hello from go client")

	scanner := bufio.NewScanner(conn)
	if scanner.Scan() {
		fmt.Println("response:", scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		log.Println("read error:", err)
	}

	os.Exit(0)
}
```

サーバーを起動した状態で実行します。

```bash
go run ./cmd/client
```

出力例:

```text
response: hello from go client
```

## Step 12: クライアントから複数行送る

次に、標準入力から読み取ってサーバーへ送るクライアントにします。

```go
package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
)

func main() {
	conn, err := net.Dial("tcp", "localhost:8080")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	stdin := bufio.NewScanner(os.Stdin)
	server := bufio.NewScanner(conn)

	for {
		fmt.Print("> ")

		if !stdin.Scan() {
			break
		}

		text := stdin.Text()
		if text == "exit" {
			break
		}

		fmt.Fprintln(conn, text)

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
```

実行します。

```bash
go run ./cmd/client
```

入力します。

```text
> hello
echo: hello
> tcp
echo: tcp
> exit
```

## Step 13: タイムアウトを設定する

TCP通信では、相手が応答しない可能性があります。

そのため、実務ではタイムアウトを考える必要があります。

まずクライアント側に接続タイムアウトを設定します。

```go
conn, err := net.DialTimeout("tcp", "localhost:8080", 3*time.Second)
```

必要なimport:

```go
import "time"
```

読み書きの期限も設定できます。

```go
conn.SetDeadline(time.Now().Add(10 * time.Second))
```

これは、読み取りと書き込み全体の期限です。

学ぶこと:

```text
ネットワーク通信では、相手が必ず応答するとは限らない。
そのため、タイムアウトを設けないと処理が止まり続ける可能性がある。
```

## Step 14: graceful shutdownを追加する

サーバーを `Ctrl+C` で止めたときに、終了シグナルを受け取るようにします。

簡易版として、`os/signal` を使います。

```go
package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()

	log.Println("tcp echo server listening on :8080")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		log.Println("shutting down server...")
		listener.Close()
	}()

	for {
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

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		text := scanner.Text()
		fmt.Fprintln(conn, text)
	}

	if err := scanner.Err(); err != nil {
		log.Println("read error:", err)
	}
}
```

`Ctrl+C` を押して終了します。

確認すること:

```text
- Ctrl+C はプロセスに SIGINT を送る
- signal.Notify でシグナルを受け取れる
- listener.Close() すると Accept() がエラーで戻る
- それをきっかけにサーバーを終了できる
```

## Step 15: テストを書く

TCPサーバーはテストしにくいですが、小さく分ければテストできます。

まず `internal/echo/server.go` に処理を移します。

```bash
mkdir -p internal/echo
touch internal/echo/server.go
```

`internal/echo/server.go`:

```go
package echo

import (
	"bufio"
	"fmt"
	"io"
)

func Handle(r io.Reader, w io.Writer) error {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		fmt.Fprintln(w, scanner.Text())
	}

	return scanner.Err()
}
```

テストを作ります。

```bash
touch internal/echo/server_test.go
```

`internal/echo/server_test.go`:

```go
package echo

import (
	"strings"
	"testing"
)

func TestHandle(t *testing.T) {
	input := strings.NewReader("hello\nworld\n")
	var output strings.Builder

	err := Handle(input, &output)
	if err != nil {
		t.Fatal(err)
	}

	want := "hello\nworld\n"
	got := output.String()

	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
```

実行します。

```bash
go test ./...
```

学ぶこと:

```text
ネットワーク処理そのものを直接テストするより、
io.Reader / io.Writer に分けるとテストしやすくなる。
```

これはGoらしい設計です。

## Step 16: 最終的な理解チェック

以下を自分の言葉で説明できれば、この課題は完了です。

### Q1. TCPサーバーは何をしているか

答えの例:

```text
TCPサーバーは、特定のポートで接続を待ち受ける。
クライアントが接続してくると Accept で接続を受け取り、
その接続からデータを読み取り、必要に応じて書き返す。
```

### Q2. `localhost:8080` の意味は何か

答えの例:

```text
localhost は自分自身のホストを表す名前。
8080 は接続先ポート番号。
つまり、自分のPCの8080番ポートに接続するという意味。
```

### Q3. `Accept()` は何をしているか

答えの例:

```text
Accept はクライアントからのTCP接続を受け付ける。
接続が来るまで待機し、接続が来たら net.Conn を返す。
```

### Q4. なぜ goroutine を使うのか

答えの例:

```text
1つの接続処理にブロックされず、次の接続を受け付けるため。
接続ごとの処理をgoroutineに任せることで、複数クライアントを同時に扱える。
```

### Q5. TCPはメッセージ単位のプロトコルか

答えの例:

```text
TCPはメッセージ単位ではなくバイトストリーム。
どこで1つのメッセージとみなすかは、アプリケーション側のプロトコルで決める。
今回のecho serverでは改行を区切りにしている。
```

### Q6. HTTPとの関係は何か

答えの例:

```text
HTTPは、TCPの上で動くことが多いアプリケーション層プロトコル。
TCPがバイト列を運び、その上でHTTPがリクエスト・レスポンスの形式を定義している。
```

## 発展課題

余裕があれば、以下に取り組みます。

### 1. 接続数をログに出す

現在接続中のクライアント数を表示します。

学べること:

```text
- 共有状態
- mutex
- goroutine間の競合
```

### 2. 簡易チャットサーバーにする

接続している全クライアントにメッセージを配信します。

学べること:

```text
- 複数接続管理
- channel
- broadcast
- 排他制御
```

### 3. 独自プロトコルを作る

たとえば、次のようなコマンドを扱います。

```text
PING
ECHO hello
TIME
QUIT
```

レスポンス例:

```text
PONG
hello
2026-06-13T12:00:00+09:00
BYE
```

学べること:

```text
- アプリケーションプロトコル
- コマンド解析
- HTTPのような上位プロトコルの考え方
```

### 4. Dockerで動かす

Dockerfileを作って、TCPサーバーをコンテナで動かします。

学べること:

```text
- ポート公開
- コンテナ内プロセス
- ホストとコンテナのネットワーク
```

例:

```bash
docker build -t tcp-echo .
docker run --rm -p 8080:8080 tcp-echo
```

## この課題で覚えるべきキーワード

```text
TCP
IPアドレス
ポート
localhost
listen
accept
connection
client
server
socket
net.Conn
goroutine
timeout
signal
バイトストリーム
```

## この課題のゴール

この課題のゴールは、TCPについて次のように説明できることです。

```text
TCPでは、サーバーがポートで接続を待ち受け、
クライアントがIPアドレスとポートを指定して接続する。
接続が確立されると、双方はバイト列を読み書きできる。
HTTPなどのアプリケーションプロトコルは、このTCP通信の上で
リクエストやレスポンスの形式を定義している。
```