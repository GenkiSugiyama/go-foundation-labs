# 07-dockerized-api

## 目的
Go APIとPostgreSQLを別コンテナで動かし、Dockerがアプリ、データ、ネットワークをどう分けるかを観察します。

## この課題で学ぶこと
- Dockerfile、イメージ、コンテナ、レイヤー
- bind mountとvolume、Composeネットワーク
- 環境変数と、コンテナ内の`localhost`の意味
- `docker ps`、`inspect`、`logs`、`exec`による観察

## 完成イメージ
```bash
docker compose up --build -d
curl -i http://localhost:8080/health
curl -i http://localhost:8080/todos
```

## ディレクトリ構成
```text
07-dockerized-api/
├── README.md
├── Dockerfile
├── compose.yaml
├── go.mod
├── cmd/server/main.go
└── internal/api/handler.go
```

## Step 1: ホストで最小APIを作る
```bash
mkdir -p cmd/server internal/api
go mod init example.com/go-foundation-labs/07-dockerized-api
```

```go
package main

import (
	"encoding/json"
	"log"
	"net/http"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	log.Println("dockerized api listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
```

起動します。

```bash
go run ./cmd/server
```

別ターミナルで `curl -i http://localhost:8080/health` を実行します。まずコンテナなしで確認します。

## Step 2: DB設定を環境変数にする
DB接続先を `DB_HOST`、`DB_PORT`、`DB_USER`、`DB_PASSWORD`、`DB_NAME` から組み立てます。Composeでは `DB_HOST=postgres` とします。`postgres` はサービス名です。APIコンテナ内の `localhost` はAPIコンテナ自身なのでDBを指しません。

PostgreSQLドライバは標準ライブラリに含まれないため、この課題ではpgxの `database/sql` 互換ドライバを追加します。

```bash
go get github.com/jackc/pgx/v5/stdlib
```

この操作により `go.mod` と `go.sum` が更新されます。依存を追加した理由と取得したバージョンを確認してください。

## Step 3: Dockerfileを作る
```dockerfile
FROM golang:1.25 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/server ./cmd/server
FROM gcr.io/distroless/static-debian12
COPY --from=build /out/server /server
EXPOSE 8080
ENTRYPOINT ["/server"]
```
```bash
docker build -t dockerized-api:dev .
docker image history dockerized-api:dev
docker image inspect dockerized-api:dev
```
FROM、COPY、RUNはキャッシュ可能なレイヤーです。ソースだけ変えたとき、どの層が再利用されるか予想します。

## Step 4: ComposeでAPIとPostgreSQLを接続する
```yaml
services:
  api:
    build: .
    ports: ["8080:8080"]
    environment:
      DB_HOST: postgres
      DB_PORT: "5432"
      DB_USER: app
      DB_PASSWORD: local-only
      DB_NAME: app
    depends_on: [postgres]
  postgres:
    image: postgres:16
    environment:
      POSTGRES_USER: app
      POSTGRES_PASSWORD: local-only
      POSTGRES_DB: app
    volumes:
      - postgres-data:/var/lib/postgresql/data
volumes:
  postgres-data:
```
```bash
docker compose config
docker compose up --build -d
docker compose ps
docker compose logs -f api
```
Composeはサービス名を名前解決できるネットワークを作ります。ホストからAPIはlocalhost:8080、APIからDBはpostgres:5432です。named volumeはDBデータを保持します。

`depends_on` はコンテナの起動順を制御しますが、PostgreSQLが接続可能になるまで待つ機能ではありません。API側で接続を再試行するか、発展課題でhealthcheckを追加して違いを確認します。

## Step 5: mountとネットワークを観察する
```bash
docker network ls
docker network inspect <compose-network>
docker volume ls
docker volume inspect <volume-name>
docker inspect <container-id>
```
このDockerfileの実行用イメージにはシェルや `getent` が含まれないため、名前解決とIPアドレスは `docker network inspect` で観察します。bind mountはソース共有向け、volumeはDBデータ向けです。

## ここで確認すること
- イメージ、コンテナ、レイヤーの関係
- コンテナ内のlocalhost:5432がDBではないこと
- logsから接続エラーを読めること
- portsと内部通信の違い

## 詰まったこと
学習後に追記します。
- どのコマンド、エラー、設定で詰まりましたか。
- localhost、サービス名、volumeのどれを誤解しましたか。
- どう観察して切り分けましたか。

## 分かったこと
学習後に追記します。
- ホスト、API、DBのどこで何が動いていましたか。
- コンテナを削除しても残るもの、消えるものは何ですか。

## 実務での関係
開発環境の再現、CIの依存サービス、設定分離、ログ調査、DBバックアップ設計に関係します。

## 理解度チェック
1. DB_HOST=postgresが機能する理由は何ですか。
2. bind mountとvolumeをどちらに使いますか。
3. イメージとコンテナはどう違いますか。

## 発展課題
- healthcheckと/readyを追加する
- 開発用bind mountを作る
- multi-stage build前後のサイズを比較する

## 覚えるべきキーワード
Dockerfile、image、container、layer、bind mount、volume、Compose、network、service name、environment variable、localhost。

## 安全注意
パスワードはローカル専用のダミー値にし、秘密情報をDockerfileやGitへ入れないでください。volume削除やクラウド接続はデータ消失・課金を確認してから行います。

## この課題のゴール
API、DB、ホストの境界を図にし、接続・設定・永続化の問題を観察コマンドで切り分けて説明できることです。
