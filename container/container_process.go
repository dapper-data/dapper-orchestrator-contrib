package container

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/dapper-data/dapper-orchestrator"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

const (
	statusOK = 0

	imageKey = "image"
	envKey   = "env"
	pullKey  = "pull"
)

// ImageMissingErr is returned when the ExecutionContext passed to
// NewContainerProcess doesn't contain tke key "image"
//
// To fix this, ensure that a container image is set
type ImageMissingErr struct{}

// Error implements the error interface and returns a contextual message
//
// This error, while simple and (at least on the face of it) an over-engineered
// version of fmt.Errorf("container image missing"), is verbosely implemented
// so that callers may use errors.Is(err, orchestrator.ContainerImageMissingErr)
// to handle error cases better
func (e ImageMissingErr) Error() string {
	return "container image missing"
}

// NonZeroExit is returned when the container exists with anything other
// than exit code 0
//
// Container logs should shed light on what went wrong
type NonZeroExit int64

// Error returns the error message associated with this error
func (e NonZeroExit) Error() string {
	return fmt.Sprintf("process exited with code %d", int64(e))
}

// Process allows for processes to be run via a container
type Process struct {
	image         string
	additionalEnv []string
	c             *client.Client

	config orchestrator.ProcessConfig
}

// New connects to a container socket, and returns a
// ContainerProcess which can be then used to run jobs
func New(conf orchestrator.ProcessConfig) (c Process, err error) {
	var ok bool

	c.config = conf
	c.image, ok = conf.ExecutionContext[imageKey]
	if !ok {
		err = ImageMissingErr{}

		return
	}

	c.additionalEnv = strings.Split(conf.ExecutionContext[envKey], ",")

	c.c, err = client.NewClientWithOpts(client.FromEnv)

	return
}

// ID returns a unique ID for a process manager
func (c Process) ID() string {
	return c.config.ID()
}

// Run takes an Event, and passes it to a container to run
func (c Process) Run(ctx context.Context, e orchestrator.Event) (ps orchestrator.ProcessStatus, err error) {
	name := c.deriveName()

	ps = orchestrator.ProcessStatus{
		Name:   name,
		Status: orchestrator.ProcessUnstarted,
		Logs:   make([]string, 0),
	}

	pull, ok := c.config.ExecutionContext[pullKey]
	if ok && pull == "true" {
		_, err = c.c.ImagePull(ctx, c.image, types.ImagePullOptions{})
		if err != nil {
			return
		}
	}

	cont, err := c.c.ContainerCreate(
		ctx,
		&container.Config{
			Image:        c.image,
			Env:          c.env(e),
			AttachStdout: false,
			AttachStderr: true,
		},
		&container.HostConfig{
			NetworkMode: container.NetworkMode("host"),
		}, nil, nil, name)
	if err != nil {
		return
	}

	err = c.c.ContainerStart(ctx, cont.ID, types.ContainerStartOptions{})
	if err != nil {
		return
	}

	ps.Status = orchestrator.ProcessUnknown

	wrC, errC := c.c.ContainerWait(ctx, cont.ID, "")
	select {
	case err = <-errC:
		ps.Status = orchestrator.ProcessFail

		return

	case wr := <-wrC:
		var rc io.ReadCloser
		rc, err = c.c.ContainerLogs(ctx, cont.ID, types.ContainerLogsOptions{
			ShowStdout: false,
			ShowStderr: true,
		})
		if err != nil {
			return ps, err
		}

		stdout := new(bytes.Buffer)
		stderr := new(bytes.Buffer)

		_, err = stdcopy.StdCopy(stdout, stderr, rc)
		if err != nil {
			return ps, err
		}

		ps.Logs = strings.Split(stderr.String(), "\n")

		switch wr.StatusCode {
		case statusOK:
			ps.Status = orchestrator.ProcessSuccess

			err = c.c.ContainerRemove(ctx, cont.ID, types.ContainerRemoveOptions{})

		default:
			ps.Status = orchestrator.ProcessFail

			return ps, NonZeroExit(wr.StatusCode)
		}
	}

	return
}

func (c Process) deriveName() string {
	return fmt.Sprintf("%s_%v", c.ID(), time.Now().UnixMicro())
}

func (c Process) env(e orchestrator.Event) (out []string) {
	ev, err := e.JSON()
	if err != nil {
		return c.additionalEnv
	}

	out = []string{
		fmt.Sprintf("PIPELINE_EVENT=%q", base64.StdEncoding.EncodeToString([]byte(ev))),
	}

	return append(out, c.additionalEnv...)
}
