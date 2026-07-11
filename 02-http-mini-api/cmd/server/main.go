package main

import (
	"log"
	"net/http"

	"github.com/GenkiSugiyama/go-foundation-labs/02-http-mini-api/internal/httpapi"
)

func main() {
	mux := httpapi.New()

	log.Println("http mini api listening on :8080")

	// http.ListenAndServe()は内部でnet.Listen()でTCPを指定しリスナーを作成し、リクエストが来るとハンドラに処理を渡す
	log.Fatal(http.ListenAndServe(":8080", mux))
}
