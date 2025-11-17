// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

//go:build !release

package oci

import (
	"crypto/tls"
	"net/http"

	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/retry"
)

func NewRepo(repository string) (*remote.Repository, error) {
	repo, err := remote.NewRepository(repository)
	if err != nil {
		return nil, err
	}
	return repo, nil
}

func WithInsecureAuth(repo *remote.Repository, user, password string) *remote.Repository {
	client := retry.DefaultClient
	client.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	repo.Client = &auth.Client{
		Client: client,
		Cache:  auth.NewCache(),
		Credential: auth.StaticCredential(repo.Reference.Registry, auth.Credential{
			Username: user,
			Password: password,
		}),
	}
	return repo
}
