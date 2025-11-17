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

	auth                  string
	dockerRegistryHandler http.Handler
}

func NewRegistry() *Registry {
	r := &Registry{}
	r.dockerRegistryHandler = registry.New()
	r.Server = httptest.NewServer(http.HandlerFunc(r.root))
	return r
}

func (r *Registry) root(res http.ResponseWriter, req *http.Request) {
	if req.RequestURI != "/v2/" &&
		len(r.auth) != 0 &&
		req.Header.Get("Authorization") != r.auth {
		res.WriteHeader(401)
		return
	}

	r.dockerRegistryHandler.ServeHTTP(res, req)
}

func (r *Registry) WithAuth(username string, password string) *Registry {
	r.auth = "Basic " + base64.StdEncoding.EncodeToString([]byte(username+":"+password))
	return r
}
