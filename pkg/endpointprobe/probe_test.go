// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package endpointprobe

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"sync/atomic"
	"testing"
	"unicode/utf8"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGuardedDialRefusesDeniedAddress(t *testing.T) {
	dial := guardedDialContext(DefaultDenyCIDRs)

	_, err := dial(context.Background(), "tcp", "127.0.0.1:9")

	var denied *DeniedError
	if !errors.As(err, &denied) {
		t.Fatalf("want DeniedError, got %v", err)
	}
	if denied.IP.String() != "127.0.0.1" {
		t.Fatalf("want 127.0.0.1, got %s", denied.IP)
	}
}

func TestGuardedDialRefusesLinkLocalMetadata(t *testing.T) {
	dial := guardedDialContext(DefaultDenyCIDRs)

	_, err := dial(context.Background(), "tcp", "169.254.169.254:80")

	var denied *DeniedError
	if !errors.As(err, &denied) {
		t.Fatalf("want DeniedError for cloud metadata address, got %v", err)
	}
}

func TestGuardedDialRefusesUnspecifiedIPv4(t *testing.T) {
	// On Linux, connect() to 0.0.0.0 is routed to loopback, which would let
	// http://0.0.0.0:<port>/ reach the controller-manager's own listeners.
	dial := guardedDialContext(DefaultDenyCIDRs)

	_, err := dial(context.Background(), "tcp", "0.0.0.0:9")

	var denied *DeniedError
	if !errors.As(err, &denied) {
		t.Fatalf("want DeniedError for 0.0.0.0, got %v", err)
	}
}

func TestGuardedDialRefusesUnspecifiedIPv6(t *testing.T) {
	dial := guardedDialContext(DefaultDenyCIDRs)

	_, err := dial(context.Background(), "tcp", "[::]:9")

	var denied *DeniedError
	if !errors.As(err, &denied) {
		t.Fatalf("want DeniedError for ::, got %v", err)
	}
}

func TestGuardedDialAllowsWhenDenyListEmpty(t *testing.T) {
	ln, err := new(net.ListenConfig).Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	dial := guardedDialContext(nil)

	conn, err := dial(context.Background(), "tcp", ln.Addr().String())
	if err != nil {
		t.Fatalf("want success with empty deny list, got %v", err)
	}
	conn.Close()
}

func TestParseDenyCIDRs(t *testing.T) {
	got, err := ParseDenyCIDRs([]string{"10.0.0.0/8", "::1/128"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("want 2 prefixes, got %d", len(got))
	}

	if _, err := ParseDenyCIDRs([]string{"not-a-cidr"}); err == nil {
		t.Fatal("want error for malformed CIDR")
	}
}

func TestDefaultDenyCIDRsAllowsPrivateRanges(t *testing.T) {
	for _, addr := range []string{"10.1.2.3", "172.16.0.5", "192.168.1.10"} {
		ip := netip.MustParseAddr(addr)
		if denied(DefaultDenyCIDRs, ip) {
			t.Fatalf("%s must not be denied by default", addr)
		}
	}
}

// proberFor returns a Prober that trusts srv's certificate and has no deny
// list, so tests can talk to httptest servers on 127.0.0.1.
func proberFor(srv *httptest.Server) *Prober {
	p := New(nil)
	p.client.Transport = srv.Client().Transport

	return p
}

func TestProbeSchemelessHostUsesRegistryPing(t *testing.T) {
	var path atomic.Value
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path.Store(r.URL.Path)
	}))
	defer srv.Close()

	host := strings.TrimPrefix(srv.URL, "https://")
	got := proberFor(srv).Probe(context.Background(), Target{RemoteURL: host})

	if got.Reachable.Status != metav1.ConditionTrue {
		t.Fatalf("want Reachable True, got %+v", got.Reachable)
	}
	if got.Reachable.Reason != ReasonReachable {
		t.Fatalf("want reason %s, got %s", ReasonReachable, got.Reachable.Reason)
	}
	if p, _ := path.Load().(string); p != "/v2/" {
		t.Fatalf("a bare host is an OCI registry: want /v2/, got %q", p)
	}
}

func TestProbeHTTPURLProbedAsGiven(t *testing.T) {
	var path atomic.Value
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path.Store(r.URL.Path)
	}))
	defer srv.Close()

	got := proberFor(srv).Probe(context.Background(), Target{RemoteURL: srv.URL + "/charts"})

	if got.Reachable.Status != metav1.ConditionTrue {
		t.Fatalf("want Reachable True, got %+v", got.Reachable)
	}
	if p, _ := path.Load().(string); p != "/charts" {
		t.Fatalf("want /charts, got %q", p)
	}
}

func TestProbeUnsupportedScheme(t *testing.T) {
	got := New(nil).Probe(context.Background(), Target{RemoteURL: "s3://my-bucket"})

	if got.Reachable.Status != metav1.ConditionUnknown {
		t.Fatalf("want Reachable Unknown, got %+v", got.Reachable)
	}
	if got.Reachable.Reason != ReasonUnsupportedScheme {
		t.Fatalf("want reason %s, got %s", ReasonUnsupportedScheme, got.Reachable.Reason)
	}
}

func TestProbeDoesNotFollowRedirects(t *testing.T) {
	var elsewhereHits atomic.Int64
	elsewhere := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		elsewhereHits.Add(1)
	}))
	defer elsewhere.Close()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, elsewhere.URL, http.StatusFound)
	}))
	defer srv.Close()

	got := proberFor(srv).Probe(context.Background(), Target{RemoteURL: srv.URL})

	if got.Reachable.Status != metav1.ConditionTrue {
		t.Fatalf("a 302 is still a response: want Reachable True, got %+v", got.Reachable)
	}
	if n := elsewhereHits.Load(); n != 0 {
		t.Fatalf("redirect target must never be contacted, got %d requests", n)
	}
}

func TestProbeRefusesDeniedTargetWithoutDialing(t *testing.T) {
	var hits atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
	}))
	defer srv.Close()

	// New(DefaultDenyCIDRs), unmodified: httptest listens on 127.0.0.1, which
	// the default deny list refuses.
	got := New(DefaultDenyCIDRs).Probe(context.Background(), Target{RemoteURL: srv.URL})

	if got.Reachable.Status != metav1.ConditionFalse {
		t.Fatalf("want Reachable False, got %+v", got.Reachable)
	}
	if got.Reachable.Reason != ReasonTargetDenied {
		t.Fatalf("want reason %s, got %s", ReasonTargetDenied, got.Reachable.Reason)
	}
	if n := hits.Load(); n != 0 {
		t.Fatalf("denied target must never be dialed, got %d requests", n)
	}
}

func TestProbeDNSFailure(t *testing.T) {
	got := New(nil).Probe(context.Background(), Target{RemoteURL: "https://arc-probe.invalid"})

	if got.Reachable.Status != metav1.ConditionFalse {
		t.Fatalf("want Reachable False, got %+v", got.Reachable)
	}
	if got.Reachable.Reason != ReasonDNSFailure {
		t.Fatalf("want reason %s, got %s", ReasonDNSFailure, got.Reachable.Reason)
	}
}

func TestAuthNoSecretIsUnknown(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()

	got := proberFor(srv).Probe(context.Background(), Target{RemoteURL: srv.URL})

	if got.Authenticated.Status != metav1.ConditionUnknown {
		t.Fatalf("want Unknown, got %+v", got.Authenticated)
	}
	if got.Authenticated.Reason != ReasonNoCredentials {
		t.Fatalf("want %s, got %s", ReasonNoCredentials, got.Authenticated.Reason)
	}
}

func TestAuthUnsupportedSecretShapeIsUnknown(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()

	// A blob Endpoint's Secret holds AWS_ACCESS_KEY_ID / AWS_SECRET_ACCESS_KEY,
	// so the controller finds no username or password to pass through.
	got := proberFor(srv).Probe(context.Background(), Target{RemoteURL: srv.URL, HasSecret: true})

	if got.Authenticated.Status != metav1.ConditionUnknown {
		t.Fatalf("want Unknown, got %+v", got.Authenticated)
	}
	if got.Authenticated.Reason != ReasonUnsupportedAuthScheme {
		t.Fatalf("want %s, got %s", ReasonUnsupportedAuthScheme, got.Authenticated.Reason)
	}
}

func TestAuthAcceptedCredentials(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, p, ok := r.BasicAuth()
		if !ok || u != "alice" || p != "s3cret" {
			w.WriteHeader(http.StatusUnauthorized)

			return
		}
	}))
	defer srv.Close()

	got := proberFor(srv).Probe(context.Background(), Target{
		RemoteURL: srv.URL, HasSecret: true, Username: "alice", Password: "s3cret",
	})

	if got.Authenticated.Status != metav1.ConditionTrue {
		t.Fatalf("want True, got %+v", got.Authenticated)
	}
	if got.Authenticated.Reason != ReasonAuthenticated {
		t.Fatalf("want %s, got %s", ReasonAuthenticated, got.Authenticated.Reason)
	}
}

func TestAuthRejectedCredentials(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	got := proberFor(srv).Probe(context.Background(), Target{
		RemoteURL: srv.URL, HasSecret: true, Username: "alice", Password: "wrong",
	})

	if got.Authenticated.Status != metav1.ConditionFalse {
		t.Fatalf("want False, got %+v", got.Authenticated)
	}
	if got.Authenticated.Reason != ReasonUnauthorized {
		t.Fatalf("want %s, got %s", ReasonUnauthorized, got.Authenticated.Reason)
	}
	if got.Reachable.Status != metav1.ConditionTrue {
		t.Fatalf("a 401 still proves reachability: %+v", got.Reachable)
	}
}

func TestAuthBearerChallengeAccepted(t *testing.T) {
	var tokenHits atomic.Int64
	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenHits.Add(1)
		if u, p, _ := r.BasicAuth(); u != "alice" || p != "s3cret" {
			w.WriteHeader(http.StatusUnauthorized)

			return
		}
		if r.URL.Query().Get("service") != "registry.example" {
			t.Errorf("challenge service param not forwarded, got %q", r.URL.RawQuery)
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"token": "abc123"})
	}))
	defer tokenSrv.Close()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("WWW-Authenticate",
			fmt.Sprintf("Bearer realm=%q,service=%q", tokenSrv.URL, "registry.example"))
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	got := proberFor(srv).Probe(context.Background(), Target{
		RemoteURL: srv.URL, HasSecret: true, Username: "alice", Password: "s3cret",
	})

	if got.Authenticated.Status != metav1.ConditionTrue {
		t.Fatalf("want True, got %+v", got.Authenticated)
	}
	if n := tokenHits.Load(); n != 1 {
		t.Fatalf("want exactly one token-service hop, got %d", n)
	}
}

func TestAuthBearerChallengeRejected(t *testing.T) {
	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer tokenSrv.Close()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("WWW-Authenticate", fmt.Sprintf("Bearer realm=%q", tokenSrv.URL))
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	got := proberFor(srv).Probe(context.Background(), Target{
		RemoteURL: srv.URL, HasSecret: true, Username: "alice", Password: "wrong",
	})

	if got.Authenticated.Status != metav1.ConditionFalse {
		t.Fatalf("want False, got %+v", got.Authenticated)
	}
	if got.Authenticated.Reason != ReasonUnauthorized {
		t.Fatalf("want %s, got %s", ReasonUnauthorized, got.Authenticated.Reason)
	}
}

func TestAuthInconclusiveResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	got := proberFor(srv).Probe(context.Background(), Target{
		RemoteURL: srv.URL, HasSecret: true, Username: "alice", Password: "s3cret",
	})

	if got.Authenticated.Status != metav1.ConditionUnknown {
		t.Fatalf("want Unknown, got %+v", got.Authenticated)
	}
	if got.Authenticated.Reason != ReasonInconclusive {
		t.Fatalf("want %s, got %s", ReasonInconclusive, got.Authenticated.Reason)
	}
}

func TestAuthRedirectIsInconclusive(t *testing.T) {
	elsewhere := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer elsewhere.Close()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, elsewhere.URL, http.StatusFound)
	}))
	defer srv.Close()

	got := proberFor(srv).Probe(context.Background(), Target{
		RemoteURL: srv.URL, HasSecret: true, Username: "alice", Password: "s3cret",
	})

	if got.Authenticated.Status != metav1.ConditionUnknown {
		t.Fatalf("want Unknown, got %+v", got.Authenticated)
	}
	if got.Authenticated.Reason != ReasonInconclusive {
		t.Fatalf("want %s, got %s", ReasonInconclusive, got.Authenticated.Reason)
	}
}

// TestAuthBearerRealmDowngradeRefused covers the case where a consumer-supplied
// Bearer challenge names an http realm for an https target. p.do attaches
// Basic auth unconditionally, so following that realm would send the
// endpoint's credentials in the clear. The realm server must never receive a
// request.
func TestAuthBearerRealmDowngradeRefused(t *testing.T) {
	var realmHits atomic.Int64
	realmSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		realmHits.Add(1)
	}))
	defer realmSrv.Close()

	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("WWW-Authenticate", fmt.Sprintf("Bearer realm=%q", realmSrv.URL))
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	got := proberFor(srv).Probe(context.Background(), Target{
		RemoteURL: srv.URL, HasSecret: true, Username: "alice", Password: "s3cret",
	})

	if got.Authenticated.Status != metav1.ConditionUnknown {
		t.Fatalf("want Unknown, got %+v", got.Authenticated)
	}
	if got.Authenticated.Reason != ReasonInconclusive {
		t.Fatalf("want %s, got %s", ReasonInconclusive, got.Authenticated.Reason)
	}
	if n := realmHits.Load(); n != 0 {
		t.Fatalf("must not send credentials to a downgraded realm, got %d requests", n)
	}
}

// nonLoopbackIP returns a local IPv4 address that DefaultDenyCIDRs does not
// deny, so a server bound to it is reachable through the guarded dialer. It
// skips the test if the host has no such interface.
func nonLoopbackIP(t *testing.T) string {
	t.Helper()

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		t.Fatalf("InterfaceAddrs: %v", err)
	}

	for _, a := range addrs {
		ipNet, ok := a.(*net.IPNet)
		if !ok {
			continue
		}

		ip4 := ipNet.IP.To4()
		if ip4 == nil {
			continue
		}

		addr, ok := netip.AddrFromSlice(ip4)
		if !ok {
			continue
		}

		if !denied(DefaultDenyCIDRs, addr) {
			return ip4.String()
		}
	}

	t.Skip("no non-loopback IPv4 interface address available")

	return ""
}

// TestAuthBearerRealmDeniedNotDialed proves the guarded dialer covers the
// Bearer hop, not just the primary request. The realm binds to loopback,
// which DefaultDenyCIDRs refuses; the primary target binds to a non-loopback
// local address so the probe still reaches it and the challenge is issued.
func TestAuthBearerRealmDeniedNotDialed(t *testing.T) {
	primaryIP := nonLoopbackIP(t)

	var tokenHits atomic.Int64
	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenHits.Add(1)
	}))
	defer tokenSrv.Close()

	primary := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("WWW-Authenticate", fmt.Sprintf("Bearer realm=%q", tokenSrv.URL))
		w.WriteHeader(http.StatusUnauthorized)
	}))
	primary.Listener.Close()

	ln, err := new(net.ListenConfig).Listen(context.Background(), "tcp", net.JoinHostPort(primaryIP, "0"))
	if err != nil {
		t.Fatal(err)
	}
	primary.Listener = ln
	primary.Start()
	defer primary.Close()

	got := New(DefaultDenyCIDRs).Probe(context.Background(), Target{
		RemoteURL: primary.URL, HasSecret: true, Username: "alice", Password: "s3cret",
	})

	if got.Authenticated.Status != metav1.ConditionUnknown {
		t.Fatalf("want Unknown, got %+v", got.Authenticated)
	}
	if got.Authenticated.Reason != ReasonInconclusive {
		t.Fatalf("want %s, got %s", ReasonInconclusive, got.Authenticated.Reason)
	}
	if n := tokenHits.Load(); n != 0 {
		t.Fatalf("denied realm must never be dialed, got %d requests", n)
	}
}

// TestAuthLongRealmIsTruncated proves a hostile target cannot grow a Check
// message without bound. The realm has no scheme, so it is rejected as
// "unusable" without any network call.
func TestAuthLongRealmIsTruncated(t *testing.T) {
	longRealm := strings.Repeat("a", 10_000)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("WWW-Authenticate", fmt.Sprintf("Bearer realm=%q", longRealm))
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	got := proberFor(srv).Probe(context.Background(), Target{
		RemoteURL: srv.URL, HasSecret: true, Username: "alice", Password: "s3cret",
	})

	if got.Authenticated.Status != metav1.ConditionUnknown {
		t.Fatalf("want Unknown, got %+v", got.Authenticated)
	}
	if got.Authenticated.Reason != ReasonInconclusive {
		t.Fatalf("want %s, got %s", ReasonInconclusive, got.Authenticated.Reason)
	}
	if n := len([]rune(got.Authenticated.Message)); n > messageTruncateLimit+64 {
		t.Fatalf("want a bounded message, got %d runes: %q", n, got.Authenticated.Message)
	}
}

func TestTruncateCutsOnRuneBoundary(t *testing.T) {
	s := strings.Repeat("é", 300) // multi-byte rune, so a byte-slice cut would corrupt it.

	got := truncate(s, 10)

	if !utf8.ValidString(got) {
		t.Fatalf("truncate produced invalid UTF-8: %q", got)
	}
	if n := len([]rune(got)); n <= 10 || n > 10+20 {
		t.Fatalf("want ~10 runes plus the truncation marker, got %d: %q", n, got)
	}
}

func TestDefaultDenyCIDRsStringRoundTrips(t *testing.T) {
	// The flag's default is rendered from DefaultDenyCIDRs so --help shows the
	// real value, and parsing it back must reproduce the same list.
	parsed, err := ParseDenyCIDRs(strings.Split(DefaultDenyCIDRsString(), ","))
	if err != nil {
		t.Fatal(err)
	}
	if len(parsed) != len(DefaultDenyCIDRs) {
		t.Fatalf("want %d prefixes, got %d", len(DefaultDenyCIDRs), len(parsed))
	}
	for i := range parsed {
		if parsed[i] != DefaultDenyCIDRs[i] {
			t.Fatalf("prefix %d: want %s, got %s", i, DefaultDenyCIDRs[i], parsed[i])
		}
	}
}

func TestParseDenyCIDRsIgnoresBlanks(t *testing.T) {
	got, err := ParseDenyCIDRs([]string{" 10.0.0.0/8 ", "", "  "})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("want 1 prefix, got %d", len(got))
	}
}
