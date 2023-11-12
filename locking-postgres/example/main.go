package main

import (
	"context"
	"log"
	"net/http"
	"os"

	orchestrator "github.com/dapper-data/dapper-orchestrator"
	"github.com/dapper-data/dapper-orchestrator-contrib/locking-postgres"
	"github.com/dapper-data/dapper-orchestrator-contrib/webhooks"
)

var (
	destination = envOrDefault("WEBHOOKS_DESTINATION", "https://example.com")
	database    = envOrDefault("DATABASE_DSN", "postgres://postgres:postgres@localhost:5432/tests?sslmode=disable")
)

func main() {
	orchestrator.ConcurrentProcessors = 1

	in, err := postgres.NewPostgresInput(orchestrator.InputConfig{
		Name:             "test-postgres-input",
		ConnectionString: database,
	})
	if err != nil {
		panic(err)
	}

	out, err := webhooks.NewWebhookProcess(orchestrator.ProcessConfig{
		Name: "test-webhook-input",
		ExecutionContext: map[string]string{
			webhooks.MethodKey:    http.MethodPost,
			webhooks.TargetURLKey: destination,
		},
	})
	if err != nil {
		panic(err)
	}

	// Configure an orchestrator, which will allow us to route events from
	// the PostgresInput to the WebhookProcess
	orch := orchestrator.New()

	err = orch.AddInput(context.Background(), in)
	if err != nil {
		panic(err)
	}

	err = orch.AddProcess(out)
	if err != nil {
		panic(err)
	}

	// Create a link from wh -> wp
	err = orch.AddLink(in, out)
	if err != nil {
		panic(err)
	}

	// Listen to errors from the orchestrator, acting accordingly
	for msg := range orch.ErrorChan {
		log.Print(msg)
	}
}

func envOrDefault(v, d string) string {
	s := os.Getenv(v)
	if s != "" {
		return s
	}

	return d
}
