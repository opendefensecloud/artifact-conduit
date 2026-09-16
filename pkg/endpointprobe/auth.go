// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package endpointprobe

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// challengeParam matches the key="value" pairs of a WWW-Authenticate header.
var challengeParam = regexp.MustCompile(`([a-zA-Z_]+)="([^"]*)"`)

// authenticated turns the anonymous probe response into a credential
// verdict. resp is the response to a request that never carried credentials
// (see Probe in probe.go): credentials are sent only from here on, and only
// once the target has actually challenged for them.
//
// Unknown is used wherever a verdict would be a guess: no credentials to
// test, credentials in a shape ARC cannot send, a response that says nothing
// either way, or a target that served the request without ever asking for
// credentials, which leaves them unexercised rather than accepted. Reporting
// False in those cases would be a lie, and the Ready condition treats
// Unknown as non-blocking for exactly that reason.
//
// A 401 without a Bearer challenge and a 403 both mean "this target gates on
// something, but the anonymous request didn't tell us what": the same URL is
// retried once with credentials attached, and that retry's response is what
// gets judged. This is also why Authenticated=True is trustworthy: it can
// only be reached by way of a challenge the target itself issued.
func (p *Prober) authenticated(ctx context.Context, resp *http.Response, t Target, u *url.URL) Check {
	switch {
	case !t.HasSecret:
		return Check{metav1.ConditionUnknown, ReasonNoCredentials,
			"endpoint has no secretRef; the target was probed anonymously"}
	case !t.usableCredentials():
		return Check{metav1.ConditionUnknown, ReasonUnsupportedAuthScheme,
			"secret has no username and password keys; ARC cannot verify credentials of this shape"}
	}

	switch {
	case resp.StatusCode < http.StatusMultipleChoices:
		// The target served the anonymous request itself. The configured
		// credentials were never sent and never exercised. Reporting True
		// here would claim a verification that never happened.
		return Check{metav1.ConditionUnknown, ReasonNotExercised,
			fmt.Sprintf("target served the request anonymously (%s); configured credentials were not sent or verified",
				truncate(resp.Status, messageTruncateLimit))}

	case resp.StatusCode < http.StatusBadRequest:
		// Redirects are never followed (see New's CheckRedirect), so a 3xx is a
		// routine outcome for a target that bounces unauthenticated requests
		// elsewhere. It says nothing about the credentials: reporting True here
		// would claim a verification that never happened.
		return Check{metav1.ConditionUnknown, ReasonInconclusive,
			fmt.Sprintf("target redirected (%s); credentials were not verified", truncate(resp.Status, messageTruncateLimit))}

	case resp.StatusCode == http.StatusUnauthorized:
		if challenge := bearerChallenge(resp); challenge != "" {
			return p.bearerHop(ctx, challenge, t, u.Scheme)
		}

		return p.authenticatedRetry(ctx, u, t)

	case resp.StatusCode == http.StatusForbidden:
		return p.authenticatedRetry(ctx, u, t)

	default:
		return Check{metav1.ConditionUnknown, ReasonInconclusive,
			fmt.Sprintf("target responded %s, which says nothing about credential validity", truncate(resp.Status, messageTruncateLimit))}
	}
}

// authenticatedRetry re-issues the request at u with credentials attached,
// after the anonymous request drew a 401 (with no Bearer challenge) or a
// 403, and judges the credentialed response. It is the one place outside
// bearerHop where the probe sends the Target's credentials, and it only runs
// because the target already demanded them once.
func (p *Prober) authenticatedRetry(ctx context.Context, u *url.URL, t Target) Check {
	resp, err := p.do(ctx, http.MethodGet, u.String(), t, true)
	if err != nil {
		return Check{metav1.ConditionUnknown, ReasonInconclusive,
			fmt.Sprintf("could not retry with credentials: %s", truncate(err.Error(), messageTruncateLimit))}
	}
	defer drainAndClose(resp)

	switch {
	case resp.StatusCode < http.StatusMultipleChoices:
		return Check{metav1.ConditionTrue, ReasonAuthenticated,
			fmt.Sprintf("credentials accepted (%s)", truncate(resp.Status, messageTruncateLimit))}

	case resp.StatusCode == http.StatusUnauthorized, resp.StatusCode == http.StatusForbidden:
		return Check{metav1.ConditionFalse, ReasonUnauthorized,
			fmt.Sprintf("credentials rejected (%s)", truncate(resp.Status, messageTruncateLimit))}

	default:
		return Check{metav1.ConditionUnknown, ReasonInconclusive,
			fmt.Sprintf("target responded %s after retrying with credentials, which says nothing about credential validity",
				truncate(resp.Status, messageTruncateLimit))}
	}
}

func bearerChallenge(resp *http.Response) string {
	for _, v := range resp.Header.Values("WWW-Authenticate") {
		if strings.HasPrefix(strings.ToLower(v), "bearer ") {
			return v
		}
	}

	return ""
}

func challengeParams(challenge string) map[string]string {
	out := map[string]string{}
	for _, m := range challengeParam.FindAllStringSubmatch(challenge, -1) {
		out[m[1]] = m[2]
	}

	return out
}

// bearerHop performs the single token exchange Docker-style registries require:
// the 401 names a realm, and credentials that the realm accepts yield a token.
// gcr.io, quay.io and ghcr.io all answer this way, so without this hop
// Authenticated would be permanently Unknown for the OCI type.
//
// It is deliberately one hop and no more. The realm is requested directly,
// redirects are still not followed, and the realm host passes through the same
// guarded dialer as the primary target.
//
// The realm is consumer-controlled, and this hop always requests p.do attach
// Basic auth (the 401 that got us here is the challenge), so a realm is
// refused when it would downgrade the connection the credentials travel
// over: an https target must not have its credentials sent to an http
// realm. Forwarding to a different host is not itself the problem — Docker
// Hub's realm is auth.docker.io, a different host than the registry — only a
// weaker scheme is.
func (p *Prober) bearerHop(ctx context.Context, challenge string, t Target, primaryScheme string) Check {
	params := challengeParams(challenge)

	realm := params["realm"]
	if realm == "" {
		return Check{metav1.ConditionUnknown, ReasonInconclusive,
			"target returned a Bearer challenge without a realm"}
	}

	u, err := url.Parse(realm)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return Check{metav1.ConditionUnknown, ReasonInconclusive,
			fmt.Sprintf("target returned an unusable Bearer realm %q", truncate(realm, messageTruncateLimit))}
	}

	if primaryScheme == "https" && u.Scheme != "https" {
		return Check{metav1.ConditionUnknown, ReasonInconclusive,
			fmt.Sprintf("refusing to send credentials to Bearer realm %q: it downgrades from https to %s",
				truncate(realm, messageTruncateLimit), u.Scheme)}
	}

	q := u.Query()
	for _, k := range []string{"service", "scope"} {
		if v := params[k]; v != "" {
			q.Set(k, v)
		}
	}
	u.RawQuery = q.Encode()

	resp, err := p.do(ctx, http.MethodGet, u.String(), t, true)
	if err != nil {
		return Check{metav1.ConditionUnknown, ReasonInconclusive,
			fmt.Sprintf("could not reach the token service: %v", truncate(err.Error(), messageTruncateLimit))}
	}
	defer drainAndClose(resp)

	switch {
	case resp.StatusCode == http.StatusUnauthorized, resp.StatusCode == http.StatusForbidden:
		return Check{metav1.ConditionFalse, ReasonUnauthorized,
			fmt.Sprintf("token service rejected the credentials (%s)", truncate(resp.Status, messageTruncateLimit))}
	case resp.StatusCode >= http.StatusBadRequest:
		return Check{metav1.ConditionUnknown, ReasonInconclusive,
			fmt.Sprintf("token service responded %s", truncate(resp.Status, messageTruncateLimit))}
	}

	var body struct {
		Token       string `json:"token"`
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxBodyBytes)).Decode(&body); err != nil {
		return Check{metav1.ConditionUnknown, ReasonInconclusive,
			"token service returned a body that could not be read"}
	}
	if body.Token == "" && body.AccessToken == "" {
		return Check{metav1.ConditionUnknown, ReasonInconclusive,
			"token service returned no token"}
	}

	return Check{metav1.ConditionTrue, ReasonAuthenticated,
		"credentials accepted by the token service"}
}
