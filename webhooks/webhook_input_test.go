package webhooks

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	orchestrator "github.com/dapper-data/dapper-orchestrator"
)

func TestWebhookInput_Handle(t *testing.T) {
	wh, err := NewWebhookInput(orchestrator.InputConfig{
		Name:             "test-webhook-input",
		ConnectionString: "/webhooks/test-webhook-input/events",
	})
	if err != nil {
		t.Fatal(err)
	}

	wh.c = make(chan orchestrator.Event)

	events := make([]orchestrator.Event, 0)
	go func() {
		for event := range wh.c {
			events = append(events, event)
		}
	}()

	for _, test := range []struct {
		name         string
		input        string
		expectStatus int
	}{
		{"Empty object returns 202 but is useless", `{}`, 202},
		{"Valid object", `{"location":"a-table","operation":"create","id":"0xabadbabe","trigger":"test-webhook-input"}`, 202},
		{"Empty input", ``, 400},
	} {
		t.Run(test.name, func(t *testing.T) {
			b := bytes.NewBufferString(test.input)
			req, err := http.NewRequest(http.MethodPost, "/webhooks/test-webhook-input/events", b)
			if err != nil {
				t.Fatal(err)
			}

			recorder := httptest.NewRecorder()

			wh.handler(recorder, req)

			result := recorder.Result()
			if test.expectStatus != result.StatusCode {
				t.Errorf("expected %d, received %d", test.expectStatus, result.StatusCode)
			}
		})
	}

	// let channels catch up
	time.Sleep(time.Millisecond)

	if len(events) != 2 {
		t.Errorf("expected 2 event(s), recieved %d:\n%#v", len(events), events)
	}
}
