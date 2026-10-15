// Copyright BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

//go:build e2e

package e2e

import (
	"fmt"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// walkThroughNamespace is where the walk-through steps run. It must not be
// "default": the namespaced ArtifactType "blob" would shadow the
// ClusterArtifactType of the same name used by the specs there.
const walkThroughNamespace = "walk-through"

// walkThroughSpecs runs every step of the walk-through (docs/walk-through) so the
// guide only ever shows examples that work. Like a reader, it applies the steps
// in order into one namespace, as later steps build on the resources of earlier ones.
// It runs against the images of this commit, while readers install the latest released
// chart (docs/walk-through/setup.md), so it can't catch a step that needs unreleased features.
//
// Called from inside the Ordered "ARC" container, like endpointControllerSpecs.
func walkThroughSpecs() {
	dir, _ := getProjectDir()
	// Every step directory has an Order. Its file is the last one in the directory,
	// as an Order must be created after the resources it uses.
	orderFiles, _ := filepath.Glob(filepath.Join(dir, "examples", "walk-through", "*", "*order.yaml"))
	if len(orderFiles) == 0 {
		// Fail loudly instead of passing without testing anything.
		panic("no walk-through steps found in " + dir)
	}

	Context("Walk-through", func() {
		BeforeAll(func() {
			// Ignore the error: the namespace may exist when re-running against a cluster.
			_, _ = run(exec.Command("kubectl", "create", "namespace", walkThroughNamespace))
			applyResource(walkThroughNamespace, filepath.Join(dir, "test", "fixtures", "service-account.yaml"))
		})

		for _, orderFile := range orderFiles {
			step := filepath.Dir(orderFile)

			It(fmt.Sprintf("should run step %s successfully", filepath.Base(step)), func() {
				// Replace instead of apply: the spec of an ArtifactType is immutable, and
				// later steps change it. --force deletes and recreates, like the guide does.
				_, err := run(exec.Command("kubectl", "replace", "--force", "-n", walkThroughNamespace, "-f", step))
				Expect(err).NotTo(HaveOccurred())

				output, err := run(exec.Command("kubectl", "get", "-n", walkThroughNamespace, "-f", orderFile, "-o", "name"))
				Expect(err).NotTo(HaveOccurred())
				orders := strings.Fields(output)
				Expect(orders).NotTo(BeEmpty())

				for _, order := range orders {
					order = path.Base(order) // "order.arc.opendefense.cloud/<name>"
					// Check final and succeeded on the same read: a cron order starts its
					// next run at any time, so a second read may no longer see Succeeded.
					Eventually(func() bool {
						return orderState(walkThroughNamespace, order, func(p string) bool {
							if stateFinal(p) && !stateSucceeded(p) {
								StopTrying(fmt.Sprintf("order %s/%s failed", walkThroughNamespace, order)).Now()
							}
							return stateSucceeded(p)
						})
					}).Should(BeTrue())
				}

				// Delete the orders, so a cron order does not keep running for the rest
				// of the suite. The files they transferred stay in the bucket for later steps.
				_, err = run(exec.Command("kubectl", "delete", "-n", walkThroughNamespace, "-f", orderFile))
				Expect(err).NotTo(HaveOccurred())
			})
		}
	})
}
