// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package envtest

import (
	"errors"
	"path/filepath"
	"time"

	"github.com/ironcore-dev/controller-utils/buildutils"
	utilsenvtest "github.com/ironcore-dev/ironcore/utils/envtest"
	utilapiserver "github.com/ironcore-dev/ironcore/utils/envtest/apiserver"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

type Environment struct {
	cfg       *rest.Config
	env       *envtest.Environment
	ext       *utilsenvtest.EnvironmentExtensions
	k8sClient client.Client
	apiServer *utilapiserver.APIServer
}

// TODO: add options to override APIServiceDirectoryPaths and MainPath

func NewEnvironment() (*Environment, error) {
	env := &envtest.Environment{
		CRDDirectoryPaths: []string{filepath.Join("..", "..", "config", "crd", "external")},
	}
	ext := &utilsenvtest.EnvironmentExtensions{
		APIServiceDirectoryPaths:       []string{filepath.Join("..", "..", "config", "apiserver", "apiservice", "bases")},
		ErrorIfAPIServicePathIsMissing: true,
	}
	return &Environment{
		env: env,
		ext: ext,
	}, nil
}

func (e *Environment) Start(scheme *runtime.Scheme) (client.Client, error) {
	cfg, err := utilsenvtest.StartWithExtensions(e.env, e.ext)
	if err != nil {
		return nil, err
	}

	k8sClient, err := client.New(cfg, client.Options{Scheme: scheme})
	if err != nil {
		return nil, errors.Join(err, e.Stop())
	}

	apiServer, err := utilapiserver.New(cfg, utilapiserver.Options{
		MainPath:     "go.opendefense.cloud/arc/cmd/arc-apiserver",
		BuildOptions: []buildutils.BuildOption{buildutils.ModModeMod},
		ETCDServers:  []string{e.env.ControlPlane.Etcd.URL.String()},
		Host:         e.ext.APIServiceInstallOptions.LocalServingHost,
		Port:         e.ext.APIServiceInstallOptions.LocalServingPort,
		CertDir:      e.ext.APIServiceInstallOptions.LocalServingCertDir,
	})
	if err != nil {
		return nil, errors.Join(err, e.Stop())
	}

	if err := apiServer.Start(); err != nil {
		return nil, errors.Join(err, e.Stop())
	}

	e.cfg = cfg
	e.k8sClient = k8sClient
	e.apiServer = apiServer
	return k8sClient, nil
}

func (e *Environment) Stop() error {
	var err error
	if e.apiServer != nil {
		err = e.apiServer.Stop()
	}
	if e.ext != nil {
		err = errors.Join(err, utilsenvtest.StopWithExtensions(e.env, e.ext))
	}
	return err
}

func (e *Environment) WaitUntilReadyWithTimeout(timeout time.Duration) error {
	return utilsenvtest.WaitUntilAPIServicesReadyWithTimeout(timeout, e.ext, e.cfg, e.k8sClient, e.k8sClient.Scheme())
}

func (e *Environment) GetRESTConfig() *rest.Config {
	return e.cfg
}
