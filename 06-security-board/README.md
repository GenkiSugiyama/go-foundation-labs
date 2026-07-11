# 06-security-board

## 目的

この課題では、あえて脆弱な掲示板をローカル学習環境で作り、その後に安全な実装へ修正しながら Web セキュリティの基本を学びます。

最初に強く注意します。  
この課題は自分のローカル学習環境だけで行ってください。脆弱なコードの再現、攻撃文字列の投入、Cookie 設定の変更は、共有環境・社内環境・公開サーバー・第三者のサービスでは絶対に行ってはいけません。

## この課題で学ぶこと

- SQL インジェクションがなぜ起きるか
- XSS がなぜ起きるか
- CSRF がどのような状況で成立するか
- Cookie 属性 `HttpOnly` / `Secure` / `SameSite` の意味
- 認証と認可の違い
- 脆弱な実装を安全な実装へ置き換える考え方

## 完成イメージ

最初は、次のような危険な状態をローカルでだけ再現します。

- ログイン SQL が文字列連結になっている
- 投稿本文がそのまま HTML として表示される
- 状態変更 API に CSRF 対策がない
- セッション Cookie の属性が弱い
- ログイン済みなら他人の投稿も削除できる

その後、次のように直します。

- SQL はプレースホルダで実行する
- 投稿本文は HTML エスケープして表示する
- 状態変更時に CSRF トークンを確認する
- Cookie に適切な属性を付ける
- 投稿者本人だけが削除できるよう認可を入れる

## ディレクトリ構成

```text
06-security-board/
├── README.md
├── go.mod
├── schema.sql
├── cmd/
│   └── server/
│       └── main.go
└── internal/
    └── board/
        ├── auth.go
        ├── handler.go
        └── repository.go
```

最初は `cmd/server/main.go` に最小構成で書いても構いません。  
ただし、脆弱な実装と修正版の差分を説明しやすいように、後で責務を分ける前提で進めます。

## Step 1: ローカル専用の学習環境を準備する

この課題では、脆弱な実装を自分のローカル環境でのみ再現します。

```bash
docker run --name go-labs-sec-postgres \
  -e POSTGRES_PASSWORD=localpass \
  -e POSTGRES_DB=boarddb \
  -p 5433:5432 \
  -d postgres:16
```

接続確認をします。

```bash
psql 'postgres://postgres:localpass@localhost:5433/boarddb?sslmode=disable' \
  -c 'SELECT current_database();'
```

`psql` がない場合はコンテナ内から確認しても構いません。

```bash
docker exec -it go-labs-sec-postgres psql -U postgres -d boarddb -c '\dt'
```

この課題で扱う「再現」は、外部に向けた攻撃ではありません。  
自分で用意したローカル DB とローカルサーバーに対してだけ、危険な挙動を観察します。

## Step 2: 最小の掲示板テーブルを作る

`schema.sql` の例です。

```sql
CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    password TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS posts (
    id BIGSERIAL PRIMARY KEY,
    author_id BIGINT NOT NULL REFERENCES users(id),
    body TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

適用します。

```bash
psql 'postgres://postgres:localpass@localhost:5433/boarddb?sslmode=disable' -f schema.sql
```

サンプルデータも入れておくと確認しやすくなります。

```bash
psql 'postgres://postgres:localpass@localhost:5433/boarddb?sslmode=disable' -c "
INSERT INTO users (email, password) VALUES
  ('alice@example.com', 'pass1234'),
  ('bob@example.com', 'pass5678')
ON CONFLICT DO NOTHING;
"
```

この段階では、あえて平文パスワードで始めても構いません。  
ただし「学習用に一時的に危険な状態を作っている」ことを意識し、後で必ず改善します。

## Step 3: まずは脆弱な実装を読む

最初に、危険な例を見ます。

```go
query := "SELECT id FROM users WHERE email = '" + email + "' AND password = '" + password + "'"
row := db.QueryRowContext(ctx, query)
```

```go
fmt.Fprintf(w, "<li>%s</li>", post.Body)
```

```go
http.SetCookie(w, &http.Cookie{
	Name:  "session",
	Value: token,
	Path:  "/",
})
```

```go
_, err := db.ExecContext(ctx,
	`DELETE FROM posts WHERE id = $1`,
	postID,
)
```

これらの問題点を、コードを書き始める前に説明してみてください。

- なぜ文字列連結の SQL が危険なのか
- なぜ本文をそのまま HTML に埋め込むと危険なのか
- なぜ Cookie 属性が弱いと困るのか
- なぜ認証済みでも認可が不足すると危険なのか

## Step 4: 最小の掲示板を作る

まずはローカルで動く最小構成のサーバーを作ります。

```go
package main

import (
	"database/sql"
	"html/template"
	"log"
	"net/http"
	"os"

	_ "github.com/lib/pq"
)

var page = template.Must(template.New("index").Parse(`
<html>
  <body>
    <h1>Security Board</h1>
    <form method="post" action="/posts">
      <textarea name="body"></textarea>
      <button type="submit">post</button>
    </form>
  </body>
</html>
`))

func main() {
	db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if err := page.Execute(w, nil); err != nil {
			http.Error(w, "template error", http.StatusInternalServerError)
		}
	})

	log.Println("security board listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
```

実行します。

```bash
go mod init example.com/go-foundation-labs/06-security-board
go get github.com/lib/pq
export DATABASE_URL='postgres://postgres:localpass@localhost:5433/boarddb?sslmode=disable'
go run ./cmd/server
```

ブラウザで確認する代わりに、まずは `curl` でも見ておきます。

```bash
curl -i http://localhost:8080/
```

## Step 5: SQL インジェクションをローカルでだけ再現し、修正する

危険なログイン例です。

```go
query := "SELECT id FROM users WHERE email = '" + email + "' AND password = '" + password + "'"
row := db.QueryRowContext(ctx, query)
```

これは、入力が SQL 構文として扱われる余地を作ります。  
このコードは学習のために読むだけにし、共有環境や公開環境では絶対に使わないでください。

修正版です。

```go
row := db.QueryRowContext(ctx,
	`SELECT id FROM users WHERE email = $1 AND password = $2`,
	email, password,
)
```

ここで確認するべきなのは、危険な文字列を「試して突破すること」ではなく、なぜプレースホルダで構文と値が分離されるのかを説明できることです。

## Step 6: XSS をローカルでだけ再現し、修正する

危険な表示例です。

```go
fmt.Fprintf(w, "<li>%s</li>", post.Body)
```

投稿本文に HTML やスクリプト風の文字列が入ると、表示側でそのまま解釈される可能性があります。

修正版です。

```go
tmpl := template.Must(template.New("posts").Parse(`
<ul>
  {{range .}}
    <li>{{.Body}}</li>
  {{end}}
</ul>
`))
```

`html/template` は HTML 文脈に応じてエスケープします。  
対して `text/template` や `fmt.Fprintf` は、Web 画面表示の安全性を自動では保証しません。

## Step 7: CSRF と Cookie 属性を確認して修正する

危険な Cookie 例です。

```go
http.SetCookie(w, &http.Cookie{
	Name:  "session",
	Value: token,
	Path:  "/",
})
```

修正版の例です。

```go
http.SetCookie(w, &http.Cookie{
	Name:     "session",
	Value:    token,
	Path:     "/",
	HttpOnly: true,
	SameSite: http.SameSiteLaxMode,
	Secure:   true,
})
```

`Secure: true` は HTTPS 前提です。ローカルで HTTP しか使わないと Cookie が送られず戸惑うことがあります。  
その場合は「本番では `Secure` を付ける」「ローカル検証では HTTPS を使うか、一時的に差を理解した上で切り替える」という形で扱ってください。

CSRF 対策の最小例です。

```go
func requireCSRFFromForm(r *http.Request) bool {
	cookie, err := r.Cookie("csrf_token")
	if err != nil {
		return false
	}
	return r.FormValue("csrf_token") == cookie.Value
}
```

この例は仕組みを観察するための最小形です。トークンは `crypto/rand` で十分な長さの予測不能な値を生成し、セッションと結び付けるか、改ざんを検出できる形で扱います。`SameSite` は防御の一部ですが、すべての状況でCSRFトークンの代わりになるわけではありません。実務では十分に検証されたCSRFミドルウェアの利用も検討します。

状態変更の POST で確認します。

```go
if !requireCSRFFromForm(r) {
	http.Error(w, "invalid csrf token", http.StatusForbidden)
	return
}
```

ここで学びたいのは、「ログイン済み」だけでは状態変更を安全と判断できないことです。

## Step 8: 認証と認可を分けて考える

認証は「誰としてログインしているか」です。  
認可は「その人にその操作を許してよいか」です。

危険な削除例です。

```go
_, err := db.ExecContext(ctx,
	`DELETE FROM posts WHERE id = $1`,
	postID,
)
```

これは、ログインしている誰でも投稿 ID さえ分かれば削除できる可能性があります。

修正版です。

```go
result, err := db.ExecContext(ctx,
	`DELETE FROM posts
	  WHERE id = $1 AND author_id = $2`,
	postID, currentUserID,
)
```

`RowsAffected` を確認し、削除対象がなかったときの扱いも考えます。

```go
affected, err := result.RowsAffected()
if err != nil {
	return err
}
if affected == 0 {
	http.Error(w, "not found", http.StatusNotFound)
	return
}
```

## Step 9: 平文パスワード保存を修正する

平文パスワードは、ローカル学習用であっても最終状態に残してはいけません。SQLインジェクションの差を確認した後は、使い捨てDBを作り直し、`users.password` を `users.password_hash` に変更します。

```sql
CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL
);
```

Goではパスワードそのものを保存せず、登録時にハッシュ化し、ログイン時に照合します。

```bash
go get golang.org/x/crypto/bcrypt
```

```go
hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
if err != nil {
	return err
}

_, err = db.ExecContext(ctx,
	`INSERT INTO users (email, password_hash) VALUES ($1, $2)`,
	email, string(hash),
)
```

```go
var passwordHash string
err := db.QueryRowContext(ctx,
	`SELECT password_hash FROM users WHERE email = $1`,
	email,
).Scan(&passwordHash)
if err != nil {
	return err
}

if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password)); err != nil {
	return errors.New("invalid credentials")
}
```

ログイン失敗時は、メールアドレスが存在しない場合とパスワードが違う場合で外部向けメッセージを分けないようにします。

## Step 10: コマンドで観察する

ログインや投稿を手で確認します。

```bash
curl -i -c cookies.txt http://localhost:8080/
```

```bash
curl -i -b cookies.txt -c cookies.txt -X POST http://localhost:8080/login \
  -H "Content-Type: application/x-www-form-urlencoded" \
  --data 'email=alice@example.com&password=pass1234'
```

```bash
curl -i -b cookies.txt -X POST http://localhost:8080/posts \
  -H "Content-Type: application/x-www-form-urlencoded" \
  --data 'body=hello&csrf_token=...'
```

Cookie を観察します。

```bash
cat cookies.txt
```

ここでは次を見ます。

- `HttpOnly` / `Secure` / `SameSite` の有無
- CSRF トークンが状態変更時に必要か
- 他人の投稿削除が拒否されるか

macOS では `sed -n` や `grep` のオプション差が小さいですが、Linux の方が `curl` や `psql` が最初から入っていないことがあります。  
コマンドがないときは先にツール導入状況を確認してください。

## ここで確認すること

- SQL インジェクションが「文字列連結」と関係すること
- XSS が「表示時の文脈」と関係すること
- CSRF が「ブラウザが自動で Cookie を送ること」と関係すること
- `HttpOnly` / `Secure` / `SameSite` の役割の違い
- 認証と認可が別物であること
- 脆弱な実装を「再現して終わり」にせず、修正まで説明すること

## 詰まったこと

学習後に記入します。

- 脆弱な例のどこが危険なのか、最初に理解しづらかった点は何ですか。
- Cookie 属性や CSRF 対策で、どの挙動が予想と違いましたか。
- 認証と認可の違いを考えるとき、どの画面や API で迷いましたか。

## 分かったこと

学習後に記入します。

- SQL インジェクションを防ぐために、なぜプレースホルダを使うのか説明してください。
- XSS を防ぐために、なぜ `html/template` が有効なのか説明してください。
- 認証と認可の違いを、掲示板の削除機能を例に説明してください。

## 実務での関係

実務では、セキュリティの問題は「珍しい特殊攻撃」ではなく、普通の実装ミスとして入り込みます。  
入力の扱い、出力の扱い、Cookie 設定、状態変更 API、所有者確認は、どれも日常的な機能に含まれます。脆弱性を知識として暗記するだけでは足りず、危険なコードを見て止められることが重要です。

## 理解度チェック

1. SQL インジェクションは、なぜ文字列連結で起きやすいのですか。
2. `fmt.Fprintf(w, post.Body)` のような表示が危険になりうるのはなぜですか。
3. `HttpOnly` と `SameSite` は、それぞれ何を守ろうとしていますか。
4. ログイン済みでも、なぜ他人の投稿削除を別途チェックしなければいけませんか。
5. この課題で脆弱な実装を公開環境で試してはいけないのはなぜですか。

## 発展課題

- ログイン失敗回数の制御を追加する
- 管理者だけ削除できる API と投稿者本人だけ削除できる API を分けて比較する
- `Content-Security-Policy` を追加して、出力エスケープとの役割の違いを整理する
- `SameSite=Lax` と `SameSite=Strict` の違いをローカルで整理する

## 覚えるべきキーワード

`SQLインジェクション`、`XSS`、`CSRF`、`Cookie`、`HttpOnly`、`Secure`、`SameSite`、`認証`、`認可`、`プレースホルダ`、`html/template`

## この課題のゴール

脆弱な掲示板を自分のローカル学習環境でのみ再現し、SQL インジェクション、XSS、CSRF、Cookie 属性、認証、認可の問題を安全な実装へ修正した上で、その理由を自分の言葉で説明できれば完了です。
