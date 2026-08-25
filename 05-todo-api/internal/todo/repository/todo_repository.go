package repository

import (
	"context"
	"database/sql"
	"strconv"
	"time"
)

type Todo struct {
	ID        int64
	Title     string
	Done      bool
	OwnerID   int64
	CreatedAt time.Time
}

type TodoRepository struct {
	db *sql.DB
}

func New(db *sql.DB) *TodoRepository {
	return &TodoRepository{db: db}
}

func (r *TodoRepository) CreateTodo(ctx context.Context, title string, ownerID int64) (int64, error) {
	var id int64
	// QueryRowContextは、1行だけ返すクエリを実行する
	// 指定のINSERT文を実行し、RETURNING句で返されたidをScan()で取得する
	err := r.db.QueryRowContext(ctx,
		`INSERT INTO todos (title, owner_id)
		 VALUES($1, $2)
		 RETURNING id`,
		title, ownerID,
	).Scan(&id)
	return id, err
}

func (r *TodoRepository) ListTodos(ctx context.Context, ownerID int64) ([]Todo, error) {
	// QueryContext はSELECTのようなクエリを実行し取得できた複数の行データを返す
	// rows, err := r.db.QueryContext(ctx,
	// 	`SELECT id, title, done, owner_id, created_at
	// 	 FROM todos
	// 	 WHERE owner_id = $1
	// 	 ORDER BY id`,
	// 	ownerID,
	// )
	query := "SELECT id, title, done, owner_id, created_at FROM todos WHERE owner_id = " + strconv.FormatInt(ownerID, 10) + " ORDER BY id"
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	// sql.Rowsはクエリ結果のストリームなので、rows.Close()を呼び出してクローズする必要がある
	defer rows.Close()

	var todos []Todo
	// ストリームをNext()で1行ずつ読み込む
	// rows.Next()の中でrows.Scan()を呼び出して、1行分のデータを構造体に格納する
	for rows.Next() {
		var t Todo
		if err := rows.Scan(&t.ID, &t.Title, &t.Done, &t.OwnerID, &t.CreatedAt); err != nil {
			return nil, err
		}
		todos = append(todos, t)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return todos, err
}

func (r *TodoRepository) UpdateTodoDone(ctx context.Context, id, ownerID int64, done *bool) error {
	// ExecContextは、SQLの結果行を返さず、実行結果の概要をsql.Resultとして返す。
	result, err := r.db.ExecContext(ctx,
		`UPDATE todos
		 SET done = $1
		 WHERE id = $2 AND owner_id = $3`,
		done, id, ownerID,
	)
	if err != nil {
		return err
	}

	// sql.Result.RowsAffected()でupdate, insert, delete文の影響を受けた行数を取得する
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func (r *TodoRepository) DeleteTodo(ctx context.Context, id, ownerID int64) error {
	result, err := r.db.ExecContext(ctx,
		`DELETE FROM todos
		 WHERE id = $1 AND owner_id = $2`,
		id, ownerID,
	)
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return sql.ErrNoRows
	}

	return nil
}
