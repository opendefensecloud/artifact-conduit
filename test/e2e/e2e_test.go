// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

//go:build e2e

package e2e

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// namespace where the project is deployed in
const namespace = "arc-system"

var _ = Describe("ARC", Ordered, func() {
	var controllerPodName string
	dir, _ := getProjectDir()
	testStart := time.Now()

	// Before running the tests, set up the environment by creating the namespace,
	// enforce the restricted security policy to the namespace, installing CRDs,
	// and deploying the controller.
	BeforeAll(func() {
		By("creating arc-system namespace")
		cmd := exec.Command("kubectl", "create", "ns", namespace)
		_, err := run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to create namespace")

		// NOTE: etcd runs as root uid, so unfortunately we can not enforce this yet
		// By("labeling the namespace to enforce the restricted security policy")
		// cmd = exec.Command("kubectl", "label", "--overwrite", "ns", namespace,
		// 	"pod-security.kubernetes.io/enforce=restricted")
		// _, err = run(cmd)
		// Expect(err).NotTo(HaveOccurred(), "Failed to label namespace with restricted policy")

		By("deploying apiserver and controller-manager")
		dir, err := getProjectDir()
		Expect(err).NotTo(HaveOccurred())
		cmd = exec.Command(helmBinary, "upgrade", "--install",
			"--namespace", namespace, "arc", filepath.Join(dir, "charts", "arc"),
			"--set", "fullnameOverride=arc",
			"--set", "apiserver.image.repository=apiserver",
			"--set", "apiserver.image.tag=e2e",
			"--set", "controller.image.repository=manager",
			"--set", "controller.image.tag=e2e",
			"--set", "apiserver.args.cronMinScheduleInterval=30s")
		_, err = run(cmd)
		Expect(err).NotTo(HaveOccurred())
	})

	// After all tests have been executed, clean up by undeploying the controller, uninstalling CRDs,
	// and deleting the namespace.
	AfterAll(func() {
		By("removing all orders")
		cmd := exec.Command("kubectl", "delete", "orders", "-n", "default", "--all")
		_, _ = run(cmd)

		By("undeploying the apiserver and controller-manager")
		cmd = exec.Command(helmBinary, "uninstall", "-n", namespace, "arc")
		_, _ = run(cmd)

		By("removing manager namespace")
		cmd = exec.Command("kubectl", "delete", "ns", namespace)
		_, _ = run(cmd)
	})

	BeforeEach(func() {
		testStart = time.Now()
	})

	// After each test, check for failures and collect logs, events,
	// and pod descriptions for debugging.
	AfterEach(func() {
		specReport := CurrentSpecReport()
		if specReport.Failed() {
			By("Fetching controller manager pod logs")
			cmd := exec.Command("kubectl", "logs", controllerPodName, "-n", namespace, "--since", time.Since(testStart).String())
			controllerLogs, err := run(cmd)
			if err == nil {
				logf("Controller logs:\n %s", controllerLogs)
			} else {
				logf("Failed to get Controller logs: %s", err)
			}

			By("Fetching Kubernetes events")
			eventNamespaces := []string{namespace, "default"}
			for _, ns := range eventNamespaces {
				cmd = exec.Command("kubectl", "get", "events", "-n", ns, "--sort-by=.lastTimestamp")
				eventsOutput, err := run(cmd)
				if err == nil {
					logf("Kubernetes events (%s):\n%s", ns, eventsOutput)
				} else {
					logf("Failed to get Kubernetes events (%s): %s", ns, err)
				}
			}

			By("Fetching controller manager pod description")
			cmd = exec.Command("kubectl", "describe", "pod", controllerPodName, "-n", namespace)
			podDescription, err := run(cmd)
			if err == nil {
				fmt.Println("Pod description:\n", podDescription)
			} else {
				fmt.Println("Failed to describe controller pod")
			}
		}
	})

	SetDefaultEventuallyTimeout(10 * time.Minute)
	SetDefaultEventuallyPollingInterval(2 * time.Second)

	Context("Extension API server and Controller Manager", func() {
		It("should run successfully", func() {
			By("validating that the controller-manager pod is running as expected")
			verifyControllerUp := func(g Gomega) {
				// Get the name of the controller-manager pod
				cmd := exec.Command("kubectl", "get",
					"pods", "-l", "app.kubernetes.io/component=controller-manager",
					"-o", "go-template={{ range .items }}"+
						"{{ if not .metadata.deletionTimestamp }}"+
						"{{ .metadata.name }}"+
						"{{ \"\\n\" }}{{ end }}{{ end }}",
					"-n", namespace,
				)

				podOutput, err := run(cmd)
				g.Expect(err).NotTo(HaveOccurred(), "Failed to retrieve controller-manager pod information")
				podNames := getNonEmptyLines(podOutput)
				g.Expect(podNames).To(HaveLen(1), "expected 1 controller pod running")
				controllerPodName = podNames[0]
				g.Expect(controllerPodName).To(ContainSubstring("controller-manager"))

				// Validate the pod's status
				cmd = exec.Command("kubectl", "get",
					"pods", controllerPodName, "-o", "jsonpath={.status.phase}",
					"-n", namespace,
				)
				output, err := run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal("Running"), "Incorrect controller-manager pod status")
			}
			Eventually(verifyControllerUp).Should(Succeed())

			cmd := exec.Command("kubectl", "wait", "apiservices/v1alpha1.arc.opendefense.cloud",
				"--for", "condition=Available",
				"--timeout", waitTimeout)
			_, err := run(cmd)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should prepare default namespace for Orders", func() {
			cmd := exec.Command("kubectl", "label", "namespace", "default", "trust=enabled", "--overwrite")
			_, err := run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("Creating the ServiceAccount")
			applyResource("default", filepath.Join(dir, "test", "fixtures", "service-account.yaml"))
			By("Creating the secret for zot")
			applyResource("default", filepath.Join(dir, "test", "fixtures", "secret.yaml"))
			By("Creating a secret with the cosign key")
			applyResource("default", filepath.Join(dir, "examples", "oci", "cosign-key.yaml"))
		})

		artifactTypes := []string{"blob", "oci", "helm", "ocm"}
		for _, artifactType := range artifactTypes {
			It(fmt.Sprintf("should create orders for %s", artifactType), func() {
				By("registering the ClusterWorkflowTemplate and ClusterArtifactType")
				applyResource("", filepath.Join(dir, "examples", artifactType, "cluster-workflow-template.yaml"))
				applyResource("", filepath.Join(dir, "examples", artifactType, "artifact-type.yaml"))
				By("creating a order")
				manifest := filepath.Join(dir, "test", "fixtures", fmt.Sprintf("%s-order.yaml", artifactType))
				applyResource("default", manifest)
			})
		}

		// We create the cron here so it has time to trigger while the other tests run (cron triggers every 2 minutes)
		It("should create cron oci order successfully", func() {
			applyResource("default", filepath.Join(dir, "test", "fixtures", "oci-cron-order.yaml"))
		})

		stateFinal := func(p string) bool {
			return p == "Succeeded" || p == "Failed"
		}

		stateSucceeded := func(p string) bool {
			return p == "Succeeded"
		}

		for _, artifactType := range artifactTypes {
			It(fmt.Sprintf("should run workflows of %s order successfully", artifactType), func() {
				resourceName := fmt.Sprintf("example-%s-order", artifactType)
				Eventually(func() bool {
					return orderState(resourceName, stateFinal)
				}).Should(BeTrue())

				Expect(orderState(resourceName, stateSucceeded)).To(BeTrue())
			})
		}

		It("should run workflows of oci cron order successfully", func() {
			Eventually(func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "-n", "default", "orders", "test-oci-cron-order", "-o", "go-template={{ range .status.artifactWorkflows }}{{.succeeded}}{{ end }}")
				output, err := run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				numOutput, err := strconv.Atoi(output)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(numOutput).To(BeNumerically(">", 0))
			}).Should(Succeed())
		})
	})
})
