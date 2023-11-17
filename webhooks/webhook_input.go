package webhooks

import (
	"context"
	"encoding/json"
	"net/http"

	orchestrator "github.com/dapper-data/dapper-orchestrator"
)

// Input implements the orchestrator.Input interface
//
// It listens to a user specified path (as specified in the InputConfig.ConnectionString
// argument to NewWebhookInput), and expects to receive a valid orchestrator.Event as
// JSON
//
// For custom input payloads, simply copy the code in github.com/dapper-data/dapper-orchestrator-contrib/webhooks
// and replace the bits you want to replace
type Input struct {
	ic orchestrator.InputConfig
	c  chan orchestrator.Event
}

// NewInput is an orchestrator.NewInputFunc which configures a new
// WebhookInput, exposed on the URL specified in the ConnectionString field
// of the InputConfig passed to this function.
//
// This Input wont automatically expose an HTTP server; the application this
// type is embedded in needs to do that- see this package's examples for an
// example of how this might be done
func NewInput(ic orchestrator.InputConfig) (wh *Input, err error) {
	wh = new(Input)
	wh.ic = ic

	return
}

// Handle implements the Handle function of the orchestrator.Input interface
//
// It registers a handler function against the default server provided by the
// net/http package and listens for Events which are then passed down Event chan `c`
//
// This function exits immediately
func (w *Input) Handle(ctx context.Context, c chan orchestrator.Event) (err error) {
	w.c = make(chan orchestrator.Event)
	http.HandleFunc(w.ic.ConnectionString, w.handler)

	// This allows us to keep this supposedly long running function
	// running, rather than registering am http.HandleFunc and returning
	// early- the orchestrator expects a Handle to return on errors only,
	// and so will bomb out
	for e := range w.c {
		c <- e
	}

	return
}

func (w Input) handler(wr http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()

	e := new(orchestrator.Event)
	err := json.NewDecoder(req.Body).Decode(e)
	if err != nil {
		wr.WriteHeader(http.StatusBadRequest)

		return
	}

	e.Trigger = w.ID()

	w.c <- *e

	wr.WriteHeader(http.StatusAccepted)
}

// ID returns an ID for this input
func (w Input) ID() string {
	return w.ic.ID()
}
