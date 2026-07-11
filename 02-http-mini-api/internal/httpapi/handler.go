package httpapi

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
)

type User struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type CreateUserRequest struct {
	Name string `json:"name"`
}

type Handler struct {
	mux *http.ServeMux
}

func New() http.Handler {
	mux := http.NewServeMux()

	h := &Handler{
		mux: mux,
	}
	h.routes()

	return h.mux
}

func (h *Handler) routes() {
	users := []User{
		{ID: 1, Name: "Alice"},
		{ID: 2, Name: "Bob"},
	}

	h.mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("method=%s, path=%s, remote=%s", r.Method, r.URL.Path, r.RemoteAddr)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status": "ok",
		})
	})

	nextID := 3

	h.mux.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {
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

			if req.Name == "" {
				http.Error(w, "name is required", http.StatusBadRequest)
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

	h.mux.HandleFunc("/users/{id}", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("path=%s", r.URL.Path)
		idText := r.PathValue("id")
		id, err := strconv.Atoi(idText)

		if err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		switch r.Method {
		case http.MethodDelete:
			for i, user := range users {
				if user.ID == id {
					users = append(users[:i], users[i+1:]...)
					w.WriteHeader(http.StatusNoContent)
					return
				}
			}
			http.Error(w, "user not found", http.StatusNotFound)
		case http.MethodPut:
			var req CreateUserRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "bad request", http.StatusBadRequest)
				return
			}

			if req.Name == "" {
				http.Error(w, "name is required", http.StatusBadRequest)
				return
			}

			// for で取り出す第二変数は元の値のコピー
			// slice内の要素を更新するにはインデックスでアクセスする必要がある
			for i, user := range users {
				if user.ID == id {
					users[i].Name = req.Name
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(users[i])
					return
				}
			}
			http.Error(w, "user not found", http.StatusNotFound)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}

	})
}
