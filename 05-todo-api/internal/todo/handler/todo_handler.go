package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/GenkiSugiyama/go-foundation-labs/05-todo-api/internal/todo/repository"
)

const dbTimeout = 3 * time.Second

type TodoRepository interface {
	CreateTodo(ctx context.Context, title string, ownerID int64) (int64, error)
	ListTodos(ctx context.Context, ownerID int64) ([]repository.Todo, error)
	UpdateTodoDone(ctx context.Context, id, ownerID int64, done *bool) error
	DeleteTodo(ctx context.Context, id, ownerID int64) error
}

type Handler struct {
	repository TodoRepository
}

func New(repository TodoRepository) *Handler {
	return &Handler{repository: repository}
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

	id, err := h.repository.CreateTodo(ctx, req.Title, req.OwnerID)
	if err != nil {
		writeError(w, http.StatusInternalServerError,
			"failed to create todo")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]int64{"id": id})
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	ownerId, err := positiveInt64(r.URL.Query().Get("owner_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest,
			"owner_id must be a positive integer")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), dbTimeout)
	defer cancel()

	todos, err := h.repository.ListTodos(ctx, ownerId)
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

	err = h.repository.UpdateTodoDone(ctx, id, req.OwnerID, req.Done)
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

	ctx, cancel := context.WithTimeout(r.Context(), dbTimeout)
	defer cancel()

	err = h.repository.DeleteTodo(ctx, id, ownerID)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "todo not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete todo")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func positiveInt64(value string) (int64, error) {
	n, err := strconv.ParseInt(value, 10, 64)
	if err != nil || n <= 0 {
		return 0, errors.New("value must be a positive integer")
	}
	return n, nil
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorResponse{Error: message})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	// json.NewEncoder(w).Encode(value) でhttp.ResponseWriterに引数で受けたvalueをJSONストリームに変換したデータを書き込む
	_ = json.NewEncoder(w).Encode(value)
}
