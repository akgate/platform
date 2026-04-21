package pg

import (
	"context"
	"errors"
	"testing"

	"github.com/akgate/platform/pkg/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type fakeDB struct {
	closeCalls int
}

func (f *fakeDB) BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error) {
	return nil, nil
}

func (f *fakeDB) ExecContext(ctx context.Context, q db.Query, args ...interface{}) (pgconn.CommandTag, error) {
	return pgconn.NewCommandTag(""), nil
}

func (f *fakeDB) QueryContext(ctx context.Context, q db.Query, args ...interface{}) (pgx.Rows, error) {
	return nil, nil
}

func (f *fakeDB) QueryRowContext(ctx context.Context, q db.Query, args ...interface{}) pgx.Row {
	return nil
}

func (f *fakeDB) Ping(ctx context.Context) error {
	return nil
}

func (f *fakeDB) Close() {
	f.closeCalls++
}

func TestNewPgClientReturnsClient(t *testing.T) {
	t.Parallel()

	originalNewPool := newPool
	t.Cleanup(func() {
		newPool = originalNewPool
	})

	expectedPool := &fakePool{}
	newPool = func(ctx context.Context, dsn string) (pool, error) {
		if dsn != "postgres://dsn" {
			t.Fatalf("expected dsn to be forwarded, got %q", dsn)
		}

		return expectedPool, nil
	}

	client, err := NewPgClient(context.Background(), "postgres://dsn")
	if err != nil {
		t.Fatalf("NewPgClient() error = %v", err)
	}

	if client == nil {
		t.Fatalf("expected client to be non-nil")
	}

	if client.DB() == nil {
		t.Fatalf("expected DB to be non-nil")
	}
}

func TestNewPgClientReturnsConstructorError(t *testing.T) {
	t.Parallel()

	originalNewPool := newPool
	t.Cleanup(func() {
		newPool = originalNewPool
	})

	expectedErr := errors.New("connect failed")
	newPool = func(ctx context.Context, dsn string) (pool, error) {
		return nil, expectedErr
	}

	client, err := NewPgClient(context.Background(), "postgres://dsn")
	if client != nil {
		t.Fatalf("expected nil client on error")
	}

	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected constructor error to be returned, got %v", err)
	}
}

func TestPgClientDBReturnsUnderlyingDB(t *testing.T) {
	t.Parallel()

	expectedDB := &fakeDB{}
	client := &pgClient{db: expectedDB}

	if gotDB := client.DB(); gotDB != db.DB(expectedDB) {
		t.Fatalf("expected DB() to return underlying db")
	}
}

func TestPgClientCloseDelegatesToUnderlyingDB(t *testing.T) {
	t.Parallel()

	underlying := &fakeDB{}
	client := &pgClient{db: underlying}

	err := client.Close()
	if err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	if underlying.closeCalls != 1 {
		t.Fatalf("expected underlying Close to be called once, got %d", underlying.closeCalls)
	}
}

func TestPgClientCloseHandlesNilDB(t *testing.T) {
	t.Parallel()

	client := &pgClient{}

	if err := client.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}
