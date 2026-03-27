// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

//go:build e2e

package e2e

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	certmanagerVersion = "v1.19.1"
	certmanagerChart   = "oci://quay.io/jetstack/charts/cert-manager"

	trustmanagerChart   = "oci://quay.io/jetstack/charts/trust-manager"
	trustmanagerVersion = "v0.20.2"

	argoWorkflowsVersion = "v3.7.4"
	argoWorkflowsURLTmpl = "https://github.com/argoproj/argo-workflows/releases/download/%s/quick-start-minimal.yaml"

	minioRepoUrl = "https://charts.min.io"

	zotRepoURL = "https://zotregistry.dev/helm-charts"

	apiserverImage = "apiserver:e2e"
	managerImage   = "manager:e2e"

	waitTimeout = "5m"
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
	helmBinary = func() string {
		if v, ok := os.LookupEnv("HELM"); ok {
			return v
		} else {
			return "helm"
		}
	}()

	kubeConfigPath = ""
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

	logf("Installing CertManager...\n")
	Expect(installCertManager()).To(Succeed(), "Failed to install CertManager")

	logf("Installing TrustManager...\n")
	Eventually(installTrustManager).Should(Succeed(), "Failed to install TrustManager")

	logf("Installing Argo Workflows...\n")
	Expect(installArgoWorkflows()).To(Succeed(), "Failed to install Argo Workflows")

	logf("Installing Zot...\n")
	Expect(installZot()).To(Succeed(), "Failed to install Argo Workflows")

	logf("Installing Minio...\n")
	Expect(installMinio()).To(Succeed(), "Failed to install Minio")
})

var _ = AfterSuite(func() {
	cmd := exec.Command("kubectl", "delete", "namespace", "zot")
	_, _ = run(cmd)

	cmd = exec.Command("kubectl", "delete", "namespace", "minio")
	_, _ = run(cmd)

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

func installMinio() error {
	dir, err := getProjectDir()
	Expect(err).NotTo(HaveOccurred())

	cmd := exec.Command(helmBinary, "upgrade", "--install", "--create-namespace", "--namespace=minio", fmt.Sprintf("--repo=%s", minioRepoUrl), "-f", filepath.Join(dir, "test", "fixtures", "dst-minio.yaml"), "dst", "minio")
	_, err = run(cmd)
	Expect(err).NotTo(HaveOccurred())

	return err
}

func installZot() error {
	dir, err := getProjectDir()

	Expect(err).NotTo(HaveOccurred())
	cmd := exec.Command(helmBinary, "upgrade", "--install", "--create-namespace", "--namespace=zot", fmt.Sprintf("--repo=%s", zotRepoURL), "-f", filepath.Join(dir, "test", "fixtures", "dst-zot.yaml"), "dst", "zot")
	_, err = run(cmd)
	Expect(err).NotTo(HaveOccurred())

	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Failed to install zot")
	cmd = exec.Command("kubectl", "apply", "-f", filepath.Join(dir, "test", "fixtures", "zot-cert.yaml"))
	_, err = run(cmd)

	return err
}

func installTrustManager() error {
	cmd := exec.Command(helmBinary, "upgrade", "--install", "--create-namespace", "--namespace=cert-manager", "trust-manager", trustmanagerChart, "--version", trustmanagerVersion)
	if _, err := run(cmd); err != nil {
		return err
	}

	// Wait for trust-manager to be ready, which can take time if trust-manager
	// was re-installed after uninstalling on a cluster.
	cmd = exec.Command("kubectl", "wait", "deployment.apps/trust-manager",
		"--for", "condition=Available",
		"--namespace", "cert-manager",
		"--timeout", "5m",
	)
	if _, err := run(cmd); err != nil {
		return err
	}

	dir, err := getProjectDir()
	if err != nil {
		return err
	}

	cmd = exec.Command("kubectl", "apply", "-f", filepath.Join(dir, "test", "fixtures", "trustmanager.yaml"))
	_, err = run(cmd)

	return err
}

// installCertManager installs the cert manager bundle.
func installCertManager() error {
	cmd := exec.Command(helmBinary, "upgrade", "--install", "cert-manager", certmanagerChart, "--version", certmanagerVersion, "--namespace", "cert-manager", "--create-namespace", "--set", "crds.enabled=true")
	if _, err := run(cmd); err != nil {
		return err
	}

	// The helm chart waits until cert-manager is fully functional, so no further tests required.

	dir, err := getProjectDir()
	Expect(err).NotTo(HaveOccurred())
	cmd = exec.Command("kubectl", "apply", "-f", filepath.Join(dir, "test", "fixtures", "certmanager.yaml"))
	_, err = run(cmd)

	return err
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

func logf(format string, a ...any) {
	_, _ = fmt.Fprintf(GinkgoWriter, format, a...)
}
