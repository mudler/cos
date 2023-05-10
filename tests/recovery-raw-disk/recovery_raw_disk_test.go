package elemental_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	sut "github.com/rancher-sandbox/ele-testhelpers/vm"
)

var _ = Describe("Elemental Recovery deploy tests", func() {
	var s *sut.SUT

	BeforeEach(func() {
		s = sut.NewSUT()
		s.EventuallyConnects(sut.TimeoutRawDiskTest)
	})

	AfterEach(func() {
		if CurrentSpecReport().Failed() {
			s.GatherAllLogs()
		}
	})

	Context("after running recovery from the raw_disk image", func() {
		It("uses cos-deploy to install", func() {
			ExpectWithOffset(1, s.BootFrom()).To(Equal(sut.Recovery))

			_, err := s.Command("elemental reset")
			Expect(err).ToNot(HaveOccurred())

			s.Reboot(sut.TimeoutRawDiskTest)
			ExpectWithOffset(1, s.BootFrom()).To(Equal(sut.Active))
		})
	})
})
