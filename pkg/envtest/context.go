// Copyright 2025 BWI GmbH and Artefact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package envtest

import (
	"context"

	"github.com/ironcore-dev/ironcore/utils/testing"
)

func Context() context.Context {
	return testing.SetupContext()
}
