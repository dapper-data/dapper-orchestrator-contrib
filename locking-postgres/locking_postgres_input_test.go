package postgres_test

import (
	"context"
	"os"
	"testing"
	"time"

	orchestrator "github.com/dapper-data/dapper-orchestrator"
	"github.com/dapper-data/dapper-orchestrator-contrib/locking-postgres"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func TestNewPostgresInput(t *testing.T) {
	for _, test := range []struct {
		url         string
		expectError bool
	}{
		{"malformed url", true},
		{"postgresql:// malformed url", true},
		{"", true},

		{os.Getenv("TEST_DB_CONN_STRING"), false},
		{os.Getenv("TEST_DB_URL"), false},
	} {
		t.Run(test.url, func(t *testing.T) {
			_, err := postgres.New(orchestrator.InputConfig{
				ConnectionString: test.url,
			})
			if err == nil && test.expectError {
				t.Errorf("expected error, received none")
			} else if err != nil && !test.expectError {
				t.Errorf("unexpected error %#v", err)
			}

		})
	}
}

func TestPostgresInput_Process(t *testing.T) {
	dsn := os.Getenv("TEST_DB_CONN_STRING")

	p, err := postgres.New(orchestrator.InputConfig{
		ConnectionString: dsn,
		Name:             "test_input",
		Type:             "postgres",
		Operations: []orchestrator.Operation{
			orchestrator.OperationCreate,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	conn, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}

	// Ensure there's at least one table in the database so we can generate
	// triggers
	conn.Exec("CREATE TABLE IF NOT EXISTS some_test_table (id numeric);")

	c := make(chan orchestrator.Event)
	go func() {
		time.Sleep(time.Millisecond * 100)
		conn.Exec("SELECT pg_notify('test_input', json_build_object('tbl', 'test', 'id', '1', 'op', 'INSERT')::Text);")
		conn.Exec("SELECT pg_notify('test_input', json_build_object('tbl', 'test', 'id', '1', 'op', 'UPDATE')::Text);")
		conn.Exec("SELECT pg_notify('test_input', json_build_object('tbl', 'test', 'id', '1', 'op', 'DELETE')::Text);")

		conn.Exec("SELECT pg_notify('test_input', json_build_object('tbl', 'test', 'id', '1', 'op', 'UPDATE')::Text);")

		conn.Exec("SELECT pg_notify('test_input', json_build_object('tbl', 'test', 'id', '1', 'op', 'INSERT')::Text);")

		// Changes to lock tables should be ignored
		conn.Exec("SELECT pg_notify('test_input', json_build_object('tbl', 'lock_postgres_input_test_input', 'id', '1', 'op', 'INSERT')::Text);")

		// Invalid json should be ignored
		conn.Exec("SELECT pg_notify('test_input', 'some bollocks');")

	}()

	count := 0
	go func() {
		for range c {
			count++
		}
	}()

	err = p.Handle(context.Background(), c)
	if err == nil {
		t.Errorf("expected error, received none")
	}

	if count != 5 {
		t.Errorf("expected 5 notification, received %d", count)
	}
}
