package container_test

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/dapper-data/dapper-orchestrator"
	"github.com/dapper-data/dapper-orchestrator-contrib/container"
)

var helloWorldContainer = func() string {
	v, ok := os.LookupEnv("HELLO_CONTAINER")
	if ok {
		return v
	}

	return "quay.io/podman/hello:latest"
}()

var validContainerProcessConfig = orchestrator.ProcessConfig{
	Name: "tests",
	Type: "container",
	ExecutionContext: map[string]string{
		"env":   `HELLO="WORLD"`,
		"image": helloWorldContainer,
		"pull":  "true",
	},
}

func TestNew(t *testing.T) {
	for _, test := range []struct {
		name        string
		pc          orchestrator.ProcessConfig
		expectError error
	}{
		{"empty config", orchestrator.ProcessConfig{}, container.ContainerImageMissingErr{}},
		{"full and valid config", validContainerProcessConfig, nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := container.New(test.pc)
			if err == nil && test.expectError != nil {
				t.Errorf("expected error, received none")
			} else if err != nil && test.expectError == nil {
				t.Errorf("unexpected error %#v", err)
			}

			if err != nil && test.expectError != nil {
				err.Error() // does nothing but increase codecoverage /shrug

				if !errors.Is(err, test.expectError) {
					t.Errorf("expected error of type %T, received %T", test.expectError, err)
				}
			}
		})
	}
}

func TestContainerProcess_Run(t *testing.T) {
	c, err := container.New(validContainerProcessConfig)
	if err != nil {
		t.Fatal(err)
	}

	ps, err := c.Run(context.Background(), orchestrator.Event{})
	if err != nil {
		t.Fatal(err)
	}

	expect := orchestrator.ProcessSuccess
	if expect != ps.Status {
		t.Errorf("status should be %v, received %v", expect, ps.Status)
	}
}
