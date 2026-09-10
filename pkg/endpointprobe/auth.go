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

// authenticated turns the probe response into a credential verdict.
//
// Unknown is used wherever a verdict would be a guess: no credentials to test,
// credentials in a shape ARC cannot send, or a response that says nothing
// either way. Reporting False in those cases would be a lie, and the Ready
// condition treats Unknown as non-blocking for exactly that reason.
func (p *Prober) authenticated(ctx context.Context, resp *http.Response, t Target, primaryScheme string) Check {
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
		return Check{metav1.ConditionTrue, ReasonAuthenticated,
			fmt.Sprintf("credentials accepted (%s)", truncate(resp.Status, messageTruncateLimit))}

	case resp.StatusCode < http.StatusBadRequest:
		// Redirects are never followed (see New's CheckRedirect), so a 3xx is a
		// routine outcome for a target that bounces unauthenticated requests
		// elsewhere. It says nothing about the credentials: reporting True here
		// would claim a verification that never happened.
		return Check{metav1.ConditionUnknown, ReasonInconclusive,
			fmt.Sprintf("target redirected (%s); credentials were not verified", truncate(resp.Status, messageTruncateLimit))}

	case resp.StatusCode == http.StatusUnauthorized:
		if challenge := bearerChallenge(resp); challenge != "" {
			return p.bearerHop(ctx, challenge, t, primaryScheme)
		}

		return Check{metav1.ConditionFalse, ReasonUnauthorized, "credentials rejected (401)"}

	case resp.StatusCode == http.StatusForbidden:
		return Check{metav1.ConditionFalse, ReasonUnauthorized, "credentials rejected (403)"}

	default:
		return Check{metav1.ConditionUnknown, ReasonInconclusive,
			fmt.Sprintf("target responded %s, which says nothing about credential validity", truncate(resp.Status, messageTruncateLimit))}
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
// The realm is consumer-controlled, and p.do attaches Basic auth
// unconditionally, so a realm is refused when it would downgrade the
// connection the credentials travel over: an https target must not have its
// credentials sent to an http realm. Forwarding to a different host is not
// itself the problem — Docker Hub's realm is auth.docker.io, a different host
// than the registry — only a weaker scheme is.
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

	resp, err := p.do(ctx, http.MethodGet, u.String(), t)
	if err != nil {
		return Check{metav1.ConditionUnknown, ReasonInconclusive,
			fmt.Sprintf("could not reach the token service: %v", err)}
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
