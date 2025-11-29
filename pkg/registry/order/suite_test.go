// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package order_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestOrderRegistry(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Order Registry Suite")
}
