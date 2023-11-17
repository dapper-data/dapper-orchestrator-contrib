# container

[![GoDoc](https://img.shields.io/badge/pkg.go.dev-doc-blue)](http://pkg.go.dev/github.com/dapper-data/dapper-orchestrator-contrib/container)

## Types

### type [ContainerImageMissingErr](/container_process.go#L31)

`type ContainerImageMissingErr struct{ ... }`

ContainerImageMissingErr is returned when the ExecutionContext passed to
NewContainerProcess doesn't contain tke key "image"

To fix this, ensure that a container image is set

#### func (ContainerImageMissingErr) [Error](/container_process.go#L39)

`func (e ContainerImageMissingErr) Error() string`

Error implements the error interface and returns a contextual message

This error, while simple and (at least on the face of it) an over-engineered
version of fmt.Errorf("container image missing"), is verbosely implemented
so that callers may use errors.Is(err, orchestrator.ContainerImageMissingErr)
to handle error cases better

### type [ContainerNonZeroExit](/container_process.go#L47)

`type ContainerNonZeroExit int64`

ContainerNonZeroExit is returned when the container exists with anything other
than exit code 0

Container logs should shed light on what went wrong

#### func (ContainerNonZeroExit) [Error](/container_process.go#L50)

`func (e ContainerNonZeroExit) Error() string`

Error returns the error message associated with this error

### type [ContainerProcess](/container_process.go#L55)

`type ContainerProcess struct { ... }`

ContainerProcess allows for processes to be run via a container

#### func (ContainerProcess) [ID](/container_process.go#L84)

`func (c ContainerProcess) ID() string`

ID returns a unique ID for a process manager

#### func (ContainerProcess) [Run](/container_process.go#L89)

`func (c ContainerProcess) Run(ctx context.Context, e orchestrator.Event) (ps orchestrator.ProcessStatus, err error)`

Run takes an Event, and passes it to a container to run

---
Readme created from Go doc with [goreadme](https://github.com/posener/goreadme)
