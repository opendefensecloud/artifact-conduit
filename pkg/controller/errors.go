// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"fmt"
	"strings"

	"github.com/go-logr/logr"
)

// errLogAndWrap is a small utility function to reduce code and be able to directly
// return errors, but wrap them and log them at the same time.
//
// The function currently capitalizes the first letter as well.
// message is expected to be NOT empty.
func errLogAndWrap(log logr.Logger, err error, message string) error {
	parts := strings.SplitN(message, "", 1)
	log.Error(err, strings.ToUpper(parts[0])+parts[1])
	return fmt.Errorf(message+": %w", err)
}
