// Copyright 2025 BWI GmbH & Artifact Conduit Contributors
// SPDX-License-Identifier: Apache-2.0

// Package tools

//go:build tools
// +build tools

package hack

import (
	_ "k8s.io/code-generator"
)
