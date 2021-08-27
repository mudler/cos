package cos_test

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/rancher-sandbox/cOS/tests/sut"
)

var _ = Describe("cOS Feature tests", func() {
	var s *sut.SUT
	BeforeEach(func() {
		s = sut.NewSUT()
		s.EventuallyConnects(360)
	})

	Context("After install", func() {
		It("can enable a persistent k3s install", func() {

			s.Command("echo 'COS_FEATURESDIR=/usr/local/features' >> /etc/cos/config")
			s.Command("mkdir /usr/local/features")
			s.Command("cp -rf /system/features/vagrant.yaml /usr/local/features")

			err := s.SendFile("../assets/k3s.yaml", "/usr/local/features/k3s.yaml", "0770")
			Expect(err).ToNot(HaveOccurred())

			out, err := s.Command("cos-feature enable k3s")
			Expect(err).ToNot(HaveOccurred())
			Expect(out).Should(ContainSubstring("k3s enabled"))

			s.Reboot()

			out, err = s.Command("cos-feature list")
			Expect(err).ToNot(HaveOccurred())
			Expect(out).Should(ContainSubstring("k3s (enabled)"))

			Eventually(func() string {
				out, _ := s.Command("k3s --data-dir /usr/local/rancher/k3s/ kubectl get pods -A")
				return out

			}, time.Duration(time.Duration(400)*time.Second), time.Duration(5*time.Second)).Should(ContainSubstring("local-path-provisioner"))

			out, err = s.Command("k3s --data-dir /usr/local/rancher/k3s/ kubectl create deployment test-sut-deployment --image nginx")
			Expect(err).ToNot(HaveOccurred())

			s.Reboot()
			Eventually(func() string {
				out, _ := s.Command("k3s --data-dir /usr/local/rancher/k3s/ kubectl get pods -A")
				return out

			}, time.Duration(time.Duration(900)*time.Second), time.Duration(5*time.Second)).Should(ContainSubstring("test-sut-deployment"))

			out, err = s.Command("cos-feature disable k3s")
			Expect(err).ToNot(HaveOccurred())
			Expect(out).Should(ContainSubstring("k3s disabled"))
			s.Reboot()

			out, err = s.Command("cos-feature disable k3s")
			Expect(err).ToNot(HaveOccurred())
			Expect(out).Should(ContainSubstring("Feature k3s not enabled"))
		})
	})
})
