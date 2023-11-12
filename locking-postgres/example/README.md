# Blocking Postgres Example

This directory contains an example of the blocking postgres input.

This is an enhanced version of the [postgres input](https://pkg.go.dev/github.com/dapper-data/dapper-orchestrator#PostgresInput) sample input which comes as part of [Dapper Orchestrator](https://github.com/dapper-data/dapper-orchestrator).

The key difference being that the sample input is not cluster aware; if the orchestrator it's installed on is run over many replicas then you'll have as many inputs to process as you have clusters.

This example contains the pipeline

```mermaid
graph LR;
    db[Sales Database]
    in0[Postgres Input]
    proc0[Webhook Process: Call Notification Endpoint]
    end0[www.example.com]

    db-->in0
    in0-->proc0
    proc0-->end0
```

Where the webhook processor comes from [`github.com/dapper-data/dapper-orchestrator-contrib/webhooks`](https://pkg.go.dev/github.com/dapper-data/dapper-orchestrator-contrib/webhooks), and is used purely for demonstration.

See that package's documentation for configuration.


## Usage

This example uses the dead standard go toolchain:

```bash
$ go build
$ ./example
```

The default postgres DSN is: `postgres://postgres:postgres@localhost:5432/tests?sslmode=disable` but this may be configured with the environment variable:

```bash
$ export DATABASE_DSN='postgres://analytics-user@salesdb.internal:5432/sales'
$ ./example
```

(Tools such as https://webhook.site can be used, for instance, to see the returned payload)

To trigger the input, and thus the process, connect to your database and insert something to a table.
