// Copyright BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

//go:build e2e

package e2e

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// endpointField renders one go-template against an Endpoint, which keeps these
// specs to kubectl like the rest of the suite. A field that is not set renders as
// "<no value>", normalised to empty here: "not set" is a verdict several of these
// specs assert, so it has to be expressible.
func endpointField(name, template string) string {
	GinkgoHelper()

	cmd := exec.Command("kubectl", "get", "-n", "default", "endpoints.arc.opendefense.cloud", name,
		"-o", "go-template="+template)
	output, err := run(cmd)
	Expect(err).NotTo(HaveOccurred())

	if trimmed := strings.TrimSpace(output); trimmed != "<no value>" {
		return trimmed
	}

	return ""
}

// endpointVerdict reports one condition as "Status/Reason", or "absent".
func endpointVerdict(name, conditionType string) string {
	GinkgoHelper()

	verdict := endpointField(name, fmt.Sprintf(
		`{{ if .status.conditions }}{{ range .status.conditions }}`+
			`{{ if eq .type %q }}{{ .status }}/{{ .reason }}{{ end }}`+
			`{{ end }}{{ end }}`, conditionType))
	if verdict == "" {
		return "absent"
	}

	return verdict
}

// endpointControllerSpecs covers the one controller here whose entire job is an
// outbound side effect, so what these specs watch is when it makes one and when it
// does not. The unit suite stubs the probe and never opens a socket; this is the
// same gate against the in-cluster registry, through the real aggregated apiserver.
//
// Called from inside the Ordered "ARC" container rather than declared as a
// container of its own: Ginkgo randomises top-level containers, and these need the
// chart that container's BeforeAll installs.
func endpointControllerSpecs() {
	Context("EndpointController", func() {
		// internal comes from the OCI order fixtures: type oci, PushOnly, pointing
		// at the in-cluster zot with credentials in dst-reg-secret.
		const probed = "internal"

		lastProbe := func() string {
			return endpointField(probed, "{{ .status.lastProbeTime }}")
		}

		BeforeAll(func() {
			dir, err := getProjectDir()
			Expect(err).NotTo(HaveOccurred())

			// Idempotent, and independent of whichever earlier spec happened to
			// apply them first.
			By("registering the oci ClusterArtifactType, the credentials and the Endpoints")
			applyResource("", filepath.Join(dir, "examples", "oci", "artifact-type.yaml"))
			applyResource("default", filepath.Join(dir, "test", "fixtures", "secret.yaml"))
			applyResource("default", filepath.Join(dir, "test", "fixtures", "oci-cron-order.yaml"))
		})

		It("should validate an Endpoint whose type and Secret resolve", func() {
			Eventually(func() string {
				return endpointVerdict(probed, "Validated")
			}).Should(Equal("True/Valid"))
		})

		It("should record what the probe ran against", func() {
			// The gate reads these instead of remembering anything, so an Endpoint
			// whose record is missing or disagrees with the object is one that gets
			// probed again on every reconcile.
			Eventually(lastProbe).ShouldNot(BeEmpty())

			Expect(endpointField(probed, "{{ .status.probedGeneration }}")).
				To(Equal(endpointField(probed, "{{ .metadata.generation }}")))

			cmd := exec.Command("kubectl", "get", "-n", "default", "secret", "dst-reg-secret",
				"-o", "go-template={{ .metadata.resourceVersion }}")
			secretVersion, err := run(cmd)
			Expect(err).NotTo(HaveOccurred())
			Expect(endpointField(probed, "{{ .status.probedSecretVersion }}")).
				To(Equal(strings.TrimSpace(secretVersion)))
		})

		It("should reach a verdict on the target", func() {
			// Any answer passes. Whether the in-cluster registry is reachable, and
			// whether the controller trusts its certificate, are properties of the
			// test cluster and not of the gate; what matters here is that a probe
			// ran and its result reached the API.
			Eventually(func() string {
				return endpointVerdict(probed, "Reachable")
			}).ShouldNot(Equal("absent"))

			logf("Reachable: %s, Ready: %s\n",
				endpointVerdict(probed, "Reachable"), endpointVerdict(probed, "Ready"))
		})

		It("should not probe again while nothing changes", func() {
			settled := lastProbe()
			Expect(settled).NotTo(BeEmpty())

			// The promise the user guide makes: an Endpoint nobody touches produces
			// no outbound traffic and does not re-present the consumer's
			// credentials. lastProbeTime moving is a probe having run.
			Consistently(lastProbe, 30*time.Second, 5*time.Second).Should(Equal(settled))
		})

		It("should probe once more when forced, and then settle again", func() {
			before := lastProbe()

			cmd := exec.Command("kubectl", "annotate", "-n", "default",
				"endpoints.arc.opendefense.cloud", probed,
				fmt.Sprintf("arc.opendefense.cloud/force-at=%d", time.Now().Unix()), "--overwrite")
			_, err := run(cmd)
			Expect(err).NotTo(HaveOccurred())

			Eventually(lastProbe).ShouldNot(Equal(before))

			// The annotation must be honoured once, not once per reconcile, which is
			// what probedForceAt is for: annotations do not move the generation, so
			// nothing else would notice it had already been acted on.
			Expect(endpointField(probed, "{{ .status.probedForceAt }}")).NotTo(BeEmpty())

			forced := lastProbe()
			Consistently(lastProbe, 30*time.Second, 5*time.Second).Should(Equal(forced))
		})

		It("should report an unknown type and keep no probe record for it", func() {
			dir, err := getProjectDir()
			Expect(err).NotTo(HaveOccurred())

			manifest := filepath.Join(dir, "test", "fixtures", "endpoint-unknown-type.yaml")
			applyResource("default", manifest)
			DeferCleanup(func() {
				cmd := exec.Command("kubectl", "delete", "-n", "default", "-f", manifest, "--ignore-not-found")
				_, _ = run(cmd)
			})

			Eventually(func() string {
				return endpointVerdict("unknown-type", "Validated")
			}).Should(Equal("False/UnknownType"))

			// Nothing was probed, so no record may claim otherwise: a record with no
			// result beside it is what the gate reads as "the result was lost", and
			// it would probe on every reconcile.
			Expect(endpointField("unknown-type", "{{ .status.lastProbeTime }}")).To(BeEmpty())
			Expect(endpointVerdict("unknown-type", "Reachable")).To(Equal("absent"))
		})
	})
}
