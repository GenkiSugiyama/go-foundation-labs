# 02-http-mini-api

## 目的

この課題では、Go の `net/http` を使って小さなHTTP APIを作りながら、HTTPリクエストとレスポンスの基本を学びます。

TCPの上でHTTPがどのような形式を定義しているかを、実装と `curl` による確認で理解することが目的です。

## この課題で学ぶこと

- HTTPメソッドで操作の種類を表すこと
- URLパスで対象のリソースを表すこと
- リクエストヘッダーとレスポンスヘッダーがあること
- ステータスコードで結果を表すこと
- JSONでデータをやり取りすること
- Goでは `net/http` でHTTPサーバーを書けること
- `curl` を使うとHTTPの中身を観察しやすいこと

## 完成イメージ

サーバーを起動します。

```bash
go run ./cmd/server
```

`/health` を確認します。

```bash
curl -i http://localhost:8080/health
```

例:

```text
HTTP/1.1 200 OK
Content-Type: application/json

{"status":"ok"}
```

`/users` にユーザーを登録します。

```bash
curl -i -X POST http://localhost:8080/users \
  -H "Content-Type: application/json" \
  -d '{"name":"Alice"}'
```

一覧を確認します。

```bash
curl -i http://localhost:8080/users
```

つまり、小さなHTTP APIを作り、HTTPの構造を観察できるようにします。

## ディレクトリ構成

```text
02-http-mini-api/
├── README.md
├── go.mod
├── cmd/
│   └── server/
│       └── main.go
└── internal/
    └── httpapi/
        └── handler.go
```

最初は `cmd/server/main.go` にまとめて書いても構いません。  
慣れてきたら `internal/httpapi/handler.go` に処理を分けます。

## Step 1: Goモジュールを作成する

```bash
mkdir -p go-foundation-labs/02-http-mini-api
cd go-foundation-labs/02-http-mini-api

go mod init example.com/go-foundation-labs/02-http-mini-api
```

確認します。

```bash
ls
```

```text
go.mod
```

## Step 2: 最小のHTTPサーバーを作る

まずは `cmd/server/main.go` を作成します。

```bash
mkdir -p cmd/server
touch cmd/server/main.go
```

実装します。

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
		json.NewEncoder(w).Encode(map[string]string{
			"status": "ok",
		})
	})

	log.Println("http mini api listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
```

## Step 3: サーバーを起動する

```bash
go run ./cmd/server
```

次のように表示されればOKです。

```text
http mini api listening on :8080
```

この状態で、サーバーは `8080` 番ポートでHTTPリクエストを待ち受けています。

## Step 4: curlで `/health` を確認する

別ターミナルから次を実行します。

```bash
curl -i http://localhost:8080/health
```

例:

```text
HTTP/1.1 200 OK
Content-Type: application/json
Date: ...
Content-Length: 16

{"status":"ok"}
```

ここで `-i` を付けているのは、レスポンスボディだけでなくヘッダーも見たいからです。

## Step 5: ここで確認すること

### HTTPにはメソッドがある

今回の `curl` はデフォルトで `GET` を送っています。

```bash
curl -i http://localhost:8080/health
```

これは次の意味です。

```text
GET /health HTTP/1.1
```

### URLパスで対象を表す

```text
/health
```

は「アプリケーションのヘルスチェック用のリソース」を表すパスです。

### ステータスコードで結果を表す

```text
200 OK
```

は「リクエストは正常に処理された」という意味です。

### ヘッダーで追加情報を表す

```text
Content-Type: application/json
```

により、レスポンスボディがJSONであることを示しています。

## Step 6: リクエスト内容をログに出す

HTTPでは、メソッドやパスが処理の分岐に使われます。  
まずはそれを観察しやすくします。

`cmd/server/main.go` の handler を次のように変更します。

```go
mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
	log.Printf("method=%s path=%s remote=%s", r.Method, r.URL.Path, r.RemoteAddr)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
	})
})
```

再起動して、もう一度実行します。

```bash
curl -i http://localhost:8080/health
```

サーバーログで、メソッドとパスが確認できればOKです。

## Step 7: `/users` 一覧APIを追加する

次に、ユーザー一覧を返すエンドポイントを追加します。

`cmd/server/main.go` を次のようにします。

```go
package main

import (
	"encoding/json"
	"log"
	"net/http"
)

type User struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func main() {
	users := []User{
		{ID: 1, Name: "Alice"},
		{ID: 2, Name: "Bob"},
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status": "ok",
		})
	})

	mux.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(users)
	})

	log.Println("http mini api listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
```

確認します。

```bash
curl -i http://localhost:8080/users
```

例:

```text
HTTP/1.1 200 OK
Content-Type: application/json

[{"id":1,"name":"Alice"},{"id":2,"name":"Bob"}]
```

## Step 8: `POST /users` を追加する

次に、JSONを受け取ってユーザーを追加します。

`/users` handler を次のように拡張します。

```go
type CreateUserRequest struct {
	Name string `json:"name"`
}

nextID := 3

mux.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(users)
	case http.MethodPost:
		var req CreateUserRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		user := User{
			ID:   nextID,
			Name: req.Name,
		}
		nextID++
		users = append(users, user)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(user)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
})
```

確認します。

```bash
curl -i -X POST http://localhost:8080/users \
  -H "Content-Type: application/json" \
  -d '{"name":"Carol"}'
```

例:

```text
HTTP/1.1 201 Created
Content-Type: application/json

{"id":3,"name":"Carol"}
```

続けて一覧を確認します。

```bash
curl -i http://localhost:8080/users
```

## Step 9: なぜ `Content-Type` が必要か確認する

リクエスト側では次のヘッダーを付けています。

```text
Content-Type: application/json
```

これは「送っているボディがJSONである」ことを示します。

APIによっては、このヘッダーがないとJSONとして受け付けないことがあります。

まずはヘッダーありで実行します。

```bash
curl -i -X POST http://localhost:8080/users \
  -H "Content-Type: application/json" \
  -d '{"name":"Dave"}'
```

次に、ヘッダーなしでも動くかを試します。

```bash
curl -i -X POST http://localhost:8080/users \
  -d '{"name":"Eve"}'
```

この課題の簡易実装では通るかもしれません。  
ただし、実務では `Content-Type` をチェックするAPIも多いです。

## Step 10: `DELETE /users/{id}` を追加する

次に、IDを指定して削除します。

`net/http` 標準だけで簡単に進めるため、ここでは `/users/` で前方一致させます。

```go
import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
)
```

```go
mux.HandleFunc("/users/", func(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idText := strings.TrimPrefix(r.URL.Path, "/users/")
	id, err := strconv.Atoi(idText)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	for i, user := range users {
		if user.ID == id {
			users = append(users[:i], users[i+1:]...)
			w.WriteHeader(http.StatusNoContent)
			return
		}
	}

	http.Error(w, "not found", http.StatusNotFound)
})
```

確認します。

```bash
curl -i -X DELETE http://localhost:8080/users/2
```

例:

```text
HTTP/1.1 204 No Content
```

その後で一覧を再確認します。

```bash
curl -i http://localhost:8080/users
```

## Step 11: ここで学ぶHTTPの考え方

### メソッドで操作の種類を分ける

```text
GET    = 取得
POST   = 作成
DELETE = 削除
```

### パスで対象を分ける

```text
/health
/users
/users/2
```

### ステータスコードで結果を伝える

```text
200 OK         = 取得成功
201 Created    = 作成成功
204 No Content = 削除成功
400 Bad Request = リクエスト不正
404 Not Found  = 対象が存在しない
405 Method Not Allowed = 許可していないメソッド
```

### JSONは構造化データを表す

```text
{"name":"Alice"}
```

のように、キーと値でデータを表現します。

## Step 12: レスポンスヘッダーを詳しく見る

次のコマンドでヘッダーだけ表示できます。

```bash
curl -I http://localhost:8080/health
```

ただし、`curl -I` は `HEAD` メソッドを送ります。  
実装によっては `GET` と同じ扱いにならないことがあります。

その違いも観察ポイントです。

`GET` のヘッダーとボディを両方見たいなら、こちらです。

```bash
curl -i http://localhost:8080/health
```

## Step 13: ハンドラーを分離する

1ファイルに書き続けると見通しが悪くなります。  
ここで `internal/httpapi/handler.go` に分けます。

例:

```bash
mkdir -p internal/httpapi
touch internal/httpapi/handler.go
```

分け方の一例:

```go
package httpapi

import "net/http"

type Handler struct {
	mux *http.ServeMux
}

func New() http.Handler {
	mux := http.NewServeMux()

	h := &Handler{mux: mux}
	h.routes()

	return h.mux
}
```

`cmd/server/main.go` は起動処理だけに寄せます。

学ぶこと:

```text
- ルーティングと起動処理を分ける
- handler を小さく保つ
- 後でテストしやすくする
```

## Step 14: テストを書く

HTTP handler は `httptest` を使うとテストできます。

例:

```go
package httpapi_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealth(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("got %d, want %d", rec.Code, http.StatusOK)
	}
}
```

実行します。

```bash
go test ./...
```

学ぶこと:

```text
HTTPサーバー全体を外から叩かなくても、
handler 単位ならメモリ上でテストできる。
```

## Step 15: 最終的な理解チェック

以下を自分の言葉で説明できれば、この課題は完了です。

### Q1. HTTP APIは何をしているか

答えの例:

```text
HTTP APIは、HTTPリクエストを受け取り、
メソッド・パス・ヘッダー・ボディを見て処理を分岐し、
ステータスコード・ヘッダー・ボディを返す。
```

### Q2. `GET /users` と `POST /users` の違いは何か

答えの例:

```text
パスは同じでも、HTTPメソッドが違う。
GET は一覧取得、POST は新規作成のように、
メソッドで操作の種類を分ける。
```

### Q3. ステータスコードは何のためにあるか

答えの例:

```text
リクエストの結果を、クライアントが機械的に判断しやすくするため。
200番台は成功、400番台はクライアント側の問題、
500番台はサーバー側の問題を表すことが多い。
```

### Q4. `Content-Type: application/json` は何を表すか

答えの例:

```text
ボディのデータ形式がJSONであることを表す。
クライアントとサーバーが、どう解釈すべきかを揃えるために必要。
```

### Q5. `curl -i` で何を確認しているか

答えの例:

```text
レスポンスボディだけでなく、
ステータスコードやヘッダーも含めたHTTPレスポンス全体を確認している。
```

### Q6. TCPとHTTPの関係は何か

答えの例:

```text
HTTPはアプリケーション層のプロトコルで、
TCPの上でリクエスト・レスポンス形式を定義している。
TCPはバイト列を運び、HTTPはその意味付けをしている。
```

## 発展課題

余裕があれば、以下に取り組みます。

### 1. バリデーションを追加する

`name` が空なら `400 Bad Request` を返します。

学べること:

```text
- 入力検証
- エラーレスポンス設計
```

### 2. `PUT /users/{id}` を追加する

既存ユーザー名を更新します。

学べること:

```text
- 更新系API
- id指定リソースの設計
```

### 3. メモリではなくファイル保存にする

JSONファイルへ保存・読み込みを試します。

学べること:

```text
- 永続化
- プロセス再起動時のデータ消失
```

### 4. ログに処理時間を出す

各リクエストの開始・終了を記録します。

学べること:

```text
- ミドルウェア的な考え方
- 可観測性
```

## この課題で覚えるべきキーワード

```text
HTTP
GET
POST
DELETE
URL
パス
ヘッダー
Content-Type
ステータスコード
JSON
handler
ServeMux
curl
net/http
```

## この課題のゴール

この課題のゴールは、HTTPについて次のように説明できることです。

```text
HTTPでは、クライアントがメソッド・パス・ヘッダー・ボディを含む
リクエストを送り、サーバーはそれを解釈して処理する。
サーバーは、ステータスコード・ヘッダー・ボディを持つレスポンスを返す。
Goでは net/http を使って、この基本的なやり取りを実装できる。
```
