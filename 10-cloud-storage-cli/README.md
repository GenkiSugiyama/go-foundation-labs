# 10-cloud-storage-cli

## 目的
AWS S3を本編として、クラウドストレージを「誰が、どのリソースに、何をしてよいか」というIAMの権限モデルから理解します。GCSは対応関係だけ補足します。

## この課題で学ぶこと
- IAMのprincipal、role、policy、service account
- bucketとobject
- 権限不足エラーの読み方
- 安全な認証、最小権限
- 課金と削除の注意

## 完成イメージ
```bash
go run ./cmd/storage-cli list --bucket "$S3_BUCKET"
go run ./cmd/storage-cli put --bucket "$S3_BUCKET" --key hello.txt --file ./hello.txt
go run ./cmd/storage-cli get --bucket "$S3_BUCKET" --key hello.txt --out ./downloaded.txt
```
まずはローカルで引数検証を行い、実クラウド操作は準備した環境だけで行います。

## ディレクトリ構成
```text
10-cloud-storage-cli/
├── README.md
├── go.mod
├── cmd/storage-cli/main.go
└── internal/storage/s3.go
```

## Step 1: ローカルで引数を検証する
```bash
go mod init example.com/go-foundation-labs/10-cloud-storage-cli
mkdir -p cmd/storage-cli internal/storage
```
```go
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("usage: storage-cli <list|put|get> [options]")
	}

	command := os.Args[1]
	flags := flag.NewFlagSet(command, flag.ExitOnError)
	bucket := flags.String("bucket", "", "S3 bucket")
	key := flags.String("key", "", "object key")
	file := flags.String("file", "", "upload source file")
	out := flags.String("out", "", "download destination")
	flags.Parse(os.Args[2:])

	if *bucket == "" {
		log.Fatal("bucket is required")
	}

	switch command {
	case "list":
		fmt.Printf("would list s3://%s\n", *bucket)
	case "put":
		if *key == "" || *file == "" {
			log.Fatal("put requires key and file")
		}
		fmt.Printf("would upload %s to s3://%s/%s\n", *file, *bucket, *key)
	case "get":
		if *key == "" || *out == "" {
			log.Fatal("get requires key and out")
		}
		fmt.Printf("would download s3://%s/%s to %s\n", *bucket, *key, *out)
	default:
		log.Fatalf("unknown command: %s", command)
	}
}
```
次のコマンドで、S3へ接続せずに入力を観察します。

```bash
go run ./cmd/storage-cli list --bucket demo
go run ./cmd/storage-cli put --bucket demo --key hello.txt --file ./hello.txt
go run ./cmd/storage-cli get --bucket demo --key hello.txt --out ./downloaded.txt
```

## Step 2: S3の権限モデルを読む
principalはリクエスト主体、roleは引き受ける権限、policyはaction/resource/conditionを記述する規則です。bucketはobjectの入れ物、objectはkeyと内容を持ちます。`s3:ListBucket`はbucket ARN、`s3:GetObject`はobject ARNが対象で、同じARNではありません。

## Step 3: AWS SDKで実装する
署名、リージョン、再試行を標準ライブラリだけで安全に実装するのは不適切なので、AWS公式SDK for Go v2を使います。
```bash
go get github.com/aws/aws-sdk-go-v2/config
go get github.com/aws/aws-sdk-go-v2/service/s3
```
```go
cfg, err := config.LoadDefaultConfig(context.Background())
if err != nil { log.Fatal(err) }
client := s3.NewFromConfig(cfg)
out, err := client.ListObjectsV2(context.Background(),
 &s3.ListObjectsV2Input{Bucket: aws.String(bucket)})
if err != nil { log.Fatal(err) }
for _, o := range out.Contents { fmt.Println(aws.ToString(o.Key)) }
```
SDKのDefault Credential Chain、環境変数、共有設定、IAM roleを使い、コードにアクセスキーを埋め込みません。

## Step 4: 実クラウドを使わず観察する
認証なしで引数検証とテストを行い、AccessDeniedを想定してprincipal、action、resource、condition、regionを読みます。
```bash
go test ./...
```

## Step 5: 実クラウドを使う場合
1. 対象アカウント、リージョン、bucket、課金アラートを確認する。
2. `aws sts get-caller-identity`で主体を確認する。
3. ListBucket、GetObject、必要時だけPutObjectの最小権限を付ける。
4. 小さなテキスト1個をput/getし、head-objectで確認する。
5. object、version、bucketを確認して削除する。
macOS/Linuxの環境変数指定は同じですが、PowerShellは `$env:AWS_PROFILE="read-only"` の形式です。履歴に秘密情報を入力しません。

## GCSとの対応関係
S3のbucket/objectはGCSのbucket/objectに対応します。S3 IAM roleに近い実行主体はGCPのservice account、policyはGCP IAM roleとbindingに対応します。ただし権限名、継承、署名、CLIは同一ではありません。

## ここで確認すること
- principal、action、resource、conditionを説明できること
- bucket ARNとobject ARNを区別すること
- AccessDeniedを管理者権限で雑に回避しないこと
- Default Credential Chainが秘密情報の直書きより安全な理由
- 保存、転送、API、versioningが課金に関係すること

## 詰まったこと
学習後に追記します。
- どの権限不足エラーで詰まりましたか。
- principal、resource、actionのどれを確認して解決しましたか。
- SDKや認証準備で詰まった点は何ですか。

## 分かったこと
学習後に追記します。
- bucketとobjectの権限はどう違いましたか。
- S3とGCSで同じ考え方、違う仕様は何ですか。

## 実務での関係
バックアップ、ログ、配布物、データ連携の設計に関係します。最小権限、短期認証、監査ログ、削除方針を合わせて考えます。

## 理解度チェック
1. principal、role、policyの関係は何ですか。
2. ListBucketとGetObjectのresourceが違う理由は何ですか。
3. なぜアクセスキーをソースコードに書いてはいけませんか。

## 発展課題
- put/get/deleteとaction別のpolicyを実装する
- prefix制限とconditionをテストする
- GCS版との権限対応表を作る

## 覚えるべきキーワード
S3、GCS、IAM、principal、role、policy、service account、bucket、object、ARN、AccessDenied、最小権限、Default Credential Chain。

## 安全注意
実クラウドは保存、転送、API、versioningで課金されます。個人情報や秘密情報をアップロードせず、短期認証、最小権限、課金アラートを使ってください。他人のbucketや認証情報を試してはいけません。

## この課題のゴール
実クラウドを不用意に操作せず、S3のbucket/objectとIAMを対応づけ、権限不足を安全に調査できるCLI設計を説明できることです。
