// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

// Package endpointprobe tests whether an ARC Endpoint's target is reachable
// and whether its credentials are accepted.
//
// It speaks plain HTTP and the OCI distribution-spec ping endpoint, which
// between them cover the Endpoint types ARC ships examples for. Types whose
// protocol or credential shape it cannot verify are reported as Unknown rather
// than guessed at.
package endpointprobe

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"strings"
	"syscall"
	"time"
	"unicode/utf8"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	probeTimeout = 5 * time.Second
	maxBodyBytes = 1 << 20
	userAgent    = "arc-endpoint-probe"

	// maxResponseHeaderBytes bounds the size of the response status line and
	// headers the transport will read. It exists independently of
	// messageTruncateLimit: this caps what the transport will even buffer,
	// truncate bounds what a remote-derived string contributes to a Check
	// message once read.
	maxResponseHeaderBytes = 64 << 10

	// messageTruncateLimit is the budget applied to every remote-derived
	// string (a status line, a challenge realm) before it is interpolated
	// into a Check message. Those messages are written to etcd via the
	// Endpoint's status conditions, so an attacker-controlled target must
	// never be able to grow one without bound.
	messageTruncateLimit = 256
)

// Reachable reasons.
const (
	ReasonReachable         = "Reachable"
	ReasonDNSFailure        = "DNSFailure"
	ReasonConnectionRefused = "ConnectionRefused"
	ReasonTLSError          = "TLSError"
	ReasonTimeout           = "Timeout"
	ReasonTargetDenied      = "TargetDenied"
	ReasonUnsupportedScheme = "UnsupportedScheme"
	ReasonProbeFailed       = "ProbeFailed"
)

// Authenticated reasons.
const (
	ReasonAuthenticated         = "Authenticated"
	ReasonUnauthorized          = "Unauthorized"
	ReasonNoCredentials         = "NoCredentials"
	ReasonUnsupportedAuthScheme = "UnsupportedAuthScheme"
	ReasonInconclusive          = "Inconclusive"
)

var errUnsupportedScheme = errors.New("only http and https targets can be probed")

// Check is one condition's worth of outcome.
type Check struct {
	Status  metav1.ConditionStatus
	Reason  string
	Message string
}

// Target is everything the probe knows about an Endpoint. Credentials are
// passed by value rather than as a Secret so the probe never touches the
// Kubernetes API.
type Target struct {
	RemoteURL string
	// HasSecret records that the Endpoint referenced a Secret, which
	// distinguishes "no credentials configured" from "credentials in a shape
	// ARC cannot use".
	HasSecret bool
	Username  string
	Password  string
}

func (t Target) usableCredentials() bool {
	return t.Username != "" && t.Password != ""
}

// Result is the pair of outcomes the controller turns into conditions.
type Result struct {
	Reachable     Check
	Authenticated Check
}

// Prober performs endpoint probes through a hardened HTTP client.
type Prober struct {
	client *http.Client
}

// New returns a Prober that refuses to connect to the given CIDRs.
func New(deny []netip.Prefix) *Prober {
	return &Prober{
		client: &http.Client{
			Timeout: probeTimeout,
			Transport: &http.Transport{
				DialContext:            guardedDialContext(deny),
				TLSHandshakeTimeout:    dialTimeout,
				DisableKeepAlives:      true,
				MaxResponseHeaderBytes: maxResponseHeaderBytes,
			},
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

// Probe tests the target and reports reachability and credential validity.
func (p *Prober) Probe(ctx context.Context, t Target) Result {
	u, err := targetURL(t.RemoteURL)
	if err != nil {
		return Result{
			Reachable:     Check{metav1.ConditionUnknown, ReasonUnsupportedScheme, err.Error()},
			Authenticated: Check{metav1.ConditionUnknown, ReasonInconclusive, "target was not probed"},
		}
	}

	ctx, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()

	resp, err := p.do(ctx, http.MethodGet, u.String(), t)
	if err != nil {
		return Result{
			Reachable:     classifyTransportError(err),
			Authenticated: Check{metav1.ConditionUnknown, ReasonInconclusive, "target was not reached"},
		}
	}
	defer drainAndClose(resp)

	return Result{
		Reachable: Check{
			metav1.ConditionTrue,
			ReasonReachable,
			fmt.Sprintf("%s responded with %s", u.Host, truncate(resp.Status, messageTruncateLimit)),
		},
		Authenticated: p.authenticated(ctx, resp, t, u.Scheme),
	}
}

// truncate bounds s to at most max runes, so a remote-derived string (a
// response status line, an auth challenge's realm) cannot grow a Check
// message — and the etcd write it eventually becomes — without bound. It cuts
// on a rune boundary so it never produces invalid UTF-8.
func truncate(s string, limit int) string {
	if utf8.RuneCountInString(s) <= limit {
		return s
	}

	runes := []rune(s)

	return string(runes[:limit]) + "…(truncated)"
}

// do issues one request, attaching basic auth when the Target carries
// credentials in a shape the probe can use.
func (p *Prober) do(ctx context.Context, method, rawURL string, t Target) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)

	if t.usableCredentials() {
		req.SetBasicAuth(t.Username, t.Password)
	}

	return p.client.Do(req)
}

// targetURL turns spec.remoteURL into the URL to probe.
//
// A bare host such as gcr.io or dst.zot is an OCI registry, so the
// distribution-spec ping endpoint is used: it answers 200 anonymously on public
// registries and 401 on private ones, which makes it a credential test as well
// as a reachability test. An http or https URL is probed exactly as given.
// Anything else is refused rather than guessed at.
func targetURL(remote string) (*url.URL, error) {
	remote = strings.TrimSpace(remote)
	if remote == "" {
		return nil, errors.New("remoteURL is empty")
	}

	if !strings.Contains(remote, "://") {
		return url.Parse("https://" + strings.TrimSuffix(remote, "/") + "/v2/")
	}

	u, err := url.Parse(remote)
	if err != nil {
		return nil, err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("%w: %s", errUnsupportedScheme, u.Scheme)
	}

	return u, nil
}

// classifyTransportError maps a failed request onto a Reachable reason. DNS is
// checked before timeout because a DNS timeout satisfies both and the name
// resolution failure is the more useful thing to tell the consumer.
func classifyTransportError(err error) Check {
	var (
		deniedErr *DeniedError
		dnsErr    *net.DNSError
		certErr   *tls.CertificateVerificationError
	)

	switch {
	case errors.As(err, &deniedErr):
		return Check{metav1.ConditionFalse, ReasonTargetDenied, deniedErr.Error()}
	case errors.As(err, &dnsErr):
		return Check{metav1.ConditionFalse, ReasonDNSFailure, dnsErr.Error()}
	case errors.As(err, &certErr):
		return Check{metav1.ConditionFalse, ReasonTLSError, certErr.Error()}
	case errors.Is(err, context.DeadlineExceeded), os.IsTimeout(err):
		return Check{metav1.ConditionFalse, ReasonTimeout, err.Error()}
	case errors.Is(err, syscall.ECONNREFUSED):
		return Check{metav1.ConditionFalse, ReasonConnectionRefused, err.Error()}
	default:
		return Check{metav1.ConditionFalse, ReasonProbeFailed, err.Error()}
	}
}

func drainAndClose(resp *http.Response) {
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, maxBodyBytes))
	_ = resp.Body.Close()
}
