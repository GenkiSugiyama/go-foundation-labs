# 03-dns-lookup

## 目的

この課題では、Goの `net.Resolver` や `net.Lookup*` 系の関数を使ってDNSを調べるCLIツールを作ります。

ブラウザやAPIクライアントが、なぜドメイン名だけで通信できるのかを、コードと `dig` の両方で観察して理解することが目的です。

## この課題で学ぶこと

- DNSが名前解決の仕組みであること
- AレコードがIPv4アドレスを返すこと
- AAAAレコードがIPv6アドレスを返すこと
- CNAMEレコードが別名を表すこと
- MXレコードがメール配送先と優先度を表すこと
- Goでは `net.Resolver` や `net.LookupIP` などでDNS問い合わせができること
- `dig` を使うと、Goの結果だけでは見えにくい情報も観察できること
- DNSの結果は環境、利用するDNSサーバー、時刻、キャッシュ、ラウンドロビンで変わり得ること

## 完成イメージ

次のように実行します。

```bash
go run . example.com
```

例:

```text
name: example.com
A:
  23.192.228.80
  23.192.228.84
AAAA:
  2600:1408:ec00:36::1736:7f24
  2600:1408:ec00:36::1736:7f31
CNAME:
  example.com.
MX:
  <none>
```

環境によって表示内容は変わります。ここが重要です。DNSの結果は固定ではありません。

## ディレクトリ構成

```text
03-dns-lookup/
├── README.md
├── go.mod
└── main.go
```

この課題では、まず `main.go` ひとつで作って構いません。

## Step 1: まずDNSを読む

先に用語だけ整理します。

- A: ドメイン名に対応するIPv4アドレス
- AAAA: ドメイン名に対応するIPv6アドレス
- CNAME: ある名前が別の名前の別名であること
- MX: メールを受け取るサーバー名と優先度

たとえば、アプリケーションが `api.example.com` に接続するとき、最初に必要なのは「その名前がどのIPアドレスに対応するか」です。

つまりDNSは、通信そのものではなく、通信先を見つけるための仕組みです。

## Step 2: 最小のDNS CLIを作る

最初はA/AAAAを表示するところから始めます。

```bash
mkdir -p 03-dns-lookup
cd 03-dns-lookup
go mod init example.com/go-foundation-labs/03-dns-lookup
touch main.go
```

`main.go` に次のコードを書きます。

```go
package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
)

func main() {
	if len(os.Args) != 2 {
		log.Fatalf("usage: %s <host>", os.Args[0])
	}

	host := os.Args[1]
	resolver := net.Resolver{}

	ips, err := resolver.LookupIP(context.Background(), "ip", host)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("name:", host)
	fmt.Println("A:")
	for _, ip := range ips {
		if ip.To4() != nil {
			fmt.Println(" ", ip.String())
		}
	}

	fmt.Println("AAAA:")
	for _, ip := range ips {
		if ip.To4() == nil {
			fmt.Println(" ", ip.String())
		}
	}
}
```

このコードは、`LookupIP` の結果を見て、IPv4ならA、IPv6ならAAAAとして分けて表示しています。

## Step 3: コマンドで観察する

実行します。

```bash
go run . example.com
go run . google.com
go run . localhost
go run . does-not-exist.invalid
```

確認したい点は次の通りです。

- `example.com` や `google.com` では複数のIPアドレスが返ることがある
- `localhost` はローカル環境の設定で解決されることがある
- `does-not-exist.invalid` では失敗する

`invalid` は予約済みのTLDなので、存在しない名前の確認に使いやすいです。

## Step 4: CNAMEとMXも追加する

次に、A/AAAAだけでなく、CNAMEとMXも確認できるようにします。

`main.go` を次のように拡張します。

```go
package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
)

func main() {
	if len(os.Args) != 2 {
		log.Fatalf("usage: %s <host>", os.Args[0])
	}

	host := os.Args[1]
	ctx := context.Background()
	resolver := net.Resolver{}

	ips, err := resolver.LookupIP(ctx, "ip", host)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("name:", host)

	fmt.Println("A:")
	for _, ip := range ips {
		if ip.To4() != nil {
			fmt.Println(" ", ip.String())
		}
	}

	fmt.Println("AAAA:")
	for _, ip := range ips {
		if ip.To4() == nil {
			fmt.Println(" ", ip.String())
		}
	}

	cname, err := resolver.LookupCNAME(ctx, host)
	if err != nil {
		fmt.Println("CNAME:")
		fmt.Println("  <lookup error>", err)
	} else {
		fmt.Println("CNAME:")
		fmt.Println(" ", cname)
	}

	fmt.Println("MX:")
	mxs, err := resolver.LookupMX(ctx, host)
	if err != nil {
		fmt.Println("  <lookup error>", err)
		return
	}
	if len(mxs) == 0 {
		fmt.Println("  <none>")
		return
	}

	for _, mx := range mxs {
		fmt.Printf("  %s priority=%d\n", mx.Host, mx.Pref)
	}
}
```

ここでは、A/AAAAに加えて、`LookupCNAME` と `LookupMX` を使っています。

ただし、`LookupCNAME` はCNAMEレコードの有無をそのまま返すAPIではなく、指定名の正規名を返します。CNAMEレコードがない場合でも名前が返ることがあるため、実際のCNAMEレコードは次のStepで `dig CNAME` と比較して判断してください。

## Step 5: `dig` と比較する

Goの結果だけで終わらせず、必ず `dig` と比較します。

```bash
dig A example.com
dig AAAA example.com
dig CNAME www.example.com
dig MX gmail.com
```

たとえば `gmail.com` にはMXがあり、`example.com` には環境や時点によって見え方が異なることがあります。

`dig` の出力では次の点を見ます。

- `ANSWER SECTION` に何が返っているか
- TTLが何秒か
- 問い合わせ先DNSサーバーがどこか
- AとAAAAが両方あるか
- CNAMEの先にさらに名前解決が必要か

macOSでは `dig` が最初から入っていることが多いですが、環境によっては入っていません。  
Linuxでは `dnsutils` や `bind-utils` の導入が必要なことがあります。

## Step 6: 結果を説明する

ここで重要なのは、単に値を眺めることではありません。

次を自分の言葉で説明できるか確認します。

- AとAAAAは「名前からIPアドレスを得る」ための情報である
- CNAMEは「別名」なので、最終IPアドレスそのものとは限らない
- MXはWebアクセスではなく、メール配送のための情報である
- DNS回答は毎回同じとは限らない

特に最後は重要です。

同じ `go run . example.com` でも、次の理由で結果が変わることがあります。

- 利用しているDNSサーバーが違う
- ローカルやOSのキャッシュが効いている
- CDNやロードバランサで複数IPが返る
- TTL切れ後に新しい値へ切り替わる
- IPv4/IPv6の優先順位が環境で違う

つまり、DNSの結果を固定値だと思う理解は間違いです。

## Step 7: 少し改造する

最後に少しだけ改造します。

たとえば `www.example.com` のような名前で試し、その時点で実際にCNAMEが返るかを確認します。DNS設定は変更され得るため、特定の名前にCNAMEが必ず存在するとは決めつけません。

```bash
go run . www.example.com
dig CNAME www.example.com
dig A www.example.com
dig AAAA www.example.com
```

比較すると、「別名を返す段階」と「最終的なIPアドレスを返す段階」が分かれていることが見えやすくなります。

余裕があれば、`net.LookupHost` も試してください。

```go
hosts, err := net.LookupHost(host)
if err != nil {
	log.Fatal(err)
}
fmt.Println("LookupHost:")
for _, h := range hosts {
	fmt.Println(" ", h)
}
```

`LookupHost` は便利ですが、AとAAAAを分けずに扱うため、観察目的では `LookupIP` のほうが分かりやすいです。

## ここで確認すること

- DNSはドメイン名をIPアドレスへ変換するための仕組みであること
- AはIPv4、AAAAはIPv6であること
- CNAMEは別名、MXはメール配送先であること
- `net.Resolver` と `net.Lookup*` 系の関数で問い合わせできること
- `dig` はGoの結果と比較する観察ツールとして使えること
- DNS結果は環境や時刻で変わるため、1回の結果を絶対視してはいけないこと

## 詰まったこと

学習後に記入します。

- どのドメインで、A/AAAA/CNAME/MXのどれが期待と違いましたか。
- Goの結果と `dig` の結果が違ったとき、DNSサーバー・TTL・IPv4/IPv6のどれを確認しましたか。
- 「同じ名前なのに結果が変わる」ことを、どう説明できるようになりましたか。

## 分かったこと

学習後に記入します。

- A、AAAA、CNAME、MXの役割をそれぞれ自分の言葉で説明してください。
- `LookupIP` と `LookupHost` の違いをどう理解しましたか。
- DNS結果が環境や時刻で変わる理由を、キャッシュや複数アドレスの観点から説明してください。

## 実務での関係

実務では、API通信、DB接続、メール配送、CDN、ロードバランサ、障害調査のどれもDNSと関係します。

たとえば、ある環境でだけ接続できないとき、原因はアプリではなくDNSの向き先の違いかもしれません。  
そのため、アプリコードだけでなく `dig` や `nslookup` で外側から確認する習慣が重要です。

## 理解度チェック

1. AレコードとAAAAレコードの違いは何ですか。
2. CNAMEが返ったあと、最終的なIPアドレスを得るには何が必要ですか。
3. MXレコードはどの用途で使われますか。
4. Goの結果と `dig` の結果が違っても、すぐにどちらかが壊れていると決めつけてはいけないのはなぜですか。
5. DNS結果が時刻で変わるのは、どんな仕組みと関係していますか。

## 発展課題

- `-type` フラグを付けて、表示対象をA、AAAA、CNAME、MXから選べるようにする
- 問い合わせ時間を計測して表示する
- `LookupAddr` を使って逆引きを試す
- `context.WithTimeout` を使ってタイムアウトを入れる

## 覚えるべきキーワード

`DNS`、`名前解決`、`resolver`、`A`、`AAAA`、`CNAME`、`MX`、`TTL`、`キャッシュ`、`dig`

## この課題のゴール

GoでDNS問い合わせを行い、A、AAAA、CNAME、MXの役割を説明できること。  
さらに、`dig` と比較しながら、DNS結果が環境や時刻で変わり得る理由まで説明できれば、この課題のゴールです。
