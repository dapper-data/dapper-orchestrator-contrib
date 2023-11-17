# dapper-orchestrator-contrib

A set of processes and inputs which can be used to provide pipelines as managed by the dapper orchestrator.

## Webhooks

`import github.com/dapper-data/dapper-orchestrator-contrib/webhooks`

Webhooks provides both an Input and a Process; the Input exposes a webhook which listens for well formed [`orchestrator.Event`s](https://pkg.go.dev/github.com/dapper-data/dapper-orchestrator#Event) which it then passes, more or less as it finds it, for processing, whereas the Process will _send_ an [`orchestrator.Event`s](https://pkg.go.dev/github.com/dapper-data/dapper-orchestrator#Event) to a specified webhook.

See the documentation in [`webhooks/`](webhooks/) and the included example code in [`webhooks/example`](webhooks/example) for more.

## Locking Postgres

`import github.com/dapper-data/dapper-orchestrator-contrib/locking-postgres`

Locking Postgres provides a postgres Input which can be used across a cluster of orchestrators; the sample [Postgres Input](https://pkg.go.dev/github.com/dapper-data/dapper-orchestrator#PostgresInput) is unsuitable for this task, as a database operation may be handled multiple times by virtue of each orchestrator in a cluster receiving each notification

See the documentation in [`locking-postgres/`](locking-postgres/) and the example code in [`locking-postgres/example`](locking-postgres/example) for more.

## Container

`import github.com/dapper-data/dapper-orchestrator-contrib/container`

Container provides a way of running containers as Processes, which is handy for a number of things, including running arbitrary code without havinvg to recompile and redeploy orchestrators.
