// Copyright 2025 BWI GmbH and Artefact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/viper"
)

type ArtifactType string

const (
	AT_OCI  ArtifactType = "oci"
	AT_HELM ArtifactType = "helm"
)

// ArcctlConfig represents the configuration for arcctl
type ArcctlConfig struct {
	Type ArtifactType `mapstructure:"type"`
	Src  Endpoint     `mapstructure:"src"`
	Dst  Endpoint     `mapstructure:"dst"`
	Spec any          `mapstructure:"spec"`
}

func (c *ArcctlConfig) GetOCISpec() *OCISpec {
	if s, ok := c.Spec.(*OCISpec); ok {
		return s
	}
	return nil
}

// Endpoint represents a source or destination endpoint configuration
type Endpoint struct {
	Type      ArtifactType `mapstructure:"type"`
	RemoteURL string       `mapstructure:"remoteURL"`
	Auth      any          `mapstructure:"auth"`
}

func (e *Endpoint) GetOCIAuth() *OCIAuth {
	if a, ok := e.Auth.(*OCIAuth); ok {
		return a
	}
	return nil
}

type OCISpec struct {
	Image    string `mapstructure:"image"`
	Override string `mapstructure:"override"`
}

type OCIAuth struct {
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
}

func parseOCIAuth(e *Endpoint) error {
	if e.Auth == nil {
		return nil
	}
	ociAuth := &OCIAuth{}
	bytes, err := json.Marshal(e.Auth)
	if err != nil {
		return fmt.Errorf("failed to parse oci auth: %w", err)
	}
	if err := json.Unmarshal(bytes, ociAuth); err != nil {
		return err
	}
	e.Auth = ociAuth
	return nil
}

func (c *ArcctlConfig) parseOCISpec() error {
	if c.Spec == nil {
		return fmt.Errorf("spec must not be empty")
	}
	ociSpec := &OCISpec{}
	bytes, err := json.Marshal(c.Spec)
	if err != nil {
		return fmt.Errorf("failed to parse oci auth: %w", err)
	}
	if err := json.Unmarshal(bytes, ociSpec); err != nil {
		return err
	}
	c.Spec = ociSpec
	return nil
}

func (c *ArcctlConfig) parseEndpointAuth() error {
	switch c.Src.Type {
	case AT_OCI:
		if err := parseOCIAuth(&c.Src); err != nil {
			return err
		}
	}
	switch c.Dst.Type {
	case AT_OCI:
		if err := parseOCIAuth(&c.Src); err != nil {
			return err
		}
	}
	return nil
}

func (c *ArcctlConfig) parseOCI() error {
	if err := c.parseOCISpec(); err != nil {
		return err
	}
	if err := c.parseEndpointAuth(); err != nil {
		return err
	}
	return nil
}

func (c *ArcctlConfig) parseTypedSpec() error {
	if c.Spec == nil {
		return nil
	}
	switch c.Type {
	case AT_OCI:
		return c.parseOCI()
	}
	return fmt.Errorf("type %s unknown", c.Type)
}

// LoadFromViper loads the arcctl configuration from viper
func LoadFromViper() (*ArcctlConfig, error) {
	var config ArcctlConfig
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}
	if err := config.parseTypedSpec(); err != nil {
		return nil, err
	}
	return &config, nil
}

func (c *ArcctlConfig) Validate(t ArtifactType) error {
	if c.Type != t {
		return fmt.Errorf("Config is of type %s and expected type is %s", c.Type, t)
	}
	return nil
}
