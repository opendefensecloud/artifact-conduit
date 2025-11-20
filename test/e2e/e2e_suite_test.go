// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

//go:build e2e
// +build e2e

package e2e

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	certmanagerVersion = "v1.19.1"
	certmanagerURLTmpl = "https://github.com/cert-manager/cert-manager/releases/download/%s/cert-manager.yaml"

	argoWorkflowsVersion = "v3.7.4"
	argoWorkflowsURLTmpl = "https://github.com/argoproj/argo-workflows/releases/download/%s/quick-start-minimal.yaml"

	apiserverImage = "apiserver:e2e"
	managerImage   = "manager:e2e"
)

var (
	kindBinary = func() string {
		if v, ok := os.LookupEnv("KIND"); ok {
			return v
		} else {
			return "kind"
		}
	}()
	kindCluster = func() string {
		if v, ok := os.LookupEnv("KIND_CLUSTER"); ok {
			return v
		} else {
			return "kind"
		}
	}()

	kubeConfigPath = ""

	isCertManagerAlreadyInstalled   = false
	isArgoWorkflowsAlreadyInstalled = false
)

// TestE2E runs the end-to-end (e2e) test suite for the project. These tests execute in an isolated,
// temporary environment to validate project changes with the purpose of being used in CI jobs.
// The default setup requires Kind, builds/loads the Manager Docker image locally, and installs
// CertManager.
func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	logf("Starting project-v4 integration test suite\n")
	RunSpecs(t, "e2e suite")
}

var _ = BeforeSuite(func() {
	// Let's retrieve the kubeconfig of the kind cluster
	By("fetching the kubeconfig from kind")
	f, err := os.CreateTemp("", "e2e-kubeconfig")
	Expect(err).NotTo(HaveOccurred())
	defer f.Close()
	cmd := exec.Command(kindBinary, "get", "kubeconfig", fmt.Sprintf("--name=%s", kindCluster))
	kc, err := run(cmd)
	Expect(err).NotTo(HaveOccurred())
	_, err = f.WriteString(kc)
	Expect(err).NotTo(HaveOccurred())
	f.Sync()
	kubeConfigPath = f.Name()

	// Build images
	By("building the apiserver image")
	cmd = exec.Command("make", "docker-build-apiserver", fmt.Sprintf("APISERVER_IMG=%s", apiserverImage))
	_, err = run(cmd)
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Failed to build the apiserver image")

	By("building the manager image")
	cmd = exec.Command("make", "docker-build-manager", fmt.Sprintf("MANAGER_IMG=%s", managerImage))
	_, err = run(cmd)
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Failed to build the manager image")

	// Load images
	By("loading the apiserver image on Kind")
	err = loadImageToKindClusterWithName(apiserverImage)
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Failed to load the apiserver image into Kind")

	By("loading the manager image on Kind")
	err = loadImageToKindClusterWithName(managerImage)
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Failed to load the manager image into Kind")

	By("checking if cert manager is installed already")
	isCertManagerAlreadyInstalled = areCRDsInstalled(
		"certificates.cert-manager.io",
		"issuers.cert-manager.io",
		"clusterissuers.cert-manager.io",
		"certificaterequests.cert-manager.io",
		"orders.acme.cert-manager.io",
		"challenges.acme.cert-manager.io",
	)
	if !isCertManagerAlreadyInstalled {
		logf("Installing CertManager...\n")
		Expect(installCertManager()).To(Succeed(), "Failed to install CertManager")
	} else {
		logf("WARNING: CertManager is already installed. Skipping installation...\n")
	}

	isArgoWorkflowsAlreadyInstalled = areCRDsInstalled(
		"workflows.argoproj.io",
		"workflowtemplates.argoproj.io",
		"clusterworkflowtemplates.argoproj.io",
	)
	if !isArgoWorkflowsAlreadyInstalled {
		logf("Installing Argo Workflows...\n")
		Expect(installArgoWorkflows()).To(Succeed(), "Failed to install Argo Workflows")
	} else {
		logf("WARNING: Argo Workflows is already installed. Skipping installation...\n")
	}
})

var _ = AfterSuite(func() {
	logf("Uninstalling CertManager...\n")
	if !isCertManagerAlreadyInstalled {
		uninstallCertManager()
	}
	if !isArgoWorkflowsAlreadyInstalled {
		uninstallArgoWorkflows()
	}
	if kubeConfigPath != "" {
		os.Remove(kubeConfigPath)
	}
})

// ------------------------------- HELPER -------------------------------------

// run executes the provided command within this context
func run(cmd *exec.Cmd) (string, error) {
	dir, _ := getProjectDir()
	cmd.Dir = dir

	if err := os.Chdir(cmd.Dir); err != nil {
		logf("chdir dir: %q\n", err)
	}

	cmd.Env = append(os.Environ(), "GO111MODULE=on", fmt.Sprintf("KUBECONFIG=%s", kubeConfigPath))
	command := strings.Join(cmd.Args, " ")
	logf("running: %q\n", command)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("%q failed with error %q: %w", command, string(output), err)
	}

	return string(output), nil
}

// loadImageToKindClusterWithName loads a local docker image to the kind cluster
func loadImageToKindClusterWithName(name string) error {
	kindOptions := []string{"load", "docker-image", name, "--name", kindCluster}
	cmd := exec.Command(kindBinary, kindOptions...)
	_, err := run(cmd)
	return err
}

func areCRDsInstalled(crds ...string) bool {
	cmd := exec.Command("kubectl", "get", "crds")
	output, err := run(cmd)
	if err != nil {
		return false
	}

	crdList := getNonEmptyLines(output)
	for _, crd := range crds {
		for _, line := range crdList {
			if strings.Contains(line, crd) {
				return true
			}
		}
	}

	return false
}

// installCertManager installs the cert manager bundle.
func installCertManager() error {
	url := fmt.Sprintf(certmanagerURLTmpl, certmanagerVersion)
	cmd := exec.Command("kubectl", "apply", "-f", url)
	if _, err := run(cmd); err != nil {
		return err
	}
	// Wait for cert-manager-webhook to be ready, which can take time if cert-manager
	// was re-installed after uninstalling on a cluster.
	cmd = exec.Command("kubectl", "wait", "deployment.apps/cert-manager-webhook",
		"--for", "condition=Available",
		"--namespace", "cert-manager",
		"--timeout", "5m",
	)

	_, err := run(cmd)
	return err
}

func uninstallCertManager() {
	url := fmt.Sprintf(certmanagerURLTmpl, certmanagerVersion)
	cmd := exec.Command("kubectl", "delete", "-f", url)
	if _, err := run(cmd); err != nil {
		warnError(err)
	}

	// Delete leftover leases in kube-system (not cleaned by default)
	kubeSystemLeases := []string{
		"cert-manager-cainjector-leader-election",
		"cert-manager-controller",
	}
	for _, lease := range kubeSystemLeases {
		cmd = exec.Command("kubectl", "delete", "lease", lease,
			"-n", "kube-system", "--ignore-not-found", "--force", "--grace-period=0")
		if _, err := run(cmd); err != nil {
			warnError(err)
		}
	}
}

func installArgoWorkflows() error {
	url := fmt.Sprintf(argoWorkflowsURLTmpl, argoWorkflowsVersion)
	cmd := exec.Command("kubectl", "create", "namespace", "argo")
	if _, err := run(cmd); err != nil {
		// Namespace might already exist, ignore the error
		logf("Note: namespace creation returned: %v (may already exist)\n", err)
	}

	cmd = exec.Command("kubectl", "apply", "-n", "argo", "-f", url)
	if _, err := run(cmd); err != nil {
		return err
	}

	// Wait for argo-server deployment to be ready
	cmd = exec.Command("kubectl", "wait", "deployment.apps/workflow-controller",
		"--for", "condition=Available",
		"--namespace", "argo",
		"--timeout", "5m",
	)

	_, err := run(cmd)
	return err
}

func uninstallArgoWorkflows() {
	url := fmt.Sprintf(argoWorkflowsURLTmpl, argoWorkflowsVersion)
	cmd := exec.Command("kubectl", "delete", "-n", "argo", "-f", url)
	if _, err := run(cmd); err != nil {
		warnError(err)
	}

	// Delete the argo namespace
	cmd = exec.Command("kubectl", "delete", "namespace", "argo", "--ignore-not-found", "--force", "--grace-period=0")
	if _, err := run(cmd); err != nil {
		warnError(err)
	}
}

// getNonEmptyLines converts given command output string into individual objects
// according to line breakers, and ignores the empty elements in it.
func getNonEmptyLines(output string) []string {
	var res []string
	elements := strings.Split(output, "\n")
	for _, element := range elements {
		if element != "" {
			res = append(res, element)
		}
	}

	return res
}

// getProjectDir will return the directory where the project is
func getProjectDir() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return wd, fmt.Errorf("failed to get current working directory: %w", err)
	}
	wd = strings.ReplaceAll(wd, "/test/e2e", "")
	return wd, nil
}

func warnError(err error) {
	logf("warning: %v\n", err)
}

func logf(format string, a ...any) {
	_, _ = fmt.Fprintf(GinkgoWriter, format, a...)
}
