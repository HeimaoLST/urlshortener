package db

import (
	"context"
	"database/sql"
	"fmt"
)

type Store struct {
	*Queries
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{
		db:      db,
		Queries: New(db),
	}
}
func (store *Store) execTx(ctx context.Context, fn func(*Queries) error) error {
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	q := New(tx)
	err = fn(q)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("tx err: %v, rb err: %v", err, rbErr)
		}
		return err
	}

	return tx.Commit()
}

func (store *Store) AddUrlClicks(ctx context.Context, clicks map[int64]int) error {
	// 使用 execTx 来确保所有更新都在一个事务中完成
	return store.execTx(ctx, func(q *Queries) error {
		// 遍历传入的 map
		for id, count := range clicks {
			if count == 0 {
				continue
			}
			// 为 map 中的每一个条目执行 AddUrlClicks 查询
			err := q.AddUrlClick(ctx, AddUrlClickParams{
				ID:       id,
				AddCount: int32(count),
			})
			if err != nil {

				return err
			}
		}
		return nil
	})
}
