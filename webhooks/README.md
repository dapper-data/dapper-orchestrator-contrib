# webhooks

[![GoDoc](https://img.shields.io/badge/pkg.go.dev-doc-blue)](http://pkg.go.dev/github.com/dapper-data/dapper-orchestrator-contrib/webhooks)

## Types

### type [BadStatusErr](/webhook_process.go#L49)

`type BadStatusErr struct { ... }`

BadStatusErr is returned when a call returns a non-2xx response

#### func (BadStatusErr) [Error](/webhook_process.go#L52)

`func (e BadStatusErr) Error() string`

Error returns the error text for this error

### type [Input](/webhook_input.go#L19)

`type Input struct { ... }`

Input implements the orchestrator.Input interface

It listens to a user specified path (as specified in the InputConfig.ConnectionString
argument to NewWebhookInput), and expects to receive a valid orchestrator.Event as
JSON

For custom input payloads, simply copy the code in github.com/dapper-data/dapper-orchestrator-contrib/webhooks
and replace the bits you want to replace

#### func (*Input) [Handle](/webhook_input.go#L44)

`func (w *Input) Handle(ctx context.Context, c chan orchestrator.Event) (err error)`

Handle implements the Handle function of the orchestrator.Input interface

It registers a handler function against the default server provided by the
net/http package and listens for Events which are then passed down Event chan `c`

This function exits immediately

#### func (Input) [ID](/webhook_input.go#L78)

`func (w Input) ID() string`

ID returns an ID for this input

### type [MissingWebhookURLErr](/webhook_process.go#L41)

`type MissingWebhookURLErr struct{ ... }`

MissingWebhookURLErr is returned when an ExecutionContext does not
contain a url to hit

#### func (MissingWebhookURLErr) [Error](/webhook_process.go#L44)

`func (e MissingWebhookURLErr) Error() string`

Error returns the error text for this error

### type [Process](/webhook_process.go#L64)

`type Process struct { ... }`

Process implements the orchestrator.Process interface

When triggered, it sends a the orchestrator.Event is was called with
as JSON to the endpoint the WebhookProcess was instantiated with via the arguments to
orchestrator.ProcessConfig.ExecutionContext, passed as argument 'pc'

For custom process endpoints, simply copy the code in github.com/dapper-data/dapper-orchestrator-contrib/webhooks
and replace the bits you want to replace

#### func (Process) [ID](/webhook_process.go#L140)

`func (w Process) ID() string`

ID returns an ID for this process

#### func (Process) [Run](/webhook_process.go#L101)

`func (w Process) Run(ctx context.Context, e orchestrator.Event) (ps orchestrator.ProcessStatus, err error)`

Run will, given an orchestrator.Event, encode that Event to JSON and send it
to the endpoint the WebhookProcess was configured with via the function NewWebhookProcess

A non-2xx response will return a webhooks.BadStatusErr which describes status
returned.

Additionally, the logs field of the returned orchestrator.ProcessStatus will contain
errors, warnings, and response metadata (which can be ignored if err == nil)

## Sub Packages

* [example](./example)

---
Readme created from Go doc with [goreadme](https://github.com/posener/goreadme)
