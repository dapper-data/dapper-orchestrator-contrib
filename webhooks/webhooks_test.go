package webhooks_test

import (
	"context"
	"log"
	"net/http"

	orchestrator "github.com/dapper-data/dapper-orchestrator"
	"github.com/dapper-data/dapper-orchestrator-contrib/webhooks"
)

func ExampleWebhookInputAndProcess() {
	// Create a WebhookInput listening on the path /webhooks/test-webhook-input/events
	wh, err := webhooks.NewWebhookInput(orchestrator.InputConfig{
		Name:             "test-webhook-input",
		ConnectionString: "/webhooks/test-webhook-input/events",
	})
	if err != nil {
		panic(err)
	}

	// Create a WebhookProcess which sends events onto a further destination
	wp, err := webhooks.NewWebhookProcess(orchestrator.ProcessConfig{
		Name: "test-webhook-input",
		ExecutionContext: map[string]string{
			webhooks.MethodKey:    http.MethodPost,
			webhooks.TargetURLKey: "https://httpbin.org/status/200",
		},
	})
	if err != nil {
		panic(err)
	}

	// Configure an orchestrator, which will allow us to route events from
	// the WebhookInput to the WebhookProcess
	orch := orchestrator.New()

	err = orch.AddInput(context.Background(), wh)
	if err != nil {
		panic(err)
	}

	err = orch.AddProcess(wp)
	if err != nil {
		panic(err)
	}

	// Create a link from wh -> wp
	err = orch.AddLink(wh, wp)
	if err != nil {
		panic(err)
	}

	go func() {
		// This will expose our webhook input on
		// http://127.0.1.1:8888/webhooks/test-webhook-input/events
		panic(http.ListenAndServe("127.0.1.1:8888", nil))
	}()

	// Listen to errors from the orchestrator, acting accordingly
	for msg := range orch.ErrorChan {
		log.Print(msg)
	}
}
