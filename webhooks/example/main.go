package main

import (
	"context"
	"log"
	"net/http"
	"os"

	orchestrator "github.com/dapper-data/dapper-orchestrator"
	"github.com/dapper-data/dapper-orchestrator-contrib/webhooks"
)

var (
	destination = envOrDefault("WEBHOOKS_DESTINATION", "https://example.com")
)

func main() {
	orchestrator.ConcurrentProcessors = 1

	in, err := webhooks.NewWebhookInput(orchestrator.InputConfig{
		Name:             "webhooks-input-example",
		ConnectionString: "/webhooks/webhooks-input-example",
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
	// the WebhookInput to the WebhookProcess
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

	go func() {
		// This will expose our webhook input on
		// http://127.0.1.1:8888/webhooks/webhooks-input-example
		panic(http.ListenAndServe("127.0.1.1:8888", nil))
	}()

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
