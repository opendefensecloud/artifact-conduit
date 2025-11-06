// Copyright 2025 BWI GmbH and Artefact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"

	"gitlab.opencode.de/bwi/ace/artifact-conduit/internal/apiserver"
	genericapiserver "k8s.io/apiserver/pkg/server"
	"k8s.io/component-base/cli"
)

func main() {
	ctx := genericapiserver.SetupSignalContext()
	options := apiserver.NewIronCoreAPIServerOptions()
	cmd := apiserver.NewCommandStartIronCoreAPIServer(ctx, options)
	code := cli.Run(cmd)
	os.Exit(code)
}
