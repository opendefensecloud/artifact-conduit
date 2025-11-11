// Copyright 2025 BWI GmbH and Artefact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"

	"github.com/opendefensecloud/artifact-conduit/pkg/apiserver"
	genericapiserver "k8s.io/apiserver/pkg/server"
	"k8s.io/component-base/cli"
)

func main() {
	ctx := genericapiserver.SetupSignalContext()
	options := apiserver.NewARCServerOptions(os.Stdout, os.Stderr)
	cmd := apiserver.NewCommandStartARCServer(ctx, options, false)
	code := cli.Run(cmd)
	os.Exit(code)
}
