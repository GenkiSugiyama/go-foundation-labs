package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/GenkiSugiyama/go-foundation-labs/05-todo-api/internal/todo/handler"
	"github.com/GenkiSugiyama/go-foundation-labs/05-todo-api/internal/todo/repository"
	_ "github.com/lib/pq"
)

func main() {
	// sql.Open()で接続設定を作る、定義しただけで接続は確立されてない
	db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// dbへの到達を確認する
	if err := db.PingContext(ctx); err != nil {
		log.Fatal(err)
	}

	log.Println("connected to postgres")

	todoRepository := repository.New(db)
	todoHandler := handler.New(todoRepository)

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

}
