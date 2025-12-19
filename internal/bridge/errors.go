// Package bridge provides the MCP bridge between stdio and SSE transport.
package bridge

import "errors"

// Error types for the bridge.
var (
	// ErrInvalidConfig is returned when the configuration is invalid.
	ErrInvalidConfig = errors.New("invalid configuration")

	// ErrProxyConnection is returned when the SOCKS proxy connection fails.
	ErrProxyConnection = errors.New("failed to connect to SOCKS proxy")

	// ErrServerConnection is returned when the MCP server connection fails.
	ErrServerConnection = errors.New("failed to connect to MCP server")

	// ErrTimeout is returned when a request times out.
	ErrTimeout = errors.New("request timeout")

	// ErrConnectionClosed is returned when the connection is unexpectedly closed.
	ErrConnectionClosed = errors.New("connection closed")
)

// WrapError wraps an error with a more user-friendly message.
func WrapError(err error, message string) error {
	if err == nil {
		return nil
	}
	return &BridgeError{
		Message: message,
		Err:     err,
	}
}

// BridgeError is a custom error type that provides more context.
type BridgeError struct {
	Message string
	Err     error
}

func (e *BridgeError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

func (e *BridgeError) Unwrap() error {
	return e.Err
}

// IsProxyError checks if the error is related to proxy connection.
func IsProxyError(err error) bool {
	return errors.Is(err, ErrProxyConnection)
}

// IsServerError checks if the error is related to server connection.
func IsServerError(err error) bool {
	return errors.Is(err, ErrServerConnection)
}

// IsTimeoutError checks if the error is a timeout.
func IsTimeoutError(err error) bool {
	return errors.Is(err, ErrTimeout)
}

// FormatUserFriendlyError formats an error for display to the user.
func FormatUserFriendlyError(err error) string {
	if err == nil {
		return ""
	}

	switch {
	case errors.Is(err, ErrProxyConnection):
		return "Cannot connect to SOCKS proxy. Please check:\n" +
			"  1. The SOCKS proxy is running\n" +
			"  2. The proxy address is correct (e.g., socks5://localhost:1080)\n" +
			"  3. No firewall is blocking the connection"

	case errors.Is(err, ErrServerConnection):
		return "Cannot connect to MCP server. Please check:\n" +
			"  1. The MCP server is running\n" +
			"  2. The server URL is correct\n" +
			"  3. The server is accessible through the SOCKS proxy"

	case errors.Is(err, ErrTimeout):
		return "Request timed out. Please check:\n" +
			"  1. Network connectivity\n" +
			"  2. Server responsiveness\n" +
			"  3. Consider increasing --timeout value"

	case errors.Is(err, ErrInvalidConfig):
		return "Invalid configuration. Run 'mcp-over-socks --help' for usage."

	default:
		return err.Error()
	}
}
