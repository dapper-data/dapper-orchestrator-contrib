package webhooks

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"reflect"
	"testing"

	orchestrator "github.com/dapper-data/dapper-orchestrator"
)

func TestNewProcess(t *testing.T) {
	for _, test := range []struct {
		name        string
		pc          orchestrator.ProcessConfig
		expectError error
	}{
		{"empty config errors on missing target url", orchestrator.ProcessConfig{}, MissingWebhookURLErr{}},
		{"valid config is fine", orchestrator.ProcessConfig{
			ExecutionContext: map[string]string{"url": "https://example.com"},
		}, nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := NewProcess(test.pc)
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

func TestProcess_Run(t *testing.T) {
	successPS := orchestrator.ProcessStatus{
		Name:   "tests",
		Status: orchestrator.ProcessSuccess,
		Logs:   []string{},
	}

	unstartedPS := orchestrator.ProcessStatus{
		Name:   "tests",
		Status: orchestrator.ProcessUnstarted,
		Logs:   []string{},
	}

	status404PS := orchestrator.ProcessStatus{
		Name:   "tests",
		Status: orchestrator.ProcessFail,
		Logs: []string{
			`error calling webhook: https://httpbin.org/status/404 returned "404 Not Found"`,
		},
	}

	status503PS := orchestrator.ProcessStatus{
		Name:   "tests",
		Status: orchestrator.ProcessFail,
		Logs: []string{
			`error calling webhook: https://httpbin.org/status/503 returned "503 Service Unavailable"`,
		},
	}

	for _, test := range []struct {
		name        string
		url         string
		expect      orchestrator.ProcessStatus
		expectError error
	}{
		{"webhook returns 200, no error", "https://httpbin.org/status/200", successPS, nil},
		{"webhook returns 201, no error", "https://httpbin.org/status/201", successPS, nil},
		{"webhook returns 202, no error", "https://httpbin.org/status/202", successPS, nil},
		{"webhook returns 203, no error", "https://httpbin.org/status/203", successPS, nil},
		{"webhook returns 204, no error", "https://httpbin.org/status/204", successPS, nil},

		{"webhook returns 404, errors", "https://httpbin.org/status/404", status404PS, BadStatusErr{}},
		{"webhook returns 503, errors", "https://httpbin.org/status/503", status503PS, BadStatusErr{}},

		{"malformed url", "this is a malformed address", unstartedPS, new(url.Error)},
		{"non-existent url", "https://webhooks.test/webhook", unstartedPS, new(url.Error)},
	} {
		t.Run(test.name, func(t *testing.T) {
			w, err := NewProcess(orchestrator.ProcessConfig{
				Name: "tests",
				ExecutionContext: map[string]string{
					"url":    test.url,
					"method": "PUT",
				},
			})
			if err != nil {
				t.Fatal(err)
			}

			ps, err := w.Run(context.Background(), orchestrator.Event{
				Location:  "testdb",
				Operation: orchestrator.OperationRead,
				ID:        "a-record",
				Trigger:   "test-input",
			})
			if err == nil && test.expectError != nil {
				t.Errorf("expected error, received none")
			} else if err != nil && test.expectError == nil {
				t.Errorf("unexpected error %#v", err)
			}

			if err != nil && test.expectError != nil {
				err.Error() // does nothing but increase codecoverage /shrug

				expectType := fmt.Sprintf("%T", test.expectError)
				receivedType := fmt.Sprintf("%T", err)

				if expectType != receivedType {
					t.Errorf("expected error of type %T, received %T", test.expectError, err)
				}
			}

			if !reflect.DeepEqual(test.expect, ps) {
				t.Errorf("expected\n%#v\nreceived\n%#v", test.expect, ps)
			}
		})
	}
}
