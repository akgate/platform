package tx

import (
	"context"

	"github.com/akgate/platform/pkg/db"
	"github.com/akgate/platform/pkg/db/pg"
	"github.com/cockroachdb/errors"
	"github.com/jackc/pgx/v5"
)

type manager struct {
	db db.Transaction
}

func NewTxManager(db db.Transaction) db.TxManager {
	return &manager{db: db}
}

func (m *manager) transaction(ctx context.Context, opts pgx.TxOptions, fn db.Handler) (err error) {
	tx, ok := ctx.Value(pg.TxKey).(pgx.Tx)
	if ok {
		return fn(ctx)
	}

	tx, err = m.db.BeginTx(ctx, opts)

	if err != nil {
		return err
	}

	ctx = pg.MakeContextTx(ctx, tx)

	defer func() {
		if p := recover(); p != nil {
			err = errors.Errorf("panic recovered: %v", p)
		}

		if err != nil {
			if errRollback := tx.Rollback(ctx); errRollback != nil {
				err = errors.Wrapf(err, "errRollback: %v", errRollback)
			}
			return
		}

		if nil == err {
			err = tx.Commit(ctx)
			if err != nil {
				err = errors.Wrap(err, "tx.Commit failed")
			}
		}
	}()

	if err := fn(ctx); err != nil {
		err = errors.Wrap(err, "failed execute code inside transaction")
	}

	return err
}

func (m *manager) ReadCommited(ctx context.Context, fn db.Handler) error {
	txOpts := pgx.TxOptions{IsoLevel: pgx.ReadCommitted}
	return m.transaction(ctx, txOpts, fn)
}
