package cos_test

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/rancher-sandbox/cOS/tests/sut"
)

var _ = Describe("cOS Deploy tests", func() {
	var s *sut.SUT

	BeforeEach(func() {
		s = sut.NewSUT()
		s.EventuallyConnects(360)
	})

	AfterEach(func() {
		// Try to gather mtree logs on failure
		if CurrentGinkgoTestDescription().Failed {
			s.GatherAllLogs()
		}
		if CurrentGinkgoTestDescription().Failed == false {
			s.Reset()
		}
	})

	Context("From active", func() {
		When("deploying again", func() {
			It("deploys only if --force flag is provided", func() {
				By("deploying without --force")
				out, err := s.Command(fmt.Sprintf("cos-deploy --docker-image %s:cos-system-%s", s.GreenRepo, s.TestVersion))
				Expect(out).Should(ContainSubstring("There is already an active deployment"))
				Expect(err).To(HaveOccurred())
				By("deploying with --force")
				out, err = s.Command(fmt.Sprintf("cos-deploy --force --docker-image %s:cos-system-%s", s.GreenRepo, s.TestVersion))
				Expect(out).Should(ContainSubstring("Forcing overwrite"))
				Expect(out).Should(ContainSubstring("now you might want to reboot"))
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})

	Context("From recovery", func() {
		When("deploying again", func() {
			It("deploys only if --force flag is provided", func() {
				err := s.ChangeBootOnce(sut.Recovery)
				Expect(err).ToNot(HaveOccurred())
				s.Reboot()
				ExpectWithOffset(1, s.BootFrom()).To(Equal(sut.Recovery))
				By("deploying with --force")
				out, err := s.Command(fmt.Sprintf("cos-deploy --force --docker-image %s:cos-system-%s", s.GreenRepo, s.TestVersion))
				Expect(out).Should(ContainSubstring("now you might want to reboot"))
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})
})
