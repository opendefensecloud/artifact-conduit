// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"errors"

	"github.com/go-logr/logr"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("errLogAndWrap", func() {
	var log logr.Logger

	BeforeEach(func() {
		log = logr.Discard()
	})

	It("should return error unchanged when text is empty", func() {
		originalErr := errors.New("original error")
		result := errLogAndWrap(log, originalErr, "")
		Expect(result).To(Equal(originalErr))
	})

	It("should handle single character text", func() {
		originalErr := errors.New("original error")
		result := errLogAndWrap(log, originalErr, "x")
		Expect(result.Error()).To(Equal("x: original error"))
	})

	It("should wrap error with normal text", func() {
		originalErr := errors.New("original error")
		result := errLogAndWrap(log, originalErr, "failed to process")
		Expect(result.Error()).To(Equal("failed to process: original error"))
	})

	It("should preserve error chain for unwrapping", func() {
		originalErr := errors.New("original error")
		result := errLogAndWrap(log, originalErr, "wrapper text")
		Expect(errors.Unwrap(result)).To(Equal(originalErr))
	})

	It("should handle two character text", func() {
		originalErr := errors.New("test")
		result := errLogAndWrap(log, originalErr, "ab")
		Expect(result.Error()).To(Equal("ab: test"))
	})
})
