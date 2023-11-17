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

// Postgres is an enhanced version of the sample PostgresInput defined in
// github.com/dapper-data/dapper-orchestrator.
//
// Notable enhancements include:
// 1. Being clusterable, which is to say that that many PostgresInputs can connect to the same database, with a guarantee that each operation will be processed only once
// 2. The New function can be used as an orchestrator.NewInputFunc, which opens up a world of building pipelines on the fly
//
// This PostgresInput is the 'locking postgres input', in the sense that while many replicas of a pipeline
// orchestrator may run at once, when this input connects to database for the first time it uses an internal
// database table to provide a locking mechanism, which ensures only one Input is listening to database
// operations at once.
//
// This Input can be used in place of the sample PostgresInput from the dapper-orchestrator package; it
// has the same configuration and provides the same knobs to twiddle.
type Postgres struct {
	conn          *sqlx.DB
	listener      *pq.Listener
	config        orchestrator.InputConfig
	listenerErrs  chan error
	lockTableName string
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
// URL.
//
// This function will error on:
// 1. Invalid postgres connection strings
// 2. Connection errors to postgres
// 3. Errors creating a listener for database operations
//
// This function, somewhat permissively, has a 500ms timeout to postgres, which should
// cover off all but the most slow networks, while at the same time not slowing execution
// down too much _on_ those slow connections
func New(ic orchestrator.InputConfig) (p Postgres, err error) {
	p.config = ic
	p.lockTableName = p.deriveLockTableName()
	p.listenerErrs = make(chan error)

	url := ic.ConnectionString
	if strings.HasPrefix(url, "postgres://") || strings.HasPrefix(url, "postgresql://") {
		url, err = pq.ParseURL(ic.ConnectionString)
		if err != nil {
			return
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*500)
	defer cancel()

	p.conn, err = sqlx.ConnectContext(ctx, "postgres", url)
	if err != nil {
		return
	}

	p.listener = pq.NewListener(ic.ConnectionString, time.Second, time.Second*10, func(event pq.ListenerEventType, err error) {
		p.listenerErrs <- err
	})

	return
}

// ID returns the ID for this Input
func (p Postgres) ID() string {
	return strings.ReplaceAll(p.config.ID(), "-", "_")
}

// Handle will:
//
// 1. Create triggers and pg_notify procedures so that changes to the database are picked up
// 2. Create an internal table which instances of this PostgresInput will use to ensure operations are handled once
// 3. Parse operations from the database, turning them into `orchestrator.Events`
// From there, the orchestrator its self handles routing of events to different inputs
//
// This function returns errors when garbage comes back from the database, and where database operations
// go away. In such a situation, and where multiple instances of this input run across mutliple replicas
// of an orchestrator, processing should carry on normally- just on another node
func (p Postgres) Handle(ctx context.Context, c chan orchestrator.Event) (err error) {
	err = p.configure()
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
	err = p.getLock(p.lockTableName)
	if err != nil {
		return
	}

	err = p.listener.Listen(p.ID())
	if err != nil {
		return err
	}

	for {
		select {
		case n := <-p.listener.NotificationChannel():
			err = p.handle(c, n)

		case err = <-p.listenerErrs:

		}

		if err != nil {
			return
		}
	}

	return
}

func (p Postgres) handle(c chan orchestrator.Event, n *pq.Notification) (err error) {
	if n == nil {
		return
	}

	input := new(postgresTriggerResult)

	err = json.Unmarshal([]byte(n.Extra), input)
	if err != nil {
		return
	}

	// Ignore changes to lock table
	if input.Table == p.lockTableName {
		return
	}

	c <- orchestrator.Event{
		Location:  input.Table,
		Operation: input.Operation,
		ID:        fmt.Sprintf("%v", input.ID),
		Trigger:   p.ID(),
	}

	return
}

// configure will connect to the database and configure triggers and
// notifies and lock tables and so on so that Handle can do what it needs to
func (p Postgres) configure() (err error) {
	tf := p.triggerFunc()
	tx := p.conn.MustBegin()

	_, err = tx.Exec(tf)
	if err != nil {
		return
	}

	tables := make([]string, 0)
	err = tx.Select(&tables, "SELECT tablename FROM pg_catalog.pg_tables where schemaname = 'public' and tablename != $1;", p.lockTableName)
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
	_, err = tx.Exec(p.createLockTable(p.lockTableName))

	return tx.Commit()
}

func (p Postgres) getLock(tbl string) (err error) {
	tx, err := p.conn.Begin()
	if err != nil {
		return
	}

	_, err = tx.Exec(fmt.Sprintf("SELECT * FROM %s WHERE resource = 1 FOR UPDATE", tbl))

	return
}

func (p Postgres) triggerFunc() string {
	return fmt.Sprintf(`CREATE OR REPLACE FUNCTION process_record_%[1]s() RETURNS TRIGGER as $process_record_%[1]s$
BEGIN
    PERFORM pg_notify('%[1]s', json_build_object('tbl', TG_TABLE_NAME, 'id', COALESCE(NEW.id, 0), 'op', TG_OP)::Text);
    RETURN NEW;
END;
$process_record_%[1]s$ LANGUAGE plpgsql;`, p.ID())
}

func (p Postgres) addTrigger(table string) string {
	return fmt.Sprintf(`CREATE OR REPLACE TRIGGER %[1]s_%[2]s_trigger
AFTER INSERT OR UPDATE OR DELETE ON %[1]s FOR EACH ROW
EXECUTE PROCEDURE process_record_%[2]s();`, table, p.ID())
}

func (p Postgres) deriveLockTableName() string {
	return fmt.Sprintf("lock_postgres_input_%s", p.ID())
}

// createLockTable will create an arbitrary table we can use for locks, thus
// allowing multiple instances of an orchestrator run, without having multiple
// pg_listen streams open- which would mean processing the same event multiple times
func (p Postgres) createLockTable(table string) string {
	return fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (resource int primary key);`, table)
}
