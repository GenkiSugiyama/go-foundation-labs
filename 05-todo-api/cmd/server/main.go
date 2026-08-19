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
}
