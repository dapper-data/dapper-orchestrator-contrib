# postgres

[![GoDoc](https://img.shields.io/badge/pkg.go.dev-doc-blue)](http://pkg.go.dev/github.com/dapper-data/dapper-orchestrator-contrib/locking-postgres)

## Types

### type [PostgresInput](/locking_postgres_input.go#L29)

`type PostgresInput struct { ... }`

PostgresInput is an enhanced version of the sample PostgresInput defined in
github.com/dapper-data/dapper-orchestrator.

Notable enhancements include:
1. Being clusterable, which is to say that that many PostgresInputs can connect to the same database, with a guarantee that each operation will be processed only once
2. The New function can be used as an orchestrator.NewInputFunc, which opens up a world of building pipelines on the fly

This PostgresInput is the 'locking postgres input', in the sense that while many replicas of a pipeline
orchestrator may run at once, when this input connects to database for the first time it uses an internal
database table to provide a locking mechanism, which ensures only one Input is listening to database
operations at once.

This Input can be used in place of the sample PostgresInput from the dapper-orchestrator package; it
has the same configuration and provides the same knobs to twiddle.

#### func (PostgresInput) [Handle](/locking_postgres_input.go#L100)

`func (p PostgresInput) Handle(ctx context.Context, c chan orchestrator.Event) (err error)`

Handle will:

1. Create triggers and pg_notify procedures so that changes to the database are picked up
2. Create an internal table which instances of this PostgresInput will use to ensure operations are handled once
3. Parse operations from the database, turning them into `orchestrator.Events`
From there, the orchestrator its self handles routing of events to different inputs

This function returns errors when garbage comes back from the database, and where database operations
go away. In such a situation, and where multiple instances of this input run across mutliple replicas
of an orchestrator, processing should carry on normally- just on another node

#### func (PostgresInput) [ID](/locking_postgres_input.go#L86)

`func (p PostgresInput) ID() string`

ID returns the ID for this Input

## Sub Packages

* [example](./example)

---
Readme created from Go doc with [goreadme](https://github.com/posener/goreadme)
