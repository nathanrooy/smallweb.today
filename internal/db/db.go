package db

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq"
)

type DB struct {
	conn *sql.DB
}

func Open(dsn string) (*DB, error) {
	// open the connection pool
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal("failed to connect database", err)
	}

	// set some basic constraints
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(time.Minute * 10)

	// verify the connection is actually alive
	if err := db.Ping(); err != nil {
		return nil, err
	}

	return &DB{conn: db}, nil
}

func (d *DB) Close() error {
	return d.conn.Close()
}

func (d *DB) SQL() *sql.DB {
	return d.conn
}

func (d *DB) WithTx(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := d.conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	if err := ctx.Err(); err != nil {
		tx.Rollback()
		return err
	}

	if err := fn(tx); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (d *DB) RefreshViews(ctx context.Context) error {
	_, err := d.conn.ExecContext(ctx, `REFRESH MATERIALIZED VIEW feeds_filtered;`)
	if err != nil {
		return fmt.Errorf("failed to refresh feeds_filtered: %w", err)
	}
	return nil
}
