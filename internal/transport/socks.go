// Package transport provides transport implementations for the MCP over SOCKS bridge.
package transport

import (
	"context"
	"net"
	"net/http"

	"golang.org/x/net/proxy"
)

// SOCKSDialer wraps a SOCKS5 proxy dialer.
type SOCKSDialer struct {
	dialer proxy.Dialer
}

// SOCKSError represents a SOCKS-related error with user-friendly message.
type SOCKSError struct {
	Message string
	Err     error
}

func (e *SOCKSError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

func (e *SOCKSError) Unwrap() error {
	return e.Err
}

// NewSOCKSDialer creates a new SOCKS5 dialer.
// proxyAddr should be in the format "host:port".
// auth can be nil for no authentication.
func NewSOCKSDialer(proxyAddr string, auth *proxy.Auth) (*SOCKSDialer, error) {
	if proxyAddr == "" {
		return nil, &SOCKSError{
			Message: "SOCKS proxy address is empty",
		}
	}

	dialer, err := proxy.SOCKS5("tcp", proxyAddr, auth, proxy.Direct)
	if err != nil {
		return nil, &SOCKSError{
			Message: "Failed to create SOCKS5 dialer for " + proxyAddr,
			Err:     err,
		}
	}
	return &SOCKSDialer{dialer: dialer}, nil
}

// Dial connects to the address on the named network through the SOCKS5 proxy.
func (d *SOCKSDialer) Dial(network, addr string) (net.Conn, error) {
	return d.dialer.Dial(network, addr)
}

// DialContext connects to the address on the named network through the SOCKS5 proxy with context.
func (d *SOCKSDialer) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	// Check if the dialer supports DialContext
	if ctxDialer, ok := d.dialer.(proxy.ContextDialer); ok {
		return ctxDialer.DialContext(ctx, network, addr)
	}

	// Fallback: use channel to handle context cancellation
	type dialResult struct {
		conn net.Conn
		err  error
	}
	resultCh := make(chan dialResult, 1)

	go func() {
		conn, err := d.dialer.Dial(network, addr)
		resultCh <- dialResult{conn: conn, err: err}
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case result := <-resultCh:
		return result.conn, result.err
	}
}

// HTTPTransport creates an http.Transport that uses this SOCKS5 dialer.
func (d *SOCKSDialer) HTTPTransport() *http.Transport {
	return &http.Transport{
		DialContext: d.DialContext,
	}
}

// HTTPClient creates an http.Client that uses this SOCKS5 dialer.
func (d *SOCKSDialer) HTTPClient() *http.Client {
	return &http.Client{
		Transport: d.HTTPTransport(),
	}
}

