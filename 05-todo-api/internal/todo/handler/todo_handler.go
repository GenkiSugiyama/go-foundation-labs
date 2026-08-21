package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
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

type listTodosRequest struct {
	OwnerID int64 `json:"owner_id"`
}

type updateTodoRequest struct {
	Done    *bool `json:"done"`
	OwnerID int64 `json:"owner_id"`
}

type deleteTodoRequest struct {
	ID      int64 `json:"id"`
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
		writeError(w, http.StatusInternalServerError,
			"failed to create todo")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]int64{"id": id})
}

func (h *handler) List(w http.ResponseWriter, r *http.Request) {
	ownerId, err := positiveInt64(r.URL.Query().Get("owner_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest,
			"owner_id must be a positive integer")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), dbTimeout)
	defer cancel()

	todos, err := repository.ListTodos(ctx, h.db, ownerId)
	if err != nil {
		writeError(w, http.StatusInternalServerError,
			"failed to list todos")
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
		writeError(w, http.StatusBadRequest,
			"id must bu a positive integer")
		return
	}

	var req updateTodoRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest,
			"invalide JSON body")
		return
	}

	if req.Done == nil || req.OwnerID <= 0 {
		writeError(w, http.StatusBadRequest,
			"done and positive owner_id are required")
	}

	ctx, cancel := context.WithTimeout(r.Context(), dbTimeout)
	defer cancel()

	err = repository.UpdateTodoDone(ctx, h.db, id, req.OwnerID, req.Done)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound,
			"todo not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError,
			"failed to update todo")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := positiveInt64(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest,
			"id must bu a positive integer")
		return
	}

	ownerID, err := positiveInt64(r.URL.Query().Get("owner_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest,
			"owner_id must bu a positive integer")
		return
	}
}
