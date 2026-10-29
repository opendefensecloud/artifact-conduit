// Copyright BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

//go:build tools

// Package hack pins the code generation tools as module dependencies.
package hack

import (
	_ "k8s.io/code-generator"
)
