# go-foundation-labs

## 概要

`go-foundation-labs` は、ソフトウェアエンジニアとして必要になる基礎知識を、Goで小さなツールやライブラリを作りながら学ぶための学習用リポジトリです。

単にドキュメントを読むだけではなく、実際にコードを書き、コマンドで観察し、動作を自分の言葉で説明できるようになることを目的とします。

## 目的

このリポジトリの目的は、次のような基礎知識を実装を通して理解することです。

* ネットワーク
* HTTP
* DNS
* Linux / プロセス
* データベース
* Webセキュリティ
* コンテナ
* クラウド / IAM
* 設計・デザインパターン
* データ構造とアルゴリズム

最終的には、Webアプリケーションがどのような仕組みの上で動いているのかを、下の層から順に説明できる状態を目指します。

## 学習方針

各セクションでは、以下の流れで学習します。

```text
読む
↓
小さく作る
↓
コマンドで観察する
↓
なぜそう動くか説明する
↓
少し改造する
```

重要なのは、コードを写すことではなく、次の3点を自分の言葉で説明できるようにすることです。

```text
1. このツールは何を確認するためのものか
2. 内部でどの仕組みを使っているか
3. 実務でどんな場面に関係するか
```

## 前提

主に以下の環境を想定します。

* Go
* Linux / macOS / WSL2
* Docker
* PostgreSQL
* curl
* dig
* nc
* ss
* psql

Goのバージョンは、特別な理由がなければ安定版を使用します。

```bash
go version
```

Dockerを使う課題では、事前にDockerが起動していることを確認します。

```bash
docker version
```

## リポジトリ構成

```text
go-foundation-labs/
├── README.md
├── 01-tcp-echo/
├── 02-http-mini-api/
├── 03-dns-lookup/
├── 04-procinfo/
├── 05-todo-api/
├── 06-security-board/
├── 07-dockerized-api/
├── 08-notifier/
├── 09-collections/
└── 10-cloud-storage-cli/
```

各ディレクトリは独立した小さな学習課題です。

## 学習セクション

### 01-tcp-echo

TCP echo server / clientを作ります。

学ぶこと:

* TCP接続
* IPアドレス
* ポート
* listen
* accept
* client / server
* net.Conn
* goroutine
* バイトストリーム

目的:

HTTPを学ぶ前に、HTTPの下で使われることが多いTCP通信の基本を理解します。

### 02-http-mini-api

Goの `net/http` を使って、小さなHTTP APIを作ります。

学ぶこと:

* HTTPメソッド
* URLパス
* リクエストヘッダー
* レスポンスヘッダー
* ステータスコード
* JSON
* curlによる確認

作るものの例:

```text
GET    /health
GET    /users
POST   /users
DELETE /users/{id}
```

目的:

HTTPリクエストとレスポンスの構造を、実装と `curl` で確認します。

### 03-dns-lookup

ドメイン名からIPアドレスを取得するCLIツールを作ります。

学ぶこと:

* DNS
* Aレコード
* AAAAレコード
* CNAME
* MXレコード
* 名前解決

目的:

DNSが「ドメイン名からIPアドレスを取得する仕組み」であることを、Goのコードと `dig` で確認します。

### 04-procinfo

プロセス情報を表示するCLIツールを作ります。

学ぶこと:

* プロセス
* PID
* 親プロセス
* 環境変数
* 標準入力
* 標準出力
* 終了コード
* シグナル

目的:

アプリケーションがOS上でどのようにプロセスとして実行されるかを確認します。

### 05-todo-api

PostgreSQLを使った小さなTODO APIを作ります。

学ぶこと:

* SQL
* INSERT / SELECT / UPDATE / DELETE
* トランザクション
* COMMIT
* ROLLBACK
* インデックス
* 複合インデックス
* EXPLAIN

目的:

DBを使うWeb APIを作り、SQL・トランザクション・インデックスの基礎を実装で確認します。

### 06-security-board

あえて脆弱な掲示板を作り、その後に修正します。

学ぶこと:

* SQLインジェクション
* XSS
* CSRF
* Cookie
* SameSite
* HttpOnly
* Secure
* 認証
* 認可

目的:

攻撃手法を知識として覚えるだけでなく、危険なコードを見つけて修正できるようにします。

### 07-dockerized-api

Go APIとPostgreSQLをDocker Composeで動かします。

学ぶこと:

* Dockerfile
* イメージ
* コンテナ
* レイヤー
* bind mount
* volume
* コンテナネットワーク
* 環境変数

目的:

コンテナが何を隔離し、何をホストと共有しているのかを確認します。

### 08-notifier

通知ライブラリを作ります。

学ぶこと:

* interface
* ポリモーフィズム
* Strategyパターン
* Observerパターン
* Composition over Inheritance
* LSP
* Decorator的な構成

作るものの例:

```text
EmailNotifier
SlackNotifier
ConsoleNotifier
LoggingNotifier
RetryNotifier
```

目的:

設計原則やデザインパターンを、Goのinterfaceとstructを使って理解します。

### 09-collections

小さなデータ構造ライブラリを作ります。

学ぶこと:

* Stack
* Queue
* Set
* LinkedList
* BinarySearch
* Trie
* 計算量
* 空間量

目的:

データ構造の特徴と計算量を、実装しながら確認します。

### 10-cloud-storage-cli

AWS S3またはGoogle Cloud Storageを操作するCLIツールを作ります。

学ぶこと:

* IAM
* principal
* role
* policy
* service account
* bucket
* object storage
* 権限不足時のエラー

目的:

クラウドIAMが「誰が、どのリソースに、何をしてよいか」を管理する仕組みであることを確認します。

## 推奨する進め方

最初は以下の順番で進めます。

```text
1. 01-tcp-echo
2. 02-http-mini-api
3. 03-dns-lookup
4. 04-procinfo
5. 05-todo-api
6. 07-dockerized-api
7. 06-security-board
8. 08-notifier
9. 09-collections
10. 10-cloud-storage-cli
```

厳密に番号順でなくても構いませんが、最初は `01-tcp-echo` と `02-http-mini-api` を先に進めることを推奨します。

理由は、Webアプリケーションの理解に必要な通信の土台になるためです。

## 各セクションのREADMEに書くこと

各ディレクトリには `README.md` を置き、以下を記録します。

```text
- この課題の目的
- 作るもの
- 学ぶこと
- 実行方法
- 確認コマンド
- 実装手順
- 詰まったこと
- 分かったこと
- 実務での関係
- 理解チェック
```

コードを書くだけでなく、学んだことを文章にすることで理解を定着させます。

## 実行例

各セクションに移動して実行します。

```bash
cd 01-tcp-echo
go run ./cmd/server
```

別ターミナルで確認します。

```bash
nc localhost 8080
```

Goのテストがある場合は、以下で実行します。

```bash
go test ./...
```

## 学習時の注意

このリポジトリでは、最初からきれいな設計を目指しすぎません。

まずは動くものを作ります。

```text
動くものを作る
↓
なぜ動くか確認する
↓
問題点を見つける
↓
少しずつ改善する
```

最初から抽象化しすぎると、何を学ぶためのコードなのかが分かりにくくなります。

特に、以下には注意します。

* パターン名を暗記するだけで終わらせない
* コードを写すだけで終わらせない
* 動いた理由を説明しないまま次へ進まない
* エラーを無視しない
* コマンドで観察する工程を省略しない

## 目標

このリポジトリを通して、最終的に次の流れを説明できるようになることを目指します。

```text
ユーザーがブラウザでURLを開く
↓
DNSでドメイン名からIPアドレスを取得する
↓
TCP接続を確立する
↓
TLSで通信を暗号化する
↓
HTTPリクエストを送る
↓
Webアプリケーションがリクエストを処理する
↓
認証・認可を確認する
↓
DBへSQLを発行する
↓
トランザクションやインデックスが関係する
↓
レスポンスを返す
↓
アプリケーションはコンテナ上で動く
↓
クラウドIAMで実行権限やリソース操作権限を管理する
```

この流れを自分の言葉で説明できるようになることが、このリポジトリのゴールです。
