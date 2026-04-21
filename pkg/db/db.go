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

type TransactionManager interface {
	ReadCommitted(ctx context.Context, fn Handler) error
}

type Query struct {
	Name     string
	QueryRaw string
}

type Transactor interface {
	BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error)
}

type ExecerContext interface {
	ExecContext(ctx context.Context, q Query, args ...interface{}) (pgconn.CommandTag, error)
}

type QueryerContext interface {
	QueryContext(ctx context.Context, q Query, args ...interface{}) (pgx.Rows, error)
	QueryRowContext(ctx context.Context, q Query, args ...interface{}) pgx.Row
}

type ExtContext interface {
	ExecerContext
	QueryerContext
}

type Pinger interface {
	Ping(ctx context.Context) error
}

type DB interface {
	ExtContext
	Transactor
	Pinger
	Close()
}
