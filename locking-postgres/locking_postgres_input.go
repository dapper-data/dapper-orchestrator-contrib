package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	orchestrator "github.com/dapper-data/dapper-orchestrator"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// PostgresInput represents a sample postgres input source
//
// This source will:
//
//  1. Create a function which notifies a channel with a json payload representing an operation
//  2. Add a trigger to every table in a database to call that function on Creat, Update, and Deletes
//  3. Listen to the channel created in step 1
//
// The operations passed by the database can then be passed to a Process
type PostgresInput struct {
	conn     *sqlx.DB
	listener *pq.Listener
	config   orchestrator.InputConfig
}

type postgresTriggerResult struct {
	Table     string                 `json:"tbl"`
	ID        any                    `json:"id"`
	Operation orchestrator.Operation `json:"op"`
}

// New accepts an InputConfig and returns a PostgresInput,
// which implements the orchestrator.Input interface
//
// The InputConfig.ConnectionString argument can be a DSN, or a postgres
// URL
func New(ic orchestrator.InputConfig) (p PostgresInput, err error) {
	p.config = ic

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*500)
	defer cancel()

	url := ic.ConnectionString
	if strings.HasPrefix(url, "postgres://") || strings.HasPrefix(url, "postgresql://") {
		url, err = pq.ParseURL(ic.ConnectionString)
		if err != nil {
			return
		}
	}

	p.conn, err = sqlx.ConnectContext(ctx, "postgres", url)
	if err != nil {
		return
	}

	p.listener = pq.NewListener(ic.ConnectionString, time.Second, time.Second*10, func(event pq.ListenerEventType, err error) {
		if err != nil {
			panic(err)
		}
	})

	return
}

// ID returns the ID for this Input
func (p PostgresInput) ID() string {
	return strings.ReplaceAll(p.config.ID(), "-", "_")
}

// Handle will configure a database for notification, and then listen to those
// notifications
func (p PostgresInput) Handle(ctx context.Context, c chan orchestrator.Event) (err error) {
	err = p.createTriggers()
	if err != nil {
		return
	}

	// Try to get a lock against our lock table
	//
	// This operation will block until we get the lock, meaning that
	// the later call to p.listener.Listen() doesn't trigger and we don't
	// end up with duplicate events.
	//
	// In theory, at least
	lockTableName := p.lockTableName()
	err = p.getLock(lockTableName)
	if err != nil {
		return
	}

	err = p.listener.Listen(p.ID())
	if err != nil {
		return err
	}

	for n := range p.listener.NotificationChannel() {
		input := new(postgresTriggerResult)

		err = json.Unmarshal([]byte(n.Extra), input)
		if err != nil {
			return
		}

		// Ignore changes to lock table
		if input.Table == lockTableName {
			continue
		}

		c <- orchestrator.Event{
			Location:  input.Table,
			Operation: input.Operation,
			ID:        fmt.Sprintf("%v", input.ID),
			Trigger:   p.ID(),
		}
	}

	return
}

// createTriggers will connect to the database and configure triggers and
// notifies ahead of processing
func (p PostgresInput) createTriggers() (err error) {
	tf := p.triggerFunc()
	tx := p.conn.MustBegin()

	_, err = tx.Exec(tf)
	if err != nil {
		return
	}

	lockTbl := p.lockTableName()

	tables := make([]string, 0)
	err = tx.Select(&tables, "SELECT tablename FROM pg_catalog.pg_tables where schemaname = 'public' and tablename != $1;", lockTbl)
	if err != nil {
		return
	}

	for _, table := range tables {
		_, err = tx.Exec(p.addTrigger(table))
		if err != nil {
			return
		}
	}

	// create lock table
	_, err = tx.Exec(p.createLockTable(lockTbl))

	return tx.Commit()
}

func (p PostgresInput) getLock(tbl string) (err error) {
	tx, err := p.conn.Begin()
	if err != nil {
		return
	}

	_, err = tx.Exec(fmt.Sprintf("SELECT * FROM %s WHERE resource = 1 FOR UPDATE", tbl))

	return
}

func (p PostgresInput) triggerFunc() string {
	return fmt.Sprintf(`CREATE OR REPLACE FUNCTION process_record_%[1]s() RETURNS TRIGGER as $process_record_%[1]s$
BEGIN
    PERFORM pg_notify('%[1]s', json_build_object('tbl', TG_TABLE_NAME, 'id', COALESCE(NEW.id, 0), 'op', TG_OP)::Text);
    RETURN NEW;
END;
$process_record_%[1]s$ LANGUAGE plpgsql;`, p.ID())
}

func (p PostgresInput) addTrigger(table string) string {
	return fmt.Sprintf(`CREATE OR REPLACE TRIGGER %[1]s_%[2]s_trigger
AFTER INSERT OR UPDATE OR DELETE ON %[1]s FOR EACH ROW
EXECUTE PROCEDURE process_record_%[2]s();`, table, p.ID())
}

func (p PostgresInput) lockTableName() string {
	return fmt.Sprintf("lock_postgres_input_%s", p.ID())
}

// createLockTable will create an arbitrary table we can use for locks, thus
// allowing multiple instances of an orchestrator run, without having multiple
// pg_listen streams open- which would mean processing the same event multiple times
func (p PostgresInput) createLockTable(table string) string {
	return fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (resource int primary key);`, table)
}
