// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"fmt"
	"strings"

	"github.com/go-logr/logr"
)

// errLogAndWrap is a small utility function to reduce code and be able to directly
// return errors, but wrap them and log them at the same time. It also capitalizes the
// first letter as well. Short texts will be handled.
func errLogAndWrap(log logr.Logger, err error, text string) error {
	textLen := len(text)
	switch textLen {
	case 0:
		return err
	case 1:
		logText := strings.ToUpper(text)
		log.Error(err, logText)
		return fmt.Errorf(text+": %w", err)
	default:
		logText := strings.ToUpper(text[:1]) + text[1:]
		log.Error(err, logText)
		return fmt.Errorf(text+": %w", err)
	}
}
