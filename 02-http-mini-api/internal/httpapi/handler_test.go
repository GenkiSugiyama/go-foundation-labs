package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealth(t *testing.T) {
	handler := New()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("got %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestGetUsers(t *testing.T) {
	handler := New()

	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	rec := httptest.NewRecorder()

	want := `[{"id":1,"name":"Alice"},{"id":2,"name":"Bob"}]` + "\n"

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("got %d, want %d", rec.Code, http.StatusOK)
	}

	if rec.Body.String() != want {
		t.Fatalf("got %q, want %q", rec.Body.String(), want)
	}
}

// jsonレスポンスは文字列比較よりJSONとしてデコードして比較した方が堅牢
// なのでレスポンス用の構造体を定義してレスポンスのバイト文字列を構造体にデコードして比較する
type response struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func TestRegisterUser(t *testing.T) {
	handler := New()

	req := httptest.NewRequest(
		http.MethodPost,
		"/users",
		strings.NewReader(`{"name": "genki"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("got %d, want %d", rec.Code, http.StatusCreated)
	}

	var got response

	// レスポンスのバイト文字列を定義した構造体の型にデコード
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("failed to decode response %v", err)
	}

	want := response{
		ID:   3,
		Name: "genki",
	}

	if got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}
