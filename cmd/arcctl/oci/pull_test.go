package oci

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func TestRunPullGinkgo(t *testing.T) {
	gomega.RegisterFailHandler(Fail)
	RunSpecs(t, "arcctl oci pull Suite")
}

var _ = Describe("Pull Command", func() {
	var (
		cmd *cobra.Command
	)

	BeforeEach(func() {
		cmd = NewPullCommand()
		cmd.SetContext(GinkgoT().Context())
	})

	AfterEach(func() {
		viper.Reset()
	})

	Context("when required configuration is missing", func() {

		It("should return an error if source.reference is missing", func() {
			err := runPull(cmd, []string{})
			gomega.Expect(err).To(gomega.HaveOccurred())
			gomega.Expect(err.Error()).To(gomega.ContainSubstring("source.reference is not set"))
		})
	})

	Context("when configuration is valid", func() {
		BeforeEach(func() {
			viper.Set("source.reference", "registry-1.docker.io/library/busybox:latest")
			viper.Set("tmp-dir", "tmp/test")
		})

		It("should pull the OCI artifact successfully", func() {
			err := runPull(cmd, []string{})
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
		})

	})
})
