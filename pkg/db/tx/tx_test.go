package tx

import (
	"context"
	"errors"
	"strings"
	"testing"

	platformpg "github.com/akgate/platform/pkg/db/pg"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type fakeTransactor struct {
	beginTxCalls int
	lastOptions  pgx.TxOptions
	beginErr     error
	tx           *fakeTx
}

func (f *fakeTransactor) BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error) {
	f.beginTxCalls++
	f.lastOptions = txOptions

	if f.beginErr != nil {
		return nil, f.beginErr
	}

	return f.tx, nil
}

type fakeTx struct {
	commitCalls   int
	rollbackCalls int
	commitErr     error
	rollbackErr   error
}

func (f *fakeTx) Begin(ctx context.Context) (pgx.Tx, error) {
	return f, nil
}

func (f *fakeTx) Commit(ctx context.Context) error {
	f.commitCalls++
	return f.commitErr
}

func (f *fakeTx) Rollback(ctx context.Context) error {
	f.rollbackCalls++
	return f.rollbackErr
}

func (f *fakeTx) CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error) {
	return 0, nil
}

func (f *fakeTx) SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults {
	return nil
}

func (f *fakeTx) LargeObjects() pgx.LargeObjects {
	return pgx.LargeObjects{}
}

func (f *fakeTx) Prepare(ctx context.Context, name, sql string) (*pgconn.StatementDescription, error) {
	return nil, nil
}

func (f *fakeTx) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	return pgconn.NewCommandTag(""), nil
}

func (f *fakeTx) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return nil, nil
}

func (f *fakeTx) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return nil
}

func (f *fakeTx) Conn() *pgx.Conn {
	return nil
}

func TestReadCommittedStartsTransactionAndCommits(t *testing.T) {
	t.Parallel()

	tx := &fakeTx{}
	transactor := &fakeTransactor{tx: tx}
	manager := NewTxManager(transactor)

	handlerCalled := false

	err := manager.ReadCommitted(context.Background(), func(ctx context.Context) error {
		handlerCalled = true

		gotTx, ok := ctx.Value(platformpg.TxKey).(pgx.Tx)
		if !ok {
			t.Fatalf("expected transaction in context")
		}

		if gotTx != tx {
			t.Fatalf("expected context transaction to match begun transaction")
		}

		return nil
	})

	if err != nil {
		t.Fatalf("ReadCommitted() error = %v", err)
	}

	if !handlerCalled {
		t.Fatalf("expected handler to be called")
	}

	if transactor.beginTxCalls != 1 {
		t.Fatalf("expected BeginTx to be called once, got %d", transactor.beginTxCalls)
	}

	if transactor.lastOptions.IsoLevel != pgx.ReadCommitted {
		t.Fatalf("expected isolation level %v, got %v", pgx.ReadCommitted, transactor.lastOptions.IsoLevel)
	}

	if tx.commitCalls != 1 {
		t.Fatalf("expected Commit to be called once, got %d", tx.commitCalls)
	}

	if tx.rollbackCalls != 0 {
		t.Fatalf("expected Rollback not to be called, got %d", tx.rollbackCalls)
	}
}

func TestReadCommittedRollsBackOnHandlerError(t *testing.T) {
	t.Parallel()

	tx := &fakeTx{}
	transactor := &fakeTransactor{tx: tx}
	manager := NewTxManager(transactor)

	expectedErr := errors.New("boom")

	err := manager.ReadCommitted(context.Background(), func(ctx context.Context) error {
		return expectedErr
	})

	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "failed execute code inside transaction") {
		t.Fatalf("expected wrapped transaction error, got %v", err)
	}

	if !strings.Contains(err.Error(), expectedErr.Error()) {
		t.Fatalf("expected original error to be preserved, got %v", err)
	}

	if transactor.beginTxCalls != 1 {
		t.Fatalf("expected BeginTx to be called once, got %d", transactor.beginTxCalls)
	}

	if tx.commitCalls != 0 {
		t.Fatalf("expected Commit not to be called, got %d", tx.commitCalls)
	}

	if tx.rollbackCalls != 1 {
		t.Fatalf("expected Rollback to be called once, got %d", tx.rollbackCalls)
	}
}

func TestReadCommittedReusesExistingTransaction(t *testing.T) {
	t.Parallel()

	existingTx := &fakeTx{}
	transactor := &fakeTransactor{tx: &fakeTx{}}
	manager := NewTxManager(transactor)

	ctx := platformpg.MakeContextTx(context.Background(), existingTx)
	handlerCalled := false

	err := manager.ReadCommitted(ctx, func(ctx context.Context) error {
		handlerCalled = true

		gotTx, ok := ctx.Value(platformpg.TxKey).(pgx.Tx)
		if !ok {
			t.Fatalf("expected transaction in context")
		}

		if gotTx != existingTx {
			t.Fatalf("expected existing transaction to be preserved")
		}

		return nil
	})

	if err != nil {
		t.Fatalf("ReadCommitted() error = %v", err)
	}

	if !handlerCalled {
		t.Fatalf("expected handler to be called")
	}

	if transactor.beginTxCalls != 0 {
		t.Fatalf("expected BeginTx not to be called, got %d", transactor.beginTxCalls)
	}

	if existingTx.commitCalls != 0 {
		t.Fatalf("expected existing transaction Commit not to be called, got %d", existingTx.commitCalls)
	}

	if existingTx.rollbackCalls != 0 {
		t.Fatalf("expected existing transaction Rollback not to be called, got %d", existingTx.rollbackCalls)
	}
}
