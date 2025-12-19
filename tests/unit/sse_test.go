package unit

import (
	"strings"
	"testing"
)

func TestSSEEventParsing(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantEvents []struct {
			Event string
			Data  string
			ID    string
		}
	}{
		{
			name:  "simple data event",
			input: "data: hello world\n\n",
			wantEvents: []struct {
				Event string
				Data  string
				ID    string
			}{
				{Data: "hello world"},
			},
		},
		{
			name:  "data with event type",
			input: "event: message\ndata: hello world\n\n",
			wantEvents: []struct {
				Event string
				Data  string
				ID    string
			}{
				{Event: "message", Data: "hello world"},
			},
		},
		{
			name:  "data with id",
			input: "id: 123\ndata: hello world\n\n",
			wantEvents: []struct {
				Event string
				Data  string
				ID    string
			}{
				{ID: "123", Data: "hello world"},
			},
		},
		{
			name:  "multiline data",
			input: "data: line1\ndata: line2\ndata: line3\n\n",
			wantEvents: []struct {
				Event string
				Data  string
				ID    string
			}{
				{Data: "line1\nline2\nline3"},
			},
		},
		{
			name:  "multiple events",
			input: "data: event1\n\ndata: event2\n\n",
			wantEvents: []struct {
				Event string
				Data  string
				ID    string
			}{
				{Data: "event1"},
				{Data: "event2"},
			},
		},
		{
			name:  "JSON data",
			input: "data: {\"jsonrpc\":\"2.0\",\"id\":1,\"result\":{}}\n\n",
			wantEvents: []struct {
				Event string
				Data  string
				ID    string
			}{
				{Data: "{\"jsonrpc\":\"2.0\",\"id\":1,\"result\":{}}"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			events := parseSSEEvents(tt.input)
			if len(events) != len(tt.wantEvents) {
				t.Errorf("got %d events, want %d", len(events), len(tt.wantEvents))
				return
			}
			for i, want := range tt.wantEvents {
				got := events[i]
				if got.Event != want.Event {
					t.Errorf("event[%d].Event = %q, want %q", i, got.Event, want.Event)
				}
				if got.Data != want.Data {
					t.Errorf("event[%d].Data = %q, want %q", i, got.Data, want.Data)
				}
				if got.ID != want.ID {
					t.Errorf("event[%d].ID = %q, want %q", i, got.ID, want.ID)
				}
			}
		})
	}
}

// SSEEvent represents a Server-Sent Event for testing.
type SSEEvent struct {
	Event string
	Data  string
	ID    string
}

// parseSSEEvents parses SSE events from a string (for testing).
func parseSSEEvents(input string) []SSEEvent {
	var events []SSEEvent
	var current SSEEvent
	var dataLines []string

	lines := strings.Split(input, "\n")
	for _, line := range lines {
		if line == "" {
			if len(dataLines) > 0 {
				current.Data = strings.Join(dataLines, "\n")
				events = append(events, current)
				current = SSEEvent{}
				dataLines = nil
			}
			continue
		}

		if strings.HasPrefix(line, "data:") {
			data := strings.TrimPrefix(line, "data:")
			data = strings.TrimPrefix(data, " ")
			dataLines = append(dataLines, data)
		} else if strings.HasPrefix(line, "event:") {
			current.Event = strings.TrimPrefix(line, "event:")
			current.Event = strings.TrimPrefix(current.Event, " ")
		} else if strings.HasPrefix(line, "id:") {
			current.ID = strings.TrimPrefix(line, "id:")
			current.ID = strings.TrimPrefix(current.ID, " ")
		}
	}

	return events
}

