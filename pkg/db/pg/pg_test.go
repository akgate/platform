package pg

import (
	"context"
	"errors"
	"testing"

	"github.com/akgate/platform/pkg/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type fakePool struct {
	beginTxCalls int
	beginTxErr   error
	beginTxOpts  pgx.TxOptions
	tx           pgx.Tx

	execCalls int
	execSQL   string
	execArgs  []any
	execErr   error
	execTag   pgconn.CommandTag

	queryCalls int
	querySQL   string
	queryArgs  []any
	queryErr   error
	queryRows  pgx.Rows

	queryRowCalls int
	queryRowSQL   string
	queryRowArgs  []any
	queryRow      pgx.Row

	pingCalls int
	pingErr   error

	closeCalls int
}

func (f *fakePool) BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error) {
	f.beginTxCalls++
	f.beginTxOpts = txOptions
	if f.beginTxErr != nil {
		return nil, f.beginTxErr
	}

	return f.tx, nil
}

func (f *fakePool) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	f.execCalls++
	f.execSQL = sql
	f.execArgs = arguments
	return f.execTag, f.execErr
}

func (f *fakePool) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	f.queryCalls++
	f.querySQL = sql
	f.queryArgs = args
	return f.queryRows, f.queryErr
}

func (f *fakePool) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	f.queryRowCalls++
	f.queryRowSQL = sql
	f.queryRowArgs = args
	return f.queryRow
}

func (f *fakePool) Ping(ctx context.Context) error {
	f.pingCalls++
	return f.pingErr
}

func (f *fakePool) Close() {
	f.closeCalls++
}

type fakeTx struct {
	execCalls     int
	execSQL       string
	execArgs      []any
	execTag       pgconn.CommandTag
	execErr       error
	queryCalls    int
	querySQL      string
	queryArgs     []any
	queryRows     pgx.Rows
	queryErr      error
	queryRowCalls int
	queryRowSQL   string
	queryRowArgs  []any
	queryRow      pgx.Row
}

func (f *fakeTx) Begin(ctx context.Context) (pgx.Tx, error) { return f, nil }
func (f *fakeTx) Commit(ctx context.Context) error          { return nil }
func (f *fakeTx) Rollback(ctx context.Context) error        { return nil }
func (f *fakeTx) CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error) {
	return 0, nil
}
func (f *fakeTx) SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults { return nil }
func (f *fakeTx) LargeObjects() pgx.LargeObjects                               { return pgx.LargeObjects{} }
func (f *fakeTx) Prepare(ctx context.Context, name, sql string) (*pgconn.StatementDescription, error) {
	return nil, nil
}
func (f *fakeTx) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	f.execCalls++
	f.execSQL = sql
	f.execArgs = arguments
	return f.execTag, f.execErr
}
func (f *fakeTx) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	f.queryCalls++
	f.querySQL = sql
	f.queryArgs = args
	return f.queryRows, f.queryErr
}
func (f *fakeTx) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	f.queryRowCalls++
	f.queryRowSQL = sql
	f.queryRowArgs = args
	return f.queryRow
}
func (f *fakeTx) Conn() *pgx.Conn { return nil }

type fakeRows struct{}

func (f *fakeRows) Close()                                       {}
func (f *fakeRows) Err() error                                   { return nil }
func (f *fakeRows) CommandTag() pgconn.CommandTag                { return pgconn.NewCommandTag("") }
func (f *fakeRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (f *fakeRows) Next() bool                                   { return false }
func (f *fakeRows) Scan(dest ...any) error                       { return nil }
func (f *fakeRows) Values() ([]any, error)                       { return nil, nil }
func (f *fakeRows) RawValues() [][]byte                          { return nil }
func (f *fakeRows) Conn() *pgx.Conn                              { return nil }

type fakeRow struct {
	scanCalls int
	scanErr   error
}

func (f *fakeRow) Scan(dest ...any) error {
	f.scanCalls++
	return f.scanErr
}

func TestNewDBReturnsUsableDB(t *testing.T) {
	t.Parallel()

	if NewDB(nil) == nil {
		t.Fatalf("expected NewDB to return non-nil db")
	}
}

func TestPgBeginTxDelegatesToPool(t *testing.T) {
	t.Parallel()

	expectedTx := &fakeTx{}
	pool := &fakePool{tx: expectedTx}
	driver := &pg{dbc: pool}
	opts := pgx.TxOptions{IsoLevel: pgx.ReadCommitted}

	gotTx, err := driver.BeginTx(context.Background(), opts)
	if err != nil {
		t.Fatalf("BeginTx() error = %v", err)
	}

	if gotTx != expectedTx {
		t.Fatalf("expected returned tx to match pool tx")
	}

	if pool.beginTxCalls != 1 {
		t.Fatalf("expected BeginTx to be called once, got %d", pool.beginTxCalls)
	}

	if pool.beginTxOpts.IsoLevel != opts.IsoLevel {
		t.Fatalf("expected tx options to be forwarded")
	}
}

func TestPgExecContextUsesPoolWithoutTransaction(t *testing.T) {
	t.Parallel()

	expectedTag := pgconn.NewCommandTag("INSERT 0 1")
	pool := &fakePool{execTag: expectedTag}
	driver := &pg{dbc: pool}

	gotTag, err := driver.ExecContext(context.Background(), db.Query{QueryRaw: "insert into photo values ($1)"}, 42)
	if err != nil {
		t.Fatalf("ExecContext() error = %v", err)
	}

	if gotTag != expectedTag {
		t.Fatalf("expected command tag to come from pool")
	}

	if pool.execCalls != 1 {
		t.Fatalf("expected pool Exec to be called once, got %d", pool.execCalls)
	}

	if pool.execSQL != "insert into photo values ($1)" {
		t.Fatalf("expected SQL to be forwarded, got %q", pool.execSQL)
	}

	if len(pool.execArgs) != 1 || pool.execArgs[0] != 42 {
		t.Fatalf("expected args to be forwarded, got %#v", pool.execArgs)
	}
}

func TestPgExecContextUsesTransactionFromContext(t *testing.T) {
	t.Parallel()

	expectedTag := pgconn.NewCommandTag("UPDATE 1")
	tx := &fakeTx{execTag: expectedTag}
	pool := &fakePool{}
	driver := &pg{dbc: pool}
	ctx := MakeContextTx(context.Background(), tx)

	gotTag, err := driver.ExecContext(ctx, db.Query{QueryRaw: "update photo set x = $1"}, 7)
	if err != nil {
		t.Fatalf("ExecContext() error = %v", err)
	}

	if gotTag != expectedTag {
		t.Fatalf("expected command tag to come from tx")
	}

	if tx.execCalls != 1 {
		t.Fatalf("expected tx Exec to be called once, got %d", tx.execCalls)
	}

	if pool.execCalls != 0 {
		t.Fatalf("expected pool Exec not to be called, got %d", pool.execCalls)
	}
}

func TestPgQueryContextUsesPoolWithoutTransaction(t *testing.T) {
	t.Parallel()

	expectedRows := &fakeRows{}
	pool := &fakePool{queryRows: expectedRows}
	driver := &pg{dbc: pool}

	gotRows, err := driver.QueryContext(context.Background(), db.Query{QueryRaw: "select * from photo where id = $1"}, 1)
	if err != nil {
		t.Fatalf("QueryContext() error = %v", err)
	}

	if gotRows != expectedRows {
		t.Fatalf("expected rows to come from pool")
	}

	if pool.queryCalls != 1 {
		t.Fatalf("expected pool Query to be called once, got %d", pool.queryCalls)
	}
}

func TestPgQueryContextUsesTransactionFromContext(t *testing.T) {
	t.Parallel()

	expectedRows := &fakeRows{}
	tx := &fakeTx{queryRows: expectedRows}
	pool := &fakePool{}
	driver := &pg{dbc: pool}
	ctx := MakeContextTx(context.Background(), tx)

	gotRows, err := driver.QueryContext(ctx, db.Query{QueryRaw: "select * from photo"}, "arg")
	if err != nil {
		t.Fatalf("QueryContext() error = %v", err)
	}

	if gotRows != expectedRows {
		t.Fatalf("expected rows to come from tx")
	}

	if tx.queryCalls != 1 {
		t.Fatalf("expected tx Query to be called once, got %d", tx.queryCalls)
	}

	if pool.queryCalls != 0 {
		t.Fatalf("expected pool Query not to be called, got %d", pool.queryCalls)
	}
}

func TestPgQueryRowContextUsesPoolWithoutTransaction(t *testing.T) {
	t.Parallel()

	expectedRow := &fakeRow{}
	pool := &fakePool{queryRow: expectedRow}
	driver := &pg{dbc: pool}

	gotRow := driver.QueryRowContext(context.Background(), db.Query{QueryRaw: "select 1"})
	if gotRow != expectedRow {
		t.Fatalf("expected row to come from pool")
	}

	if pool.queryRowCalls != 1 {
		t.Fatalf("expected pool QueryRow to be called once, got %d", pool.queryRowCalls)
	}
}

func TestPgQueryRowContextUsesTransactionFromContext(t *testing.T) {
	t.Parallel()

	expectedRow := &fakeRow{}
	tx := &fakeTx{queryRow: expectedRow}
	pool := &fakePool{}
	driver := &pg{dbc: pool}
	ctx := MakeContextTx(context.Background(), tx)

	gotRow := driver.QueryRowContext(ctx, db.Query{QueryRaw: "select 1"})
	if gotRow != expectedRow {
		t.Fatalf("expected row to come from tx")
	}

	if tx.queryRowCalls != 1 {
		t.Fatalf("expected tx QueryRow to be called once, got %d", tx.queryRowCalls)
	}

	if pool.queryRowCalls != 0 {
		t.Fatalf("expected pool QueryRow not to be called, got %d", pool.queryRowCalls)
	}
}

func TestPgPingDelegatesToPool(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("ping failed")
	pool := &fakePool{pingErr: expectedErr}
	driver := &pg{dbc: pool}

	err := driver.Ping(context.Background())
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected ping error to be returned, got %v", err)
	}

	if pool.pingCalls != 1 {
		t.Fatalf("expected Ping to be called once, got %d", pool.pingCalls)
	}
}

func TestPgCloseDelegatesToPool(t *testing.T) {
	t.Parallel()

	pool := &fakePool{}
	driver := &pg{dbc: pool}

	driver.Close()

	if pool.closeCalls != 1 {
		t.Fatalf("expected Close to be called once, got %d", pool.closeCalls)
	}
}

func TestMakeContextTxStoresTransaction(t *testing.T) {
	t.Parallel()

	tx := &fakeTx{}
	ctx := MakeContextTx(context.Background(), tx)

	gotTx, ok := ctx.Value(TxKey).(pgx.Tx)
	if !ok {
		t.Fatalf("expected tx in context")
	}

	if gotTx != tx {
		t.Fatalf("expected stored tx to match original")
	}
}
