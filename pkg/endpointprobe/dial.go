// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package endpointprobe

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"strings"
	"time"
)

const dialTimeout = 3 * time.Second

// DefaultDenyCIDRs are the ranges the probe refuses to connect to. The
// controller-manager holds cluster credentials and, unlike a workflow pod, is
// not sandboxed, so a consumer-controlled remoteURL is an SSRF vector.
//
// Loopback and link-local are denied because they address the controller
// itself and the cloud instance metadata service.
//
// The unspecified addresses (0.0.0.0, ::) are denied alongside loopback: on
// Linux, connect() to the unspecified address is routed to loopback, so
// omitting them would let a consumer-supplied http://0.0.0.0:8081/ reach the
// controller-manager's own health endpoint.
var DefaultDenyCIDRs = []netip.Prefix{
	netip.MustParsePrefix("127.0.0.0/8"),
	netip.MustParsePrefix("::1/128"),
	netip.MustParsePrefix("0.0.0.0/8"),
	netip.MustParsePrefix("::/128"),
	netip.MustParsePrefix("169.254.0.0/16"),
	netip.MustParsePrefix("fe80::/10"),
	netip.MustParsePrefix("fd00:ec2::254/128"),
}

// DeniedError reports that a probe target resolved to a denied address.
type DeniedError struct {
	IP netip.Addr
}

func (e *DeniedError) Error() string {
	return fmt.Sprintf("target address %s is not permitted", e.IP)
}

// ParseDenyCIDRs converts operator-supplied CIDR strings into prefixes. Blank
// entries are ignored so a trailing comma in the flag value is not an error.
func ParseDenyCIDRs(in []string) ([]netip.Prefix, error) {
	out := make([]netip.Prefix, 0, len(in))
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}

		p, err := netip.ParsePrefix(s)
		if err != nil {
			return nil, fmt.Errorf("invalid deny CIDR %q: %w", s, err)
		}
		out = append(out, p)
	}

	return out, nil
}

// DefaultDenyCIDRsString renders DefaultDenyCIDRs as a flag value, so --help
// shows the addresses that are actually refused rather than the word "default".
func DefaultDenyCIDRsString() string {
	parts := make([]string, 0, len(DefaultDenyCIDRs))
	for _, p := range DefaultDenyCIDRs {
		parts = append(parts, p.String())
	}

	return strings.Join(parts, ",")
}

func denied(deny []netip.Prefix, ip netip.Addr) bool {
	ip = ip.Unmap()
	for _, p := range deny {
		if p.Contains(ip) {
			return true
		}
	}

	return false
}

// guardedDialContext resolves the target, refuses denied addresses, then dials
// the vetted IP rather than the hostname.
//
// TLS is unaffected — http.Transport takes the SNI name and certificate
// hostname from the request URL, not from the dial address.
func guardedDialContext(deny []netip.Prefix) func(ctx context.Context, network, addr string) (net.Conn, error) {
	dialer := &net.Dialer{Timeout: dialTimeout}

	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, err
		}

		ips, err := net.DefaultResolver.LookupNetIP(ctx, "ip", host)
		if err != nil {
			return nil, err
		}
		if len(ips) == 0 {
			return nil, &net.DNSError{Err: "no addresses returned", Name: host}
		}

		for _, ip := range ips {
			if denied(deny, ip) {
				return nil, &DeniedError{IP: ip.Unmap()}
			}
		}

		return dialer.DialContext(ctx, network, net.JoinHostPort(ips[0].Unmap().String(), port))
	}
}
