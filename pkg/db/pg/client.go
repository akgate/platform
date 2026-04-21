package pg

import (
	"context"

	"github.com/akgate/platform/pkg/db"
	"github.com/jackc/pgx/v5/pgxpool"
)

var newPool = func(ctx context.Context, dsn string) (pool, error) {
	return pgxpool.New(ctx, dsn)
}

type pgClient struct {
	db db.DB
}

func NewPgClient(ctx context.Context, dsn string) (db.Client, error) {
	pool, err := newPool(ctx, dsn)
	if err != nil {
		return nil, err
	}

	return &pgClient{
		db: &pg{dbc: pool},
	}, nil
}

func (c *pgClient) DB() db.DB {
	return c.db
}

func (c *pgClient) Close() error {
	if c.db != nil {
		c.db.Close()
	}

	return nil
}
