package pg

import (
	"context"

	"github.com/akgate/platform/pkg/db"
	"github.com/jackc/pgx/v5/pgxpool"
)

type pgClient struct {
	db db.DB
}

func NewPgClient(ctx context.Context, dsn string) (db.Client, error) {
	db, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}

	return &pgClient{
		db: &pg{dbc: db},
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
