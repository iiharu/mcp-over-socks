package unit

import (
	"testing"

	"github.com/iiharu/mcp-over-socks/internal/transport"
)

func TestParseTransportType(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  transport.TransportType
	}{
		{
			name:  "sse lowercase",
			input: "sse",
			want:  transport.TransportTypeSSE,
		},
		{
			name:  "sse uppercase",
			input: "SSE",
			want:  transport.TransportTypeSSE,
		},
		{
			name:  "streamable",
			input: "streamable",
			want:  transport.TransportTypeStreamableHTTP,
		},
		{
			name:  "http",
			input: "http",
			want:  transport.TransportTypeStreamableHTTP,
		},
		{
			name:  "auto",
			input: "auto",
			want:  transport.TransportTypeAuto,
		},
		{
			name:  "empty string",
			input: "",
			want:  transport.TransportTypeAuto,
		},
		{
			name:  "unknown",
			input: "unknown",
			want:  transport.TransportTypeAuto,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := transport.ParseTransportType(tt.input)
			if got != tt.want {
				t.Errorf("ParseTransportType(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestTransportTypeString(t *testing.T) {
	tests := []struct {
		name string
		t    transport.TransportType
		want string
	}{
		{
			name: "SSE",
			t:    transport.TransportTypeSSE,
			want: "sse",
		},
		{
			name: "Streamable HTTP",
			t:    transport.TransportTypeStreamableHTTP,
			want: "streamable",
		},
		{
			name: "Auto",
			t:    transport.TransportTypeAuto,
			want: "auto",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := string(tt.t)
			if got != tt.want {
				t.Errorf("TransportType string = %q, want %q", got, tt.want)
			}
		})
	}
}

