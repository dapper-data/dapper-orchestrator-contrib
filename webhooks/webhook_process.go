package webhooks

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	orchestrator "github.com/dapper-data/dapper-orchestrator"
)

// Constant ExecutionContext keys used in creating
// WebhookProcess instances; such as:
//
//	webhooks.NewWebhookProcess(orchestrator.ProcessConfig{
//	    Name: "foo",
//	    ExecutionContext: map[string]string{
//		webhooks.TargetURLKey: "https://example.com",
//		webhooks.MethodKey:    "PUT",
//	    },
//	})
//
// By using these keys, rather than naked strings, developers can rely on
// compile time checks on the correctness of their webhooks, rather than
// runtime index errors
const (
	// TargetURLKey should point towards the URL that receives
	// an event
	TargetURLKey = "url"

	// MethodKey should point to the http verb/ method used to
	// hit the endpoint defined under TargetURLKey
	//
	// The default value is POST
	MethodKey = "method"
)

// MissingWebhookURLErr is returned when an ExecutionContext does not
// contain a url to hit
type MissingWebhookURLErr struct{}

// Error returns the error text for this error
func (e MissingWebhookURLErr) Error() string {
	return fmt.Sprintf("error creating webhook: missing %q config value", TargetURLKey)
}

// BadStatusErr is returned when a call returns a non-2xx response
type BadStatusErr struct{ url, status string }

// Error returns the error text for this error
func (e BadStatusErr) Error() string {
	return fmt.Sprintf("error calling webhook: %s returned %q", e.url, e.status)
}

// WebhookProcess implements the orchestrator.Process interface
//
// When triggered, it sends a the orchestrator.Event is was called with
// as JSON to the endpoint the WebhookProcess was instantiated with via the arguments to
// orchestrator.ProcessConfig.ExecutionContext, passed as argument 'pc'
//
// For custom process endpoints, simply copy the code in github.com/dapper-data/dapper-orchestrator-contrib/webhooks
// and replace the bits you want to replace
type WebhookProcess struct {
	pc        orchestrator.ProcessConfig
	targetURL string
	method    string
}

// NewWebhookProcess is an orchestrator.NewProcessFunc which configures a new
// WebhookProcess.
//
// It expects the following keys set within the ExecutionContext:
//
//	var pc orchestrator.ProcessConfig
//	pc.ExecutionContext = map[string]string{
//	    webhooks.MethodKey:     http.MethodPut,              // defaults to POST
//	    webhooks.TargetURLKey: "https://example.com/",       // errors if unset or empty
//	}
func NewWebhookProcess(pc orchestrator.ProcessConfig) (wh WebhookProcess, err error) {
	var ok bool

	wh.pc = pc
	wh.method = wh.executionContextOrDefault(MethodKey, http.MethodPost)
	wh.targetURL, ok = pc.ExecutionContext[TargetURLKey]
	if !ok {
		err = MissingWebhookURLErr{}
	}

	return
}

// Run will, given an orchestrator.Event, encode that Event to JSON and send it
// to the endpoint the WebhookProcess was configured with via the function NewWebhookProcess
//
// A non-2xx response will return a webhooks.BadStatusErr which describes status
// returned.
//
// Additionally, the logs field of the returned orchestrator.ProcessStatus will contain
// errors, warnings, and response metadata (which can be ignored if err == nil)
func (w WebhookProcess) Run(ctx context.Context, e orchestrator.Event) (ps orchestrator.ProcessStatus, err error) {
	ps.Name = w.ID()
	ps.Status = orchestrator.ProcessUnstarted
	ps.Logs = make([]string, 0)

	b := new(bytes.Buffer)
	err = json.NewEncoder(b).Encode(e)
	if err != nil {
		ps.Logs = append(ps.Logs, err.Error())

		return
	}

	req, err := http.NewRequestWithContext(ctx, w.method, w.targetURL, b)
	if err != nil {
		return
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}

	if resp.StatusCode/100 != 2 {
		err = BadStatusErr{w.targetURL, resp.Status}
	}

	switch err {
	case nil:
		ps.Status = orchestrator.ProcessSuccess
	default:
		ps.Logs = append(ps.Logs, err.Error())
		ps.Status = orchestrator.ProcessFail
	}

	return
}

// ID returns an ID for this process
func (w WebhookProcess) ID() string {
	return w.pc.ID()
}

func (w WebhookProcess) executionContextOrDefault(key, def string) string {
	v, ok := w.pc.ExecutionContext[key]
	if ok {
		return v
	}

	return def
}
