package main

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
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
	"golang.org/x/crypto/bcrypt"
)

type Session struct {
	UserID    int64
	Email     string
	CSRFToken string
}

type Post struct {
	ID          int64
	AuthorID    int64
	AuthorEmail string
	Body        string
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
	<input type="hidden" name="csrf_token" value="{{.Current.CSRFToken}}">
  </form>

  <h2>新規投稿</h2>
  <form method="post" action="/posts">
    <textarea name="body" required></textarea>
    <button type="submit">post</button>
	<input type="hidden" name="csrf_token" value="{{.Current.CSRFToken}}">
  </form>

  <h2>投稿一覧</h2>
  {{range .Posts}}
    <article>
      <p>#{{.ID}} by {{.AuthorEmail}} / author ID: {{.AuthorID}}</p>
      <div>{{.Body}}</div>
      <form method="post" action="/posts/delete">
        <input type="hidden" name="id" value="{{.ID}}">
        <button type="submit">delete</button>
		<input type="hidden" name="csrf_token" value="{{$.Current.CSRFToken}}">
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
	mux.HandleFunc("/login", app.login)
	mux.HandleFunc("/", app.home)
	mux.HandleFunc("/posts", app.createPost)
	mux.HandleFunc("/posts/delete", app.deletePost)
	mux.HandleFunc("/logout", app.logout)

	server := &http.Server{
		Addr:              "localhost:8080",
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Println("security board listening on :8080")
	log.Fatal(server.ListenAndServe())
}

// TODO: 登録機能がないので、登録時にパスワードをハッシュ化して保存する機能をつくる

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

		// 既存のDBのユーザー情報と照合して、認証に成功したらセッションを作成する
		current, err := app.authenticate(r.Context(), email, password)

		// 認証成功後にCSRFトークンを生成してセッションに保存する
		csrfToken, err := randomToken()
		if err != nil {
			http.Error(w, "failed to create CSRF token", http.StatusInternalServerError)
			return
		}
		current.CSRFToken = csrfToken

		// randomToken()で生成したトークンをセッションIDとして、トークンをkeyとしたMapの値にSession構造体を保存する
		token, err := randomToken()
		if err != nil {
			http.Error(w, "failed to create session", http.StatusInternalServerError)
			return
		}
		app.mu.Lock()
		app.sessions[token] = current
		app.mu.Unlock()

		// Cookieに"session"という名前でsessionIDを保存し、ブラウザに返す
		// ブラウザはこのサイトにリクエストする際に、Cookieに保存されたsessionIDを送信する
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

func (app *application) authenticate(ctx context.Context, email, password string) (Session, error) {
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
		http.Error(w, "failed to render board", http.StatusInternalServerError)
	}
}

func (app *application) listPosts(ctx context.Context) ([]Post, error) {
	rows, err := app.db.QueryContext(ctx, `
		SELECT p.id, p.author_id, u.email, p.body
		FROM posts p
		JOIN users u ON p.author_id = u.id
		ORDER BY p.id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []Post
	for rows.Next() {
		var post Post
		var body string
		if err := rows.Scan(&post.ID, &post.AuthorID, &post.AuthorEmail, &body); err != nil {
			return nil, err
		}
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
	if !validCSRF(r, current) {
		http.Error(w, "invalid csrf token", http.StatusForbidden)
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
	_, current, ok := app.currentSession(r)
	if !ok {
		http.Error(w, "login required", http.StatusUnauthorized)
		return
	}
	// セッションストアに存在するセッションであることが確認できたらセッション情報内のCSRFトークンとリクエストのフォームに含まれるCSRFトークンを比較して、同一であることを確認する
	if !validCSRF(r, current) {
		http.Error(w, "invalid csrf token", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	postID, err := strconv.ParseInt(r.FormValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid post ID", http.StatusBadRequest)
		return
	}

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
	token, current, ok := app.currentSession(r)
	if ok {
		app.mu.Lock()
		delete(app.sessions, token)
		app.mu.Unlock()
	}
	if !validCSRF(r, current) {
		http.Error(w, "invalid csrf token", http.StatusForbidden)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:   "session",
		Path:   "/",
		MaxAge: -1,
	})
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// リクエストからセッション情報を取得してセッションストアに存在するかを確認する
func (app *application) currentSession(r *http.Request) (string, Session, bool) {
	// ブラウザから送られたCookie情報から"session"という名前の情報を取得する
	cookie, err := r.Cookie("session")
	if err != nil {
		return "", Session{}, false
	}
	app.mu.RLock()
	// cokkie.Valueをキーとしたセッション情報が存在するかを確認した結果を返す
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

// 投稿フォームや削除フォーム、ログアウトフォームから送信されるCSRFトークンとサーバーで管理しているセッション情報に保存されているトークンが一致するかを確認する
func validCSRF(r *http.Request, current Session) bool {
	if err := r.ParseForm(); err != nil {
		return false
	}
	provided := r.FormValue("csrf_token")
	return subtle.ConstantTimeCompare([]byte(provided), []byte(current.CSRFToken)) == 1
}
