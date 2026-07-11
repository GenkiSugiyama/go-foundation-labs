# 04-procinfo

## 目的

この課題では、Goの `os` パッケージを中心に、小さなCLIツールを作りながら、プログラムがOS上でプロセスとしてどう動くかを学びます。

普段は見えにくいPID、親プロセス、環境変数、標準入出力、終了コード、シグナルを、コードとコマンドで観察して理解することが目的です。

## この課題で学ぶこと

- `os` パッケージでプロセス情報を扱う方法
- PIDとPPIDの意味
- 環境変数がどのように渡されるか
- stdin、stdout、stderrが別々の経路であること
- 終了コードで成功・失敗を伝えること
- シグナルでプロセスに通知できること
- macOSとLinuxで見える情報やコマンドの差があること

## 完成イメージ

次のように実行します。

```bash
go run . hello
```

例:

```text
pid=12345
ppid=12340
args=[/tmp/go-build.../exe/procinfo hello]
APP_MODE is not set
type one line and press Enter:
hello from stdin
stdout: hello from stdin
```

失敗時には標準エラー出力へメッセージを出し、終了コードも変わるようにします。

## ディレクトリ構成

```text
04-procinfo/
├── README.md
├── go.mod
└── main.go
```

この課題では、最初は `main.go` ひとつで作って構いません。

## Step 1: まずプロセスを読む

先に概念を整理します。

- PID: そのプロセス自身のID
- PPID: そのプロセスを起動した親プロセスのID
- 環境変数: 起動時に親から引き継がれる設定値
- stdin: 標準入力
- stdout: 標準出力
- stderr: 標準エラー出力
- 終了コード: コマンドの成功・失敗を親に返す値
- シグナル: 外部からプロセスへ送る通知

CLIツールやコンテナ、systemd、CI、シェルスクリプトは、これらを前提に動いています。

## Step 2: 最小のprocinfoを作る

まずはPID、PPID、引数、環境変数を表示します。

```bash
mkdir -p 04-procinfo
cd 04-procinfo
go mod init example.com/go-foundation-labs/04-procinfo
touch main.go
```

`main.go` に次のコードを書きます。

```go
package main

import (
	"bufio"
	"fmt"
	"os"
)

func main() {
	fmt.Printf("pid=%d\n", os.Getpid())
	fmt.Printf("ppid=%d\n", os.Getppid())
	fmt.Printf("args=%v\n", os.Args)

	if value, ok := os.LookupEnv("APP_MODE"); ok {
		fmt.Println("APP_MODE=", value)
	} else {
		fmt.Println("APP_MODE is not set")
	}

	fmt.Println("type one line and press Enter:")

	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		fmt.Fprintln(os.Stderr, "no input")
		os.Exit(2)
	}

	fmt.Fprintln(os.Stdout, "stdout:", scanner.Text())

	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "read error:", err)
		os.Exit(1)
	}
}
```

この時点で、`os` パッケージから次を使っています。

- `os.Getpid`
- `os.Getppid`
- `os.Args`
- `os.LookupEnv`
- `os.Stdin`
- `os.Stdout`
- `os.Stderr`
- `os.Exit`

## Step 3: コマンドで観察する

まずは普通に実行します。

```bash
go run . hello
```

次に、ビルドしたバイナリでも試します。

```bash
go build -o procinfo .
./procinfo hello
```

ここで確認したいのは、`go run` と `./procinfo` で見え方が少し違うことです。

特に `go run` は、一度ビルドしてから別プロセスとして実行するため、親子関係を厳密に観察したいときには少し分かりにくくなります。  
PIDやPPIDを観察するときは、ビルド済みバイナリのほうが理解しやすいです。

## Step 4: 環境変数を観察する

環境変数を付けて起動します。

```bash
APP_MODE=study ./procinfo hello
```

環境によっては、次のように見えます。

```text
APP_MODE= study
```

`APP_MODE` はコードの中で定義したものではありません。  
親のシェルが、子プロセスである `./procinfo` に引き渡しています。

## Step 5: stdin / stdout / stderr を分けて観察する

標準入力と標準出力、標準エラー出力を分けて確認します。

```bash
printf 'hello from stdin\n' | ./procinfo hello >out.txt 2>err.txt
cat out.txt
cat err.txt
```

ここで見たいことは次の通りです。

- `printf` の出力が `stdin` に入る
- 通常の結果は `stdout` に出る
- エラーは `stderr` に出る

入力なしのケースも試します。

```bash
./procinfo hello < /dev/null
printf 'exit=%s\n' "$?"
```

`/dev/null` を標準入力につなぐと、`scanner.Scan()` が失敗し、終了コード `2` で終わることが確認できます。

macOS/Linuxのシェルでは、`>` はstdout、`2>` はstderrのリダイレクトです。

## Step 6: 終了コードを観察する

失敗時と成功時で終了コードが違うことを確認します。

```bash
printf 'ok\n' | ./procinfo hello
printf 'exit=%s\n' "$?"

./procinfo hello < /dev/null
printf 'exit=%s\n' "$?"
```

シェルの `$?` は、直前のプロセスの終了コードです。

実務ではこの値を使って、シェルスクリプトやCIが次の処理を続けるか、止めるかを判断します。

## Step 7: シグナルを観察する

次に、シグナルを受け取って終了するように少し改造します。

`main.go` を次のように変更します。

```go
package main

import (
	"bufio"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	fmt.Printf("pid=%d\n", os.Getpid())
	fmt.Printf("ppid=%d\n", os.Getppid())
	fmt.Printf("args=%v\n", os.Args)

	if value, ok := os.LookupEnv("APP_MODE"); ok {
		fmt.Println("APP_MODE=", value)
	} else {
		fmt.Println("APP_MODE is not set")
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	fmt.Println("type one line and press Enter:")

	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		fmt.Fprintln(os.Stderr, "no input")
		os.Exit(2)
	}

	fmt.Fprintln(os.Stdout, "stdout:", scanner.Text())
	fmt.Println("waiting for Ctrl+C or SIGTERM")

	sig := <-sigCh
	fmt.Println("received signal:", sig)
}
```

ビルドして実行します。

```bash
go build -o procinfo .
printf 'hello\n' | ./procinfo hello
```

別ターミナルからシグナルを送ります。

```bash
ps -p <PID> -o pid,ppid,command
kill -TERM <PID>
```

`Ctrl+C` でも試せます。

```bash
./procinfo hello
```

入力後に `Ctrl+C` を押すと、`os.Interrupt` を受け取れます。

## Step 8: macOS / Linux の差を確認する

この課題では、macOSとLinuxで完全に同じ表示になるとは限りません。

たとえば次の差があります。

- `ps` の列名や表示形式が少し違う
- 利用できる環境変数の初期値が違う
- `/proc/<PID>` はLinuxにはあるが、macOSには基本的にない
- シグナル名は共通でも、周辺コマンドの出力が違うことがある

確認コマンド:

```bash
ps -p <PID> -o pid,ppid,command
kill -l
env | grep APP_MODE
```

Linuxでは追加で `/proc` も観察できます。

```bash
ls /proc/$$
cat /proc/$$/status
```

ただし、macOSには同じ形の `/proc` はありません。  
この差を知らずに「どのOSでも同じはずだ」と考えるのは間違いです。

## ここで確認すること

- `os` パッケージでPID、PPID、環境変数、標準入出力を扱えること
- PIDは自分自身、PPIDは親プロセスを表すこと
- 環境変数は親プロセスから引き継がれること
- stdin、stdout、stderrは別の経路であり、リダイレクトも別々にできること
- 終了コードで成功・失敗を親へ伝えること
- SIGTERMや `Ctrl+C` はプロセスへの通知であること
- macOSとLinuxでは観察方法や見える情報に差があること

## 詰まったこと

学習後に記入します。

- `go run` とビルド済みバイナリで、PIDやPPIDの見え方はどう違いましたか。
- stdin、stdout、stderrを分けたとき、どこで混乱しましたか。
- macOSとLinuxの差で、どのコマンドや出力に戸惑いましたか。

## 分かったこと

学習後に記入します。

- PIDとPPIDの関係を、自分の言葉で説明してください。
- 環境変数が親から子へ渡るとは、どういう意味ですか。
- 終了コードとシグナルが、運用やシェルスクリプトでどう使われるか説明してください。

## 実務での関係

実務では、CLIツール、バッチ、コンテナ、CI、systemd、Kubernetesなど、ほぼすべてがプロセスとして動いています。

標準出力にログを出す設計、異常時に非0の終了コードを返す設計、SIGTERMを受けたときに後片付けして止まる設計は、どれも日常的に必要です。

特にコンテナ運用では、PID 1 の振る舞い、終了コード、シグナル処理を理解していないと、停止や再起動の挙動を誤解しやすくなります。

## 理解度チェック

1. PIDとPPIDはそれぞれ何を表しますか。
2. 環境変数はどこからプロセスへ渡されますか。
3. stdoutとstderrを分ける利点は何ですか。
4. 終了コード `0` と非 `0` は、一般にどう使い分けられますか。
5. SIGTERMとSIGKILLの違いは何ですか。
6. `os.Exit` のあとに `defer` が実行されないことは、どんな場面で問題になりますか。

## 発展課題

- `os/exec` を使って子プロセスを起動し、その終了コードを読む
- 子プロセスのstdoutとstderrを親で受け取る
- Linuxでは `/proc/<PID>` を読み、macOSでは `ps` や `lsof` で代替観察する
- シグナル受信後にタイムアウト付きで安全に終了する処理を追加する

## 覚えるべきキーワード

`os`、`PID`、`PPID`、`環境変数`、`stdin`、`stdout`、`stderr`、`終了コード`、`シグナル`、`SIGINT`、`SIGTERM`、`SIGKILL`

## この課題のゴール

Goの `os` パッケージを使って、PID、PPID、環境変数、標準入出力、終了コード、シグナルを観察できること。  
さらに、macOSとLinuxの差も踏まえて、プロセスがOS上でどう扱われるかを説明できれば、この課題のゴールです。
