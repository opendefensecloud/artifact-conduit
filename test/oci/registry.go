//go:build !release

// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package oci

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"

	"github.com/google/go-containerregistry/pkg/registry"
)

// Registry basic http test server to mock an oci registry
type Registry struct {
	*httptest.Server

	wantedAuthHeader      string
	dockerRegistryHandler http.Handler
}

// NewRegistry creates a new http test server to mock an oci registry
func NewRegistry() *Registry {
	r := &Registry{}
	r.dockerRegistryHandler = registry.New()
	r.Server = httptest.NewServer(http.HandlerFunc(r.root))
	return r
}

func (r *Registry) root(w http.ResponseWriter, req *http.Request) {
	if r.wantedAuthHeader != "" && req.Header.Get("Authorization") != r.wantedAuthHeader {
		w.Header().Set("Www-Authenticate", `Basic realm="Test Server"`)
		w.WriteHeader(http.StatusUnauthorized)
	}
	r.dockerRegistryHandler.ServeHTTP(w, req)
}

// WithAuth sets the authorization header for the http test server
func (r *Registry) WithAuth(username string, password string) *Registry {
	r.wantedAuthHeader = "Basic " + base64.StdEncoding.EncodeToString([]byte(username+":"+password))
	return r
}
