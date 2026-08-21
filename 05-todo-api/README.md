# 05-todo-api

## 目的

この課題では、PostgreSQL と Go の `database/sql` を使って小さな TODO API を作りながら、DB を使う Web アプリケーションの基本を学びます。

単に SQL を書くだけでなく、CRUD、トランザクション、インデックス、実行計画を自分の手で確認し、なぜその設計にするのかを説明できる状態を目指します。

## この課題で学ぶこと

- PostgreSQL をローカルで準備して接続すること
- Go の `database/sql` で DB 操作を書くこと
- `INSERT` / `SELECT` / `UPDATE` / `DELETE` で CRUD を実装すること
- プレースホルダで値と SQL 構文を分離すること
- `COMMIT` / `ROLLBACK` で変更を確定・取り消しすること
- 単一インデックスと複合インデックスの役割を理解すること
- `EXPLAIN` で実行計画を観察すること

## 完成イメージ

サーバーを起動します。

```bash
go run ./cmd/server
```

作成します。

```bash
curl -i -X POST http://localhost:8080/todos \
  -H "Content-Type: application/json" \
  -d '{"title":"Buy milk","owner_id":1}'
```

一覧を確認します。

```bash
curl -i 'http://localhost:8080/todos?owner_id=1'
```

更新します。

```bash
curl -i -X PATCH http://localhost:8080/todos/1 \
  -H "Content-Type: application/json" \
  -d '{"done":true,"owner_id":1}'
```

削除します。

```bash
curl -i -X DELETE 'http://localhost:8080/todos/1?owner_id=1'
```

つまり、PostgreSQL を使った最小の TODO API を作り、DB の読み書きとトランザクションを観察できるようにします。

## ディレクトリ構成

```text
05-todo-api/
├── README.md
├── go.mod
├── go.sum
├── schema.sql
├── cmd/
│   └── server/
│       └── main.go
└── internal/
    └── todo/
        ├── handler/
        │   └── todo_handler.go
        └── repository/
            └── todo_repository.go
```

最初は `cmd/server/main.go` にまとめて書いても構いません。  
慣れてきたら HTTP 処理と DB 処理を分けます。

## Step 1: DB を準備する

まずはローカル学習用の PostgreSQL を準備します。

```bash
docker run --name go-labs-postgres \
  -e POSTGRES_PASSWORD=localpass \
  -e POSTGRES_DB=tododb \
  -p 5432:5432 \
  -d postgres:16
```

すでに作成済みなら起動だけで構いません。

```bash
docker start go-labs-postgres
```

接続確認をします。

```bash
docker exec go-labs-postgres \
  psql -U postgres -d tododb -c 'SELECT version();'
```

この README では、Mac に `psql` をインストールせず、PostgreSQL コンテナ内の `psql` を使います。
別の SQL で接続を確認する例は次のとおりです。

```bash
docker exec go-labs-postgres psql -U postgres -d tododb -c 'SELECT current_database();'
```

クラウド DB、本番 DB、共有 DB は使わないでください。学習中に `DROP TABLE` や意図的な失敗を試すため、必ずローカルの使い捨て環境で行います。

## Step 2: テーブルを作る

`schema.sql` に次を書きます。

```sql
CREATE TABLE IF NOT EXISTS todos (
    id BIGSERIAL PRIMARY KEY,
    title TEXT NOT NULL,
    done BOOLEAN NOT NULL DEFAULT FALSE,
    owner_id BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS todos_owner_id_idx
    ON todos (owner_id);

CREATE INDEX IF NOT EXISTS todos_owner_done_idx
    ON todos (owner_id, done);
```

適用します。

```bash
docker exec -i go-labs-postgres \
  psql -U postgres -d tododb < schema.sql
```

`schema.sql` は Mac 上にあるため、標準入力を使ってコンテナ内の `psql` に渡しています。
このように標準入力をリダイレクトする場合は、TTY を割り当てる `-t` を付けず、標準入力を開く `-i` だけを指定します。

作成結果を確認します。

```bash
docker exec go-labs-postgres \
  psql -U postgres -d tododb -c '\d todos'
```

ここでは主に次を見ます。

- 主キーが `id` になっているか
- `done` のデフォルト値が `false` か
- 単一インデックスと複合インデックスが作成されているか

## Step 3: Go モジュールと接続コードを作る

最小構成を用意します。

```bash
mkdir -p cmd/server internal/todo
go mod init example.com/go-foundation-labs/05-todo-api
go get github.com/lib/pq
export DATABASE_URL='postgres://postgres:localpass@localhost:5432/tododb?sslmode=disable'
```

まずは接続確認だけを行う最小コードです。

```go
package main

import (
	"context"
	"database/sql"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
)

func main() {
	db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		log.Fatal(err)
	}

	log.Println("connected to postgres")
}
```

実行します。

```bash
go run ./cmd/server
```

`sql.Open` は「接続設定を作る」処理であり、この時点では DB 到達性を保証しません。`Ping` または最初のクエリで確認する必要があります。

## Step 4: CRUD を作る

次に、TODO を作成・取得・更新・削除するコードを書きます。最初は repository に相当する最小コードだけでも十分です。

```go
type Todo struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	Done      bool      `json:"done"`
	OwnerID   int64     `json:"owner_id"`
	CreatedAt time.Time `json:"created_at"`
}

func CreateTodo(ctx context.Context, db *sql.DB, title string, ownerID int64) (int64, error) {
	var id int64
	err := db.QueryRowContext(ctx,
		`INSERT INTO todos (title, owner_id)
		 VALUES ($1, $2)
		 RETURNING id`,
		title, ownerID,
	).Scan(&id)
	return id, err
}

func ListTodos(ctx context.Context, db *sql.DB, ownerID int64) ([]Todo, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT id, title, done, owner_id, created_at
		   FROM todos
		  WHERE owner_id = $1
		  ORDER BY id`,
		ownerID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var todos []Todo
	for rows.Next() {
		var t Todo
		if err := rows.Scan(&t.ID, &t.Title, &t.Done, &t.OwnerID, &t.CreatedAt); err != nil {
			return nil, err
		}
		todos = append(todos, t)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return todos, nil
}

func UpdateTodoDone(ctx context.Context, db *sql.DB, id, ownerID int64, done bool) error {
	result, err := db.ExecContext(ctx,
		`UPDATE todos
		    SET done = $1
		  WHERE id = $2 AND owner_id = $3`,
		done, id, ownerID,
	)
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func DeleteTodo(ctx context.Context, db *sql.DB, id, ownerID int64) error {
	result, err := db.ExecContext(ctx,
		`DELETE FROM todos
		  WHERE id = $1 AND owner_id = $2`,
		id, ownerID,
	)
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return sql.ErrNoRows
	}

	return nil
}
```

ここで重要なのは、`owner_id` を条件に含めて更新・削除の対象を所有者ごとに絞っていることです。ただし、このStepのようにクライアントが送った `owner_id` をそのまま使うだけでは、本当の認可にはなりません。実務では認証済みユーザーの情報から `owner_id` を取得し、リクエストボディやクエリパラメーターの値を所有者情報として信用しないようにします。

このコードを `internal/todo/repository/todo_repository.go` に置く場合は、パッケージ名を `repository` にします。次のStepで別パッケージのハンドラーから呼び出すため、関数名は先頭を大文字にして公開しています。

## Step 5: HTTP ハンドラーと DB 操作を接続する

02-http-mini-api で作ったHTTPハンドラーのインメモリ処理を、このStepでrepositoryの関数へ置き換えます。

- `POST /todos`: JSONを検証し、`repository.CreateTodo` を呼んで `201 Created` を返す
- `GET /todos?owner_id=1`: `owner_id` を数値へ変換し、`repository.ListTodos` の結果をJSONで返す
- `PATCH /todos/{id}`: パスの `id` とリクエストの `owner_id` を使って `repository.UpdateTodoDone` を呼ぶ
- `DELETE /todos/{id}`: `repository.DeleteTodo` を呼び、対象がなければ `404 Not Found` を返す

### ハンドラーを実装する

`internal/todo/handler/todo_handler.go` を作成します。

```go
package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/GenkiSugiyama/go-foundation-labs/05-todo-api/internal/todo/repository"
)

const dbTimeout = 3 * time.Second

type Handler struct {
	db *sql.DB
}

func New(db *sql.DB) *Handler {
	return &Handler{db: db}
}

type createTodoRequest struct {
	Title   string `json:"title"`
	OwnerID int64  `json:"owner_id"`
}

type updateTodoRequest struct {
	Done    *bool `json:"done"`
	OwnerID int64 `json:"owner_id"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req createTodoRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" || req.OwnerID <= 0 {
		writeError(w, http.StatusBadRequest, "title and positive owner_id are required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), dbTimeout)
	defer cancel()

	id, err := repository.CreateTodo(ctx, h.db, req.Title, req.OwnerID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create todo")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]int64{"id": id})
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	ownerID, err := positiveInt64(r.URL.Query().Get("owner_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "owner_id must be a positive integer")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), dbTimeout)
	defer cancel()

	todos, err := repository.ListTodos(ctx, h.db, ownerID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list todos")
		return
	}
	if todos == nil {
		todos = []repository.Todo{}
	}

	writeJSON(w, http.StatusOK, todos)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := positiveInt64(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "id must be a positive integer")
		return
	}

	var req updateTodoRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Done == nil || req.OwnerID <= 0 {
		writeError(w, http.StatusBadRequest, "done and positive owner_id are required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), dbTimeout)
	defer cancel()

	err = repository.UpdateTodoDone(ctx, h.db, id, req.OwnerID, *req.Done)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "todo not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update todo")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := positiveInt64(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "id must be a positive integer")
		return
	}

	ownerID, err := positiveInt64(r.URL.Query().Get("owner_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "owner_id must be a positive integer")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), dbTimeout)
	defer cancel()

	err = repository.DeleteTodo(ctx, h.db, id, ownerID)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "todo not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete todo")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func positiveInt64(value string) (int64, error) {
	n, err := strconv.ParseInt(value, 10, 64)
	if err != nil || n <= 0 {
		return 0, errors.New("value must be a positive integer")
	}
	return n, nil
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorResponse{Error: message})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
```

`PATCH`の`done`を`*bool`にしているのは、`false`が指定された場合と、フィールド自体が省略された場合を区別するためです。また、repositoryから返されたDBエラーの内容はレスポンスへ直接出さず、クライアント向けの固定メッセージに変換します。

### ルーティングを設定する

`cmd/server/main.go` の接続確認後に、ハンドラーとルートを登録してHTTPサーバーを起動します。Go 1.22以降の`http.ServeMux`では、HTTPメソッドとパスを組み合わせたパターンと、`{id}`のようなパス変数を利用できます。

```go
todoHandler := handler.New(db)

mux := http.NewServeMux()
mux.HandleFunc("POST /todos", todoHandler.Create)
mux.HandleFunc("GET /todos", todoHandler.List)
mux.HandleFunc("PATCH /todos/{id}", todoHandler.Update)
mux.HandleFunc("DELETE /todos/{id}", todoHandler.Delete)

server := &http.Server{
	Addr:              ":8080",
	Handler:           mux,
	ReadHeaderTimeout: 5 * time.Second,
}

log.Println("listening on http://localhost:8080")
log.Fatal(server.ListenAndServe())
```

`main.go` のimportには次を追加します。

```go
import (
	"net/http"

	"github.com/GenkiSugiyama/go-foundation-labs/05-todo-api/internal/todo/handler"
)
```

コードを配置したら、フォーマットとコンパイルを確認します。

```bash
go fmt ./...
go test ./...
go run ./cmd/server
```

各リクエストでは `context.WithTimeout` でDB処理に期限を設け、クライアント入力をそのままSQL文字列へ連結しないでください。実装後、完成イメージの `curl` を上から順に実行し、HTTPレスポンスとDBの行が一致することを確認します。

HTTPの解析、JSON、ステータスコードが曖昧な場合は、先に `02-http-mini-api` の該当Stepへ戻ります。この課題ではHTTPを作り直すことより、HTTP層と永続化層の境界を説明できることが重要です。

## Step 6: プレースホルダを観察する

危険な例です。

```go
query := "SELECT id, title FROM todos WHERE title = '" + userInput + "'"
```

これは、入力値が SQL の一部として解釈される可能性があります。

安全な例です。

```go
rows, err := db.QueryContext(ctx,
	`SELECT id, title FROM todos WHERE title = $1`,
	userInput,
)
```

プレースホルダを使うと、SQL 構文と値が分離されます。

実際に確認したい場合は、危険なコードを本番環境や共有 DB で試してはいけません。  
ローカルの使い捨て DB だけで「なぜ文字列連結が危険なのか」を説明できるようにしてください。

## Step 7: トランザクションを作る

次は `COMMIT` と `ROLLBACK` を観察します。

```go
func markDoneTx(ctx context.Context, db *sql.DB, id, ownerID int64) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx,
		`UPDATE todos
		    SET done = TRUE
		  WHERE id = $1 AND owner_id = $2`,
		id, ownerID,
	); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx,
		`INSERT INTO todos (title, owner_id)
		 VALUES ($1, $2)`,
		"audit: updated todo", ownerID,
	); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}
```

途中でわざとエラーを返すと `ROLLBACK` され、更新は残りません。  
最後まで成功したときだけ `COMMIT` されます。

確認用の SQL 例です。

```bash
docker exec go-labs-postgres \
  psql -U postgres -d tododb -c \
  'SELECT id, title, done, owner_id FROM todos ORDER BY id;'
```

## Step 8: `EXPLAIN` でインデックスを観察する

まずはテストデータを少し入れます。

```bash
docker exec go-labs-postgres \
  psql -U postgres -d tododb -c "
INSERT INTO todos (title, done, owner_id)
SELECT 'task-' || gs, (gs % 2 = 0), (gs % 5) + 1
FROM generate_series(1, 1000) AS gs;
"
```

実行計画を確認します。

```bash
docker exec go-labs-postgres \
  psql -U postgres -d tododb -c \
  'EXPLAIN SELECT id, title FROM todos WHERE owner_id = 1;'
```

```bash
docker exec go-labs-postgres \
  psql -U postgres -d tododb -c \
  'EXPLAIN SELECT id, title FROM todos WHERE owner_id = 1 AND done = false;'
```

小さい表では `Seq Scan` になることもあります。  
それは「インデックスが無意味」という意味ではなく、PostgreSQL がその時点のデータ量や統計情報から順次走査の方が安いと判断しただけです。

複合インデックス `(owner_id, done)` は、左から順に使われやすいことも確認してください。

今回の検索だけを見ると、複合インデックスの先頭列が `owner_id` なので、単一インデックス `(owner_id)` と役割が重なる場合があります。インデックスは多いほどよいわけではなく、書き込みコストと容量も増えます。両方の実行計画を比較した後、単一インデックスを残す必要があるか考えてください。

## ここで確認すること

- `sql.Open` と `PingContext` の役割の違い
- CRUD の各 SQL がどの操作に対応しているか
- `rows.Close()` と `rows.Err()` を確認する理由
- `COMMIT` と `ROLLBACK` がどの分岐で起こるか
- プレースホルダが SQL インジェクション対策の基本であること
- 単一インデックスと複合インデックスの使い分け
- `EXPLAIN` は実行結果ではなく実行計画であること

## 詰まったこと

学習後に記入します。

- PostgreSQL の準備や接続確認で、どこで止まりましたか。
- `ROLLBACK` を自分で確認するために、どのエラーの入れ方を試しましたか。
- `EXPLAIN` の出力のどの行が読みづらかったですか。

## 分かったこと

学習後に記入します。

- `INSERT` / `SELECT` / `UPDATE` / `DELETE` を、それぞれ何のために使うか説明してください。
- `defer tx.Rollback()` を先に書く理由を説明してください。
- 複合インデックスの列順が重要な理由を、自分の言葉で説明してください。

## 実務での関係

実務では、API の正しさだけでなく、データ破壊を防ぐこと、途中失敗で不整合を残さないこと、必要なクエリだけを速くすることが重要です。  
また、プレースホルダで SQL インジェクションを防いでも、`owner_id` などの認可条件を忘れると他人のデータを操作できてしまいます。

## 理解度チェック

1. `sql.Open` だけでは接続確認にならないのはなぜですか。
2. `UPDATE todos SET done = $1 WHERE id = $2` だけでは不十分なことがあるのはなぜですか。
3. `COMMIT` 前にエラーが起きたとき、なぜ `ROLLBACK` が必要ですか。
4. `(owner_id, done)` の複合インデックスは、どんな検索条件で役に立ちやすいですか。
5. `EXPLAIN` に `Seq Scan` が出たとき、すぐに「失敗だ」と判断してはいけないのはなぜですか。

## 発展課題

- `GET /todos/{id}` を追加する
- `context.WithTimeout` を各 DB 操作に入れる
- `created_at DESC` の一覧に対して別のインデックス案を考える
- `EXPLAIN ANALYZE` をローカル環境で実行し、推定コストと実測の違いを見る
- `BEGIN` 中に複数の TODO を一括作成する処理を追加する

## 覚えるべきキーワード

`PostgreSQL`、`database/sql`、`CRUD`、`INSERT`、`SELECT`、`UPDATE`、`DELETE`、`プレースホルダ`、`トランザクション`、`COMMIT`、`ROLLBACK`、`インデックス`、`複合インデックス`、`EXPLAIN`

## この課題のゴール

PostgreSQL をローカルで準備し、Go の `database/sql` で TODO の CRUD を実装し、プレースホルダ、`COMMIT` / `ROLLBACK`、インデックス、`EXPLAIN` の役割を自分の言葉で説明できれば完了です。
