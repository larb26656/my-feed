// Package outbound provides HTTP access to untrusted origins.
package outbound

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"strings"
	"time"
)

// NewClient applies the same destination policy to initial requests and redirects.
// The supplied client's transport is used when explicitly configured, allowing
// callers to supply a controlled transport; otherwise connections use guarded DNS.
func NewClient(base *http.Client) *http.Client {
	if base == nil {
		base = http.DefaultClient
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	dialer := &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}
	transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}
		addresses, err := net.DefaultResolver.LookupNetIP(ctx, "ip", host)
		if err != nil {
			return nil, err
		}
		for _, ip := range addresses {
			if !publicIP(ip) {
				return nil, fmt.Errorf("outbound destination is not public")
			}
		}
		if len(addresses) == 0 {
			return nil, fmt.Errorf("outbound destination has no addresses")
		}
		var last error
		for _, ip := range addresses {
			var conn net.Conn
			conn, last = dialer.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
			if last == nil {
				return conn, nil
			}
		}
		return nil, last
	}
	return &http.Client{Timeout: base.Timeout, Transport: &guardedTransport{base: base, fallback: transport}, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 10 {
			return fmt.Errorf("too many redirects")
		}
		if err := validate(req); err != nil {
			return err
		}
		if base.CheckRedirect != nil {
			return base.CheckRedirect(req, via)
		}
		return nil
	}}
}

type guardedTransport struct {
	base     *http.Client
	fallback http.RoundTripper
}

func (t *guardedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if err := validate(req); err != nil {
		return nil, err
	}
	transport := t.base.Transport
	if transport == nil {
		transport = t.fallback
	}
	return transport.RoundTrip(req)
}

func validate(req *http.Request) error {
	if req.URL.Scheme != "http" && req.URL.Scheme != "https" {
		return fmt.Errorf("unsupported outbound scheme")
	}
	host := req.URL.Hostname()
	if host == "" || strings.EqualFold(host, "localhost") || strings.HasSuffix(strings.ToLower(host), ".localhost") {
		return fmt.Errorf("invalid outbound destination")
	}
	if ip, err := netip.ParseAddr(host); err == nil && !publicIP(ip) {
		return fmt.Errorf("outbound destination is not public")
	}
	return nil
}

func publicIP(ip netip.Addr) bool {
	ip = ip.Unmap()
	return ip.IsGlobalUnicast() && !ip.IsPrivate() && !ip.IsLoopback() && !ip.IsLinkLocalUnicast() && !ip.IsUnspecified()
}
