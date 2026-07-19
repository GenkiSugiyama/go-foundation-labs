package httpapi

import (
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

	want := `{"id":3,"name":"genki"}` + "\n"

	t.Log(rec.Body)
	if rec.Body.String() != want {
		t.Fatalf("got %s, want %s", rec.Body.String(), want)
	}
}
