// Copyright BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

// Package tools

//go:build tools
// +build tools

package hack

import (
	_ "k8s.io/code-generator"
)
