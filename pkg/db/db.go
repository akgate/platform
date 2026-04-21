package db

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Handler func(ctx context.Context) error

type Client interface {
	DB() DB
	Close() error
}

type TxManager interface {
	ReadCommited(ctx context.Context, fn Handler) error
}

type Query struct {
	Name     string
	QueryRaw string
}

type Transaction interface {
	BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error)
}

type SQLExecer interface {
	QueryExecer
}

type QueryExecer interface {
	ExecContext(ctx context.Context, q Query, args ...interface{}) (pgconn.CommandTag, error)
	QueryContext(ctx context.Context, q Query, args ...interface{}) (pgx.Rows, error)
	QueryRowContext(ctx context.Context, q Query, args ...interface{}) pgx.Row
}

type Pinger interface {
	Ping(ctx context.Context) error
}

type DB interface {
	SQLExecer
	Transaction
	Pinger
	Close()
}
