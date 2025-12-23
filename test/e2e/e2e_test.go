// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

//go:build e2e

package e2e

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// namespace where the project is deployed in
const namespace = "arc-system"

var _ = Describe("ARC", Ordered, func() {
	var controllerPodName string
	dir, _ := getProjectDir()

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
		cmd = exec.Command("kubectl", "apply", "-n", namespace, "-k", filepath.Join(dir, "test", "fixtures"))
		cmd = exec.Command(helmBinary, "upgrade", "--install",
			"--namespace", namespace, "arc", filepath.Join(dir, "charts", "arc"),
			"--set", "fullnameOverride=arc",
			"--set", "apiserver.image.repository=apiserver",
			"--set", "apiserver.image.tag=e2e",
			"--set", "controller.image.repository=manager",
			"--set", "controller.image.tag=e2e")
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

	// After each test, check for failures and collect logs, events,
	// and pod descriptions for debugging.
	AfterEach(func() {
		specReport := CurrentSpecReport()
		if specReport.Failed() {
			By("Fetching controller manager pod logs")
			cmd := exec.Command("kubectl", "logs", controllerPodName, "-n", namespace)
			controllerLogs, err := run(cmd)
			if err == nil {
				logf("Controller logs:\n %s", controllerLogs)
			} else {
				logf("Failed to get Controller logs: %s", err)
			}

			By("Fetching Kubernetes events")
			cmd = exec.Command("kubectl", "get", "events", "-n", namespace, "--sort-by=.lastTimestamp")
			eventsOutput, err := run(cmd)
			if err == nil {
				logf("Kubernetes events:\n%s", eventsOutput)
			} else {
				logf("Failed to get Kubernetes events: %s", err)
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

			verifyAPIServicesAvailable := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "apiservices", "v1alpha1.arc.opendefense.cloud", "-o", "go-template={{ range .status.conditions }}{{ if eq .type \"Available\" }}{{ .status }}{{ end }}{{ end }}")
				output, err := run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal("True"))
			}
			Eventually(verifyAPIServicesAvailable).Should(Succeed())
		})

		It("should create oci workflowtemplate and artifact type", func() {
			cmd := exec.Command("kubectl", "apply", "-n", namespace, "-f", filepath.Join(dir, "examples", "oci", "cluster-workflow-template.yaml"))
			_, err := run(cmd)
			Expect(err).NotTo(HaveOccurred())

			cmd = exec.Command("kubectl", "apply", "-n", namespace, "-f", filepath.Join(dir, "examples", "oci", "artifact-type.yaml"))
			_, err = run(cmd)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should create helm workflowtemplate and artifact type", func() {
			cmd := exec.Command("kubectl", "apply", "-n", namespace, "-f", filepath.Join(dir, "examples", "helm", "cluster-workflow-template.yaml"))
			_, err := run(cmd)
			Expect(err).NotTo(HaveOccurred())

			cmd = exec.Command("kubectl", "apply", "-n", namespace, "-f", filepath.Join(dir, "examples", "helm", "artifact-type.yaml"))
			_, err = run(cmd)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should create blob workflowtemplate and artifact type", func() {
			cmd := exec.Command("kubectl", "apply", "-n", namespace, "-f", filepath.Join(dir, "examples", "blob", "cluster-workflow-template.yaml"))
			_, err := run(cmd)
			Expect(err).NotTo(HaveOccurred())

			cmd = exec.Command("kubectl", "apply", "-n", namespace, "-f", filepath.Join(dir, "examples", "blob", "artifact-type.yaml"))
			_, err = run(cmd)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should prepare default namespace for Argo Workflows", func() {
			cmd := exec.Command("kubectl", "label", "namespace", "default", "trust=enabled", "--overwrite")
			_, err := run(cmd)
			Expect(err).NotTo(HaveOccurred())

			cmd = exec.Command("kubectl", "apply", "-n", "default", "-f", filepath.Join(dir, "test", "fixtures", "service-account.yaml"))
			_, err = run(cmd)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should run workflows of blob order successfully", func() {
			cmd := exec.Command("kubectl", "apply", "-n", "default", "-f", filepath.Join(dir, "test", "fixtures", "secret.yaml"))
			_, err := run(cmd)
			Expect(err).NotTo(HaveOccurred())

			cmd = exec.Command("kubectl", "apply", "-n", "default", "-f", filepath.Join(dir, "examples", "blob", "order-and-endpoints.yaml"))
			_, err = run(cmd)
			Expect(err).NotTo(HaveOccurred())

			verifyOrderSuccessful := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "-n", "default", "orders", "example-blob-order", "-o", "go-template={{ range .status.artifactWorkflows }}{{.phase}}{{ end }}")
				output, err := run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal("SucceededSucceeded"))
			}
			Eventually(verifyOrderSuccessful).Should(Succeed())
		})

		It("should run workflows of helm order successfully", func() {
			cmd := exec.Command("kubectl", "apply", "-n", "default", "-f", filepath.Join(dir, "test", "fixtures", "secret.yaml"))
			_, err := run(cmd)
			Expect(err).NotTo(HaveOccurred())

			cmd = exec.Command("kubectl", "apply", "-n", "default", "-f", filepath.Join(dir, "examples", "helm", "order-and-endpoints.yaml"))
			_, err = run(cmd)
			Expect(err).NotTo(HaveOccurred())

			verifyOrderSuccessful := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "-n", "default", "orders", "example-helm-order", "-o", "go-template={{ range .status.artifactWorkflows }}{{.phase}}{{ end }}")
				output, err := run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal("SucceededSucceeded")) // two artifacts are ordered
			}
			Eventually(verifyOrderSuccessful).Should(Succeed())
		})

		It("should run workflows of oci order successfully", func() {
			cmd := exec.Command("kubectl", "apply", "-n", "default", "-f", filepath.Join(dir, "test", "fixtures", "secret.yaml"))
			_, err := run(cmd)
			Expect(err).NotTo(HaveOccurred())

			cmd = exec.Command("kubectl", "apply", "-n", "default", "-f", filepath.Join(dir, "examples", "oci", "cosign-key.yaml"))
			_, err = run(cmd)
			Expect(err).NotTo(HaveOccurred())

			cmd = exec.Command("kubectl", "apply", "-n", "default", "-f", filepath.Join(dir, "examples", "oci", "order-and-endpoints.yaml"))
			_, err = run(cmd)
			Expect(err).NotTo(HaveOccurred())

			verifyOrderSuccessful := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "-n", "default", "orders", "example-oci-order", "-o", "go-template={{ range .status.artifactWorkflows }}{{.phase}}{{ end }}")
				output, err := run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal("Succeeded"))
			}
			Eventually(verifyOrderSuccessful).Should(Succeed())
		})
	})
})
