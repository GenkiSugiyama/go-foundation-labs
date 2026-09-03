# 06-security-board

## 目的

この課題では、あえて脆弱な掲示板を自分で実装し、ローカル環境で問題を再現した後、同じアプリを安全な実装へ修正します。

完成済みのアプリを起動する課題ではありません。以下の仕様と確認手順に沿って、まず「対策前」を作り、観察結果を残してから「対策後」へ書き換えてください。

> **警告**
>
> 脆弱版は `localhost` だけで起動し、学習用の使い捨て DB だけを使用してください。共有環境、社内環境、公開サーバー、第三者のサービスでは、脆弱なコードの実行や攻撃文字列の投入を絶対に行わないでください。

## この課題で学ぶこと

- SQL インジェクションが「入力と SQL 構文の混在」で起きること
- 保存型 XSS が「保存時」ではなく主に「HTML 出力時」の問題であること
- CSRF がブラウザによる Cookie の自動送信を利用すること
- Cookie の `HttpOnly` / `Secure` / `SameSite` がそれぞれ別の役割を持つこと
- 認証と認可は別の確認であること
- パスワードを平文で保存してはいけないこと
- 対策後に同じ操作を繰り返して、修正の効果を確認すること

## 完成条件

次の順番で作業します。

1. ログイン、投稿一覧、投稿作成、投稿削除を持つ脆弱版を実装する
2. 通常操作ができることを確認する
3. 5 種類の問題をローカルで再現し、結果を記録する
4. 脆弱版を Git のコミットとして残す
5. SQL、HTML 出力、CSRF、Cookie、認可、パスワード保存を修正する
6. 同じ確認を繰り返し、攻撃操作が失敗することを確認する
7. 脆弱版と対策版の差分を説明する

この課題の対策版は、学習対象の差を確認するための最小アプリです。レート制限、セッション永続化、監査ログ、TLS 終端などは含まないため、そのまま本番投入できる完成品ではありません。

## 実装する HTTP インターフェース

手順と動作を揃えるため、最初は次のルートで実装します。

| Method | Path | 動作 | ログイン |
|---|---|---|---|
| `GET` | `/login` | ログインフォームを表示 | 不要 |
| `POST` | `/login` | 認証してセッション Cookie を発行 | 不要 |
| `GET` | `/` | 投稿一覧と投稿フォームを表示 | 必要 |
| `POST` | `/posts` | 投稿を作成 | 必要 |
| `POST` | `/posts/delete` | form の `id` で投稿を削除 | 必要 |
| `POST` | `/logout` | セッションを破棄 | 必要 |

HTML の `<form>` は `DELETE` を直接送れないため、この課題の削除は `POST /posts/delete` とします。状態を変更する処理を `GET` にしないでください。

## 推奨ディレクトリ構成

```text
06-security-board/
├── README.md
├── go.mod
├── go.sum
├── schema.sql
├── cmd/
│   └── server/
│       └── main.go
└── internal/
    └── board/
        ├── auth.go
        ├── handler.go
        ├── repository.go
        └── session.go
```

最初は `cmd/server/main.go` だけで実装して構いません。脆弱版が動いて観察を終えるまでは、リファクタリングより挙動の理解を優先します。

## Step 1: ローカル専用 DB を準備する

PostgreSQL コンテナを `localhost:5433` だけで使用します。

```bash
docker run --name go-labs-sec-postgres \
  -e POSTGRES_PASSWORD=localpass \
  -e POSTGRES_DB=boarddb \
  -p 127.0.0.1:5433:5432 \
  -d postgres:16
```

すでに作成済みなら起動します。

```bash
docker start go-labs-sec-postgres
```

接続確認をします。

```bash
docker exec go-labs-sec-postgres \
  psql -U postgres -d boarddb -c 'SELECT current_database();'
```

`-p 127.0.0.1:5433:5432` としているのは、学習用 DB を外部インターフェースへ公開しないためです。

## Step 2: 脆弱版用のテーブルを作る

最初は、SQL インジェクションと平文パスワードの問題を観察するため、`schema.sql` の `users.password` をそのまま使用します。この列は対策版で削除します。

```bash
docker exec -i go-labs-sec-postgres \
  psql -U postgres -d boarddb < schema.sql
```

利用者を 2 人登録します。

```bash
docker exec go-labs-sec-postgres psql -U postgres -d boarddb -c "
INSERT INTO users (email, password) VALUES
  ('alice@example.com', 'pass1234'),
  ('bob@example.com', 'pass5678')
ON CONFLICT (email) DO NOTHING;
"
```

Alice と Bob の投稿を 1 件ずつ作ります。

```bash
docker exec go-labs-sec-postgres psql -U postgres -d boarddb -c "
INSERT INTO posts (author_id, body)
SELECT id, 'Alice の投稿' FROM users WHERE email = 'alice@example.com';
INSERT INTO posts (author_id, body)
SELECT id, 'Bob の投稿' FROM users WHERE email = 'bob@example.com';
"
```

何度も実行すると投稿が増えるため、やり直すときは使い捨て DB を初期化してから再適用してください。

```bash
docker exec go-labs-sec-postgres \
  psql -U postgres -d boarddb \
  -c 'TRUNCATE posts, users RESTART IDENTITY CASCADE;'
```

## Step 3: 脆弱版の実装仕様を確認する

脆弱版には、学習対象として次の問題を意図的に入れます。それ以外の箇所まで無制限に危険にする必要はありません。

| 対象 | 脆弱版の実装 |
|---|---|
| ログイン SQL | email と password を SQL 文字列へ連結する |
| 投稿表示 | body をエスケープせず HTML へ埋め込む |
| CSRF | 状態変更時にトークンを検証しない |
| Cookie | `Path` だけを設定し、セキュリティ属性を付けない |
| 投稿削除 | 投稿 ID だけを条件にし、所有者を確認しない |
| パスワード | DB の平文 password と直接比較する |

たとえば、脆弱なログイン SQL は入力値をそのまま連結した形です。

```go
query := "SELECT id, email FROM users WHERE email = '" + email +
	"' AND password = '" + password + "'"
```

脆弱な投稿表示は、`body` を `html/template` の通常の値として渡さず、そのままレスポンスへ書き込みます。

```go
fmt.Fprintf(w, "<div>%s</div>", body)
```

これらは脆弱版の再現に限定したコードです。コードコメントにも `intentionally vulnerable` などと明記してください。

## Step 4: 脆弱版の掲示板を実装する

ここまでの `cmd/server/main.go` は画面を表示するだけなので、掲示板としては未完成です。次のサンプルを参考に、ログイン、一覧、投稿、削除、ログアウトを実装してください。

このサンプルは、後の Step で問題を再現するため意図的に次の脆弱性を含みます。

- ログイン SQL の文字列連結
- `template.HTML` による投稿本文のエスケープ回避
- CSRF トークンなしの状態変更
- セキュリティ属性なしの Cookie
- 所有者を確認しない投稿削除
- 平文パスワードでの照合

セッショントークンの推測まで学習対象にすると問題の切り分けが難しくなるため、そこだけは最初から `crypto/rand` で生成します。

### Step 4-1: 最小サンプルを `main.go` に実装する

以下は 1 ファイルで動作する脆弱版の例です。最初はこのまま動かし、各処理の流れを確認してから関数分割してください。

```go
package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "github.com/lib/pq"
)

type Session struct {
	UserID int64
	Email  string
}

type Post struct {
	ID       int64
	AuthorID int64
	Author   string
	Body     template.HTML // intentionally vulnerable
}

type PageData struct {
	Current Session
	Posts   []Post
}

type application struct {
	db       *sql.DB
	sessions map[string]Session
	mu       sync.RWMutex
}

var loginPage = template.Must(template.New("login").Parse(`<!doctype html>
<html lang="ja">
<head><meta charset="utf-8"><title>Login</title></head>
<body>
  <h1>Security Board - Login</h1>
  <form method="post" action="/login">
    <label>email <input type="email" name="email" required></label><br>
    <label>password <input type="password" name="password" required></label><br>
    <button type="submit">login</button>
  </form>
</body>
</html>`))

var boardPage = template.Must(template.New("board").Parse(`<!doctype html>
<html lang="ja">
<head><meta charset="utf-8"><title>Security Board</title></head>
<body>
  <h1>Security Board</h1>
  <p>login: {{.Current.Email}} / user ID: {{.Current.UserID}}</p>
  <form method="post" action="/logout">
    <button type="submit">logout</button>
  </form>

  <h2>新規投稿</h2>
  <form method="post" action="/posts">
    <textarea name="body" required></textarea>
    <button type="submit">post</button>
  </form>

  <h2>投稿一覧</h2>
  {{range .Posts}}
    <article>
      <p>#{{.ID}} by {{.Author}} / author ID: {{.AuthorID}}</p>
      <div>{{.Body}}</div>
      <form method="post" action="/posts/delete">
        <input type="hidden" name="id" value="{{.ID}}">
        <button type="submit">delete</button>
      </form>
    </article>
  {{else}}
    <p>投稿はありません。</p>
  {{end}}
</body>
</html>`))

func main() {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		log.Fatal(err)
	}

	app := &application{
		db:       db,
		sessions: make(map[string]Session),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", app.home)
	mux.HandleFunc("/login", app.login)
	mux.HandleFunc("/posts", app.createPost)
	mux.HandleFunc("/posts/delete", app.deletePost)
	mux.HandleFunc("/logout", app.logout)

	server := &http.Server{
		Addr:              "localhost:8080",
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Println("security board listening on http://localhost:8080")
	log.Fatal(server.ListenAndServe())
}

func (app *application) login(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		if err := loginPage.Execute(w, nil); err != nil {
			log.Printf("render login: %v", err)
		}
	case http.MethodPost:
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}

		email := r.FormValue("email")
		password := r.FormValue("password")

		// intentionally vulnerable: 入力値を SQL 文字列へ連結している
		query := "SELECT id, email FROM users WHERE email = '" + email +
			"' AND password = '" + password + "'"

		var current Session
		if err := app.db.QueryRowContext(r.Context(), query).
			Scan(&current.UserID, &current.Email); err != nil {
			http.Error(w, "invalid email or password", http.StatusUnauthorized)
			return
		}

		token, err := randomToken()
		if err != nil {
			http.Error(w, "failed to create session", http.StatusInternalServerError)
			return
		}
		app.mu.Lock()
		app.sessions[token] = current
		app.mu.Unlock()

		// intentionally vulnerable: Cookie のセキュリティ属性がない
		http.SetCookie(w, &http.Cookie{
			Name:  "session",
			Value: token,
			Path:  "/",
		})
		http.Redirect(w, r, "/", http.StatusSeeOther)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (app *application) home(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	_, current, ok := app.currentSession(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	posts, err := app.listPosts(r.Context())
	if err != nil {
		log.Printf("list posts: %v", err)
		http.Error(w, "failed to load posts", http.StatusInternalServerError)
		return
	}
	if err := boardPage.Execute(w, PageData{Current: current, Posts: posts}); err != nil {
		log.Printf("render board: %v", err)
	}
}

func (app *application) listPosts(ctx context.Context) ([]Post, error) {
	rows, err := app.db.QueryContext(ctx, `
		SELECT p.id, p.author_id, u.email, p.body
		FROM posts AS p
		JOIN users AS u ON u.id = p.author_id
		ORDER BY p.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []Post
	for rows.Next() {
		var post Post
		var body string
		if err := rows.Scan(
			&post.ID, &post.AuthorID, &post.Author, &body,
		); err != nil {
			return nil, err
		}
		// intentionally vulnerable: 保存された本文を安全な HTML と見なしている
		post.Body = template.HTML(body)
		posts = append(posts, post)
	}
	return posts, rows.Err()
}

func (app *application) createPost(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	_, current, ok := app.currentSession(r)
	if !ok {
		http.Error(w, "login required", http.StatusUnauthorized)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	body := strings.TrimSpace(r.FormValue("body"))
	if body == "" {
		http.Error(w, "body is required", http.StatusBadRequest)
		return
	}

	_, err := app.db.ExecContext(r.Context(),
		`INSERT INTO posts (author_id, body) VALUES ($1, $2)`,
		current.UserID, body,
	)
	if err != nil {
		log.Printf("create post: %v", err)
		http.Error(w, "failed to create post", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (app *application) deletePost(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if _, _, ok := app.currentSession(r); !ok {
		http.Error(w, "login required", http.StatusUnauthorized)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	postID, err := strconv.ParseInt(r.FormValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid post id", http.StatusBadRequest)
		return
	}

	// intentionally vulnerable: ログイン利用者が投稿者本人か確認していない
	if _, err := app.db.ExecContext(r.Context(),
		`DELETE FROM posts WHERE id = $1`, postID,
	); err != nil {
		log.Printf("delete post: %v", err)
		http.Error(w, "failed to delete post", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (app *application) logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	token, _, ok := app.currentSession(r)
	if ok {
		app.mu.Lock()
		delete(app.sessions, token)
		app.mu.Unlock()
	}
	http.SetCookie(w, &http.Cookie{
		Name:   "session",
		Path:   "/",
		MaxAge: -1,
	})
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (app *application) currentSession(r *http.Request) (string, Session, bool) {
	cookie, err := r.Cookie("session")
	if err != nil {
		return "", Session{}, false
	}
	app.mu.RLock()
	current, ok := app.sessions[cookie.Value]
	app.mu.RUnlock()
	return cookie.Value, current, ok
}

func randomToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
```

この時点では、投稿作成の SQL にはプレースホルダを使っています。SQL インジェクションの観察対象はログイン処理だけに絞り、無関係な脆弱性を増やさないためです。

### Step 4-2: 起動して通常操作を確認する

実行します。

```bash
export DATABASE_URL='postgres://postgres:localpass@localhost:5433/boarddb?sslmode=disable'
go run ./cmd/server
```

別のターミナルで確認します。

```bash
curl -i http://localhost:8080/login
```

ブラウザで `http://localhost:8080/login` を開き、最初に通常のログイン、投稿、削除、ログアウトができることを確認します。

## Step 5: 脆弱版で SQL インジェクションを再現する

正しいパスワードを使わず、ローカルアプリにだけ次を送ります。

```bash
curl -i -c vulnerable.cookies \
  -X POST http://localhost:8080/login \
  --data-urlencode "email=' OR 1=1 -- " \
  --data-urlencode 'password=incorrect'
```

ログイン成功時に `/` へリダイレクトする実装なら、`303 See Other` と `Set-Cookie` が返り、次のリクエストで投稿一覧を取得できます。

```bash
curl -i -b vulnerable.cookies http://localhost:8080/
```

確認することは、攻撃文字列の暗記ではありません。サーバー側で組み立てられた SQL が概念上どうなり、なぜ password 条件がコメント扱いになるのかを書き出してください。

## Step 6: 脆弱版で保存型 XSS を再現する

ブラウザで Alice としてログインし、次の文字列を投稿します。

```html
<script>document.title = 'XSS executed';</script>
```

一覧を再表示したとき、タブのタイトルが `XSS executed` に変われば、保存した本文が HTML ではなくスクリプトとして解釈されています。

この実習では、画面タイトルを変えるだけの無害な確認に留めます。Cookie や入力内容を外部へ送信するコードは使わないでください。

次の 2 点を区別して記録します。

- DB には単なる文字列として保存されている
- HTML 出力時にエスケープしないため、ブラウザが要素として解釈している

## Step 7: 脆弱版で Cookie 属性を観察する

通常のアカウントでログインし、レスポンスヘッダーを確認します。

```bash
curl -i -c vulnerable.cookies \
  -X POST http://localhost:8080/login \
  --data 'email=alice@example.com&password=pass1234'
```

脆弱版の `Set-Cookie` に `HttpOnly`、`Secure`、`SameSite` がないことを確認します。

ブラウザの開発者ツールの Console で次を実行し、`session` が JavaScript から見えることも確認します。

```js
document.cookie
```

この値は記録や共有をしないでください。`HttpOnly` は JavaScript から Cookie を読みにくくする属性であり、XSS 自体を修正する機能ではありません。

## Step 8: 脆弱版で CSRF を再現する

Security Board に Alice としてログインしたブラウザをそのまま使います。次の内容を `csrf-demo.html` として任意の一時ディレクトリに保存してください。`value` には Bob の投稿 ID を入れます。

```html
<!doctype html>
<html lang="ja">
<body>
  <p>攻撃者サイトを模したローカルページ</p>
  <form method="post" action="http://localhost:8080/posts/delete">
    <input type="hidden" name="id" value="BOB_POST_ID">
    <button type="submit">外部ページから送信</button>
  </form>
</body>
</html>
```

その一時ディレクトリで静的サーバーを起動します。

```bash
python3 -m http.server 9090 --bind localhost
```

ブラウザで `http://localhost:9090/csrf-demo.html` を開き、ボタンを押します。投稿が削除された場合、外部ページがセッション Cookie の値を知らなくても、ブラウザが `localhost:8080` 宛ての Cookie を自動送信したことになります。

`localhost:8080` と `localhost:9090` は別オリジンですが同一サイトです。そのため、対策版で `SameSite` を付けた後も、この条件では CSRF トークンによる拒否を個別に確認できます。

## Step 9: 脆弱版で認可漏れを再現する

Alice としてログインしている状態で、一覧に表示された Bob の投稿 ID を確認します。Alice の Cookie を使い、Bob の投稿を削除します。

```bash
curl -i -b vulnerable.cookies \
  -X POST http://localhost:8080/posts/delete \
  --data 'id=BOB_POST_ID'
```

削除に成功した場合、Alice が「認証済み」であることは確認できていますが、その投稿を削除してよいかという「認可」の確認がありません。

CSRF と認可は別の問題です。正しい CSRF トークンを持つ Alice であっても、Bob の投稿削除は拒否されなければなりません。

## Step 10: 脆弱版を保存する

対策を始める前に、観察結果をメモし、脆弱版を Git のコミットとして残します。

```bash
git diff -- 06-security-board
git add 06-security-board
git commit -m 'chapter06: implement vulnerable security board'
```

コミットしたくない場合でも、後で差分を比較できるようにパッチやブランチを残してください。脆弱版をリモートの公開環境へデプロイしてはいけません。

## Step 11: SQL インジェクションとパスワード保存を修正する

ログインは、最初に email だけをプレースホルダ付き SQL で検索し、取得したハッシュと入力パスワードを Go 側で比較する形にします。Step 4 の文字列連結部分を、たとえば次の関数の呼び出しへ置き換えます。

```go
func (app *application) authenticate(
	ctx context.Context, email, password string,
) (Session, error) {
	var current Session
	var passwordHash string
	err := app.db.QueryRowContext(ctx,
		`SELECT id, email, password_hash FROM users WHERE email = $1`,
	email,
	).Scan(&current.UserID, &current.Email, &passwordHash)
	if err != nil {
		return Session{}, err
	}
	if err := bcrypt.CompareHashAndPassword(
		[]byte(passwordHash), []byte(password),
	); err != nil {
		return Session{}, err
	}
	return current, nil
}
```

パスワードハッシュには、パスワード向けに設計された bcrypt、Argon2id などを使用します。この課題では `golang.org/x/crypto/bcrypt` を使えます。

```bash
go get golang.org/x/crypto/bcrypt
```

移行手順は次のとおりです。

1. `users` に `password_hash` 列を一時的に追加する
2. デモ利用者の password から bcrypt ハッシュを生成して保存する
3. 対策版が password_hash だけでログインできることを確認する
4. 平文の `password` 列を削除する
5. `schema.sql` も password_hash だけを持つ最終形へ更新する

ログイン失敗時は、email が存在しない場合と password が違う場合で、外部向けメッセージを分けないでください。

## Step 12: XSS を修正する

Step 4 のサンプルは `html/template` を使っていますが、`template.HTML` へ変換して自動エスケープを意図的に回避しています。対策版では `Post.Body` を通常の `string` に戻し、変換を削除します。

```go
type Post struct {
	ID       int64
	AuthorID int64
	Author   string
	Body     string
}
```

`listPosts` では、DB から取得した body をそのまま代入します。

```go
var post Post
if err := rows.Scan(
	&post.ID, &post.AuthorID, &post.Author, &post.Body,
); err != nil {
	return nil, err
}
posts = append(posts, post)
```

テンプレート側は同じ `{{.Body}}` のままで構いません。型を `string` にしたことで、`html/template` が HTML 文脈に合わせてエスケープします。

```html
{{range .Posts}}
  <article>{{.Body}}</article>
{{end}}
```

`template.HTML(body)` のように安全扱いへ変換すると自動エスケープを回避してしまうため、投稿本文には使いません。入力時に `<` を削除する方法ではなく、HTML、URL、JavaScript など出力先の文脈に合わせて扱うことが重要です。

追加防御として Content Security Policy も有効ですが、出力エスケープの代わりにはなりません。

## Step 13: CSRF を修正する

ログイン時に `crypto/rand` で十分な長さの CSRF トークンを作り、サーバー側セッションへ保存します。Step 4 の `Session` へフィールドを追加します。

```go
type Session struct {
	UserID    int64
	Email     string
	CSRFToken string
}
```

認証成功後、セッションを map に保存する前にトークンを生成します。セッション ID と CSRF トークンに同じ値を使ってはいけません。

```go
csrfToken, err := randomToken()
if err != nil {
	http.Error(w, "failed to create csrf token", http.StatusInternalServerError)
	return
}
current.CSRFToken = csrfToken
```

次のすべての状態変更フォームへ hidden input を追加します。

- 投稿作成
- 投稿削除
- ログアウト

投稿フォームとログアウトフォームでは次を追加します。

```html
<input type="hidden" name="csrf_token" value="{{.Current.CSRFToken}}">
```

`range .Posts` の内側にある削除フォームからルートのデータを参照するときは `$` を付けます。

```html
<input type="hidden" name="csrf_token" value="{{$.Current.CSRFToken}}">
```

各 POST ハンドラーで、フォームのトークンとセッションのトークンを比較します。不足または不一致なら、DB を変更する前に `403 Forbidden` を返します。

```go
func validCSRF(r *http.Request, current Session) bool {
	if err := r.ParseForm(); err != nil {
		return false
	}
	provided := r.FormValue("csrf_token")
	return subtle.ConstantTimeCompare(
		[]byte(provided), []byte(current.CSRFToken),
	) == 1
}
```

`createPost`、`deletePost`、`logout` でセッションを取得した直後に検証します。

```go
if !validCSRF(r, current) {
	http.Error(w, "invalid csrf token", http.StatusForbidden)
	return
}
```

この変更では import に `crypto/subtle` を追加します。

最低限、次を確認します。

- トークンありの正規フォームは成功する
- トークンなしは `403 Forbidden` になる
- 別セッションのトークンは `403 Forbidden` になる
- GET リクエストでは状態が変わらない

## Step 14: Cookie 属性を修正する

対策版のセッション Cookie には次を設定します。

```go
http.SetCookie(w, &http.Cookie{
	Name:     "session",
	Value:    token,
	Path:     "/",
	HttpOnly: true,
	SameSite: http.SameSiteStrictMode,
	Secure:   useHTTPS,
})
```

- `HttpOnly`: JavaScript からセッション Cookie を読み取りにくくする
- `Secure`: HTTPS 通信でだけ Cookie を送る
- `SameSite`: クロスサイトリクエストでの Cookie 送信を制限する

この課題を HTTP の `localhost` で動かす間は、`Secure: true` にするとブラウザが Cookie を送信せず、ログイン状態を維持できないことがあります。ローカル HTTP では `Secure` を設定しない構成値を使い、本番相当の HTTPS では必ず有効にする設計にしてください。

ログアウト時に削除用 Cookie を発行する場合も、`Path`、`Secure`、`SameSite` などを発行時と揃えます。

## Step 15: 投稿削除の認可を修正する

投稿 ID だけでなく、現在の user ID も削除条件に含めます。

```go
result, err := app.db.ExecContext(r.Context(),
	`DELETE FROM posts WHERE id = $1 AND author_id = $2`,
	postID, current.UserID,
)
if err != nil {
	log.Printf("delete post: %v", err)
	http.Error(w, "failed to delete post", http.StatusInternalServerError)
	return
}
affected, err := result.RowsAffected()
if err != nil {
	http.Error(w, "failed to check result", http.StatusInternalServerError)
	return
}
if affected == 0 {
	http.Error(w, "post not found", http.StatusNotFound)
	return
}
```

`RowsAffected` が 0 の場合は削除成功として扱わず、`404 Not Found` または設計に応じた `403 Forbidden` を返します。リソースの存在を不要に知らせないため、実務では 404 に統一することもあります。

画面で他人の投稿の削除ボタンを隠すだけでは対策になりません。HTTP リクエストは画面を経由せず送れるため、サーバー側の SQL または処理で必ず所有者を確認します。

## Step 16: 同じ操作で対策版を再検証する

サーバーを再起動するとメモリ上のセッションが消えるため、対策版へ変更した後はログインし直します。脆弱版と同じ操作を行い、結果を次の表と比較してください。

| 確認項目 | 脆弱版 | 対策版 |
|---|---|---|
| SQL インジェクション文字列でログイン | 成功してしまう | `401 Unauthorized` |
| script を含む投稿を一覧表示 | script が実行される | 文字列として表示される |
| `document.cookie` | session が読める | session は読めない |
| CSRF デモページから削除 | 削除される | `403 Forbidden` |
| Alice が Bob の投稿を削除 | 削除される | 404 または 403 |
| DB の password 列 | 平文がある | 列自体がなく hash だけ |

SQL インジェクションを再実行します。

```bash
curl -i -c secure.cookies \
  -X POST http://localhost:8080/login \
  --data-urlencode "email=' OR 1=1 -- " \
  --data-urlencode 'password=incorrect'
```

Cookie 属性はログインレスポンスの `Set-Cookie` とブラウザの開発者ツールの両方で確認します。

CSRF は Step 8 のページを再利用します。`localhost` の別ポートは同一サイトなので、`403 Forbidden` の理由が CSRF トークン不足であることをサーバーログや処理順でも確認してください。

認可は、正規フォームから得た Alice の CSRF トークンを付けた状態でも Bob の投稿を削除できないことを確認します。CSRF で先に拒否されただけでは、認可を確認したことにはなりません。

## Step 17: 差分をレビューする

```bash
git diff HEAD -- 06-security-board
go test ./...
go vet ./...
```

各修正について、次の 3 点を説明できるようにします。

1. 攻撃者が制御できる入力は何か
2. 脆弱版では、その入力がどのデータや処理へ到達したか
3. 対策版では、どの境界で安全に扱うようになったか

## うまく再現できないとき

### SQL インジェクションが 401 になる

- すでにプレースホルダを使っていないか確認する
- ブラウザの `type="email"` に阻止される場合は、指定の `curl --data-urlencode` を使う
- SQL の `--` の直後に空白があるか確認する
- ログへ password やセッショントークンを出力しない

### XSS の script が実行されない

- 脆弱版で `html/template` が自動エスケープしていないか確認する
- レスポンス HTML を開発者ツールの Elements で確認する
- CSP を先に有効化していないか確認する

### CSRF デモが 401 になる

- Board とデモページの両方を `localhost` で開いているか確認する
- Board へログインしたブラウザと同じブラウザプロファイルを使う
- Board を `127.0.0.1`、デモを `localhost` のように別ホスト名で開いていないか確認する

### `Secure` を付けるとログインできない

HTTP では Secure Cookie が送られないことが原因です。ローカル HTTP 用と HTTPS 用の設定を分け、HTTPS 用設定でだけ `Secure: true` にします。

## 記録すること

学習後に README または別のノートへ記入します。

- 脆弱版の各操作で、予想と実際の結果は一致したか
- SQL の構文と値を分離すると何が変わるか
- XSS の保存地点と実行地点はどこか
- CSRF トークンと SameSite は、それぞれ何を防いだか
- HttpOnly を付けても XSS 修正が必要なのはなぜか
- 認証済みの Alice が Bob の投稿を削除できてはいけないのはなぜか
- 対策版にも残っている、本番運用上の不足は何か

## 発展課題

- `httptest` とテスト DB を使って、脆弱版の再現結果と対策版の拒否結果を自動テストにする
- ログイン失敗回数の制限を追加する
- セッション ID を DB または Redis に保存し、有効期限とローテーションを実装する
- Content Security Policy を追加し、出力エスケープとの役割の違いを整理する
- `SameSite=Lax` と `SameSite=Strict` を cross-site / cross-origin の違いと合わせて検証する
- 管理者による削除と投稿者本人による削除を別の認可ルールとして実装する

## 覚えるべきキーワード

`SQL インジェクション`、`プレースホルダ`、`保存型 XSS`、`html/template`、`CSRF`、`Cookie`、`HttpOnly`、`Secure`、`SameSite`、`認証`、`認可`、`bcrypt`

## この課題のゴール

脆弱版をローカルだけで再現して終わりではありません。同一の入力と操作を対策版にも行い、なぜ結果が変わったかを SQL、HTML、HTTP、Cookie、認証・認可の観点から自分の言葉で説明できれば完了です。
