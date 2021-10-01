package sut

import (
	"fmt"
	"github.com/bramvdbogaerde/go-scp"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	ssh "golang.org/x/crypto/ssh"
)

const (
	grubSwapOnce = "grub2-editenv /oem/grubenv set next_entry=%s"
	grubSwap     = "grub2-editenv /oem/grubenv set saved_entry=%s"

	Passive     = 0
	Active      = iota
	Recovery    = iota
	UnknownBoot = iota

	TimeoutRawDiskTest = 600  // Timeout to connect for recovery_raw_disk_test
	arm64 = "arm64"
	x86_64 = "x86_64"
)

type SUT struct {
	Host     string
	Username string
	Password string
	Arch 	 string
}

func NewSUT() *SUT {

	user := os.Getenv("COS_USER")
	if user == "" {
		user = "root"
	}
	pass := os.Getenv("COS_PASS")
	if pass == "" {
		pass = "cos"
	}

	host := os.Getenv("COS_HOST")
	if host == "" {
		host = "127.0.0.1:2222"
	}

	arch := os.Getenv("ARCH")
	if arch == "" {
		arch = x86_64
	}

	return &SUT{
		Host:     host,
		Username: user,
		Password: pass,
		Arch: 	  arch,
	}
}

func (s *SUT) ChangeBoot(b int) error {

	var bootEntry string

	switch b {
	case Active:
		bootEntry = "cos"
	case Passive:
		bootEntry = "fallback"
	case Recovery:
		bootEntry = "recovery"
	}

	_, err := s.command(fmt.Sprintf(grubSwap, bootEntry), false)
	Expect(err).ToNot(HaveOccurred())

	return nil
}

func (s *SUT) ChangeBootOnce(b int) error {

	var bootEntry string

	switch b {
	case Active:
		bootEntry = "cos"
	case Passive:
		bootEntry = "fallback"
	case Recovery:
		bootEntry = "recovery"
	}

	_, err := s.command(fmt.Sprintf(grubSwapOnce, bootEntry), false)
	Expect(err).ToNot(HaveOccurred())

	return nil
}

// Reset runs reboots cOS into Recovery and runs cos-reset.
// It will boot back the system from the Active partition afterwards
func (s *SUT) Reset() {
	if s.BootFrom() != Recovery {
		By("Reboot to recovery before reset")
		err := s.ChangeBootOnce(Recovery)
		Expect(err).ToNot(HaveOccurred())
		s.Reboot()
		Expect(s.BootFrom()).To(Equal(Recovery))
	}

	By("Running cos-reset")
	out, err := s.command("cos-reset", false)
	Expect(err).ToNot(HaveOccurred())
	Expect(out).Should(ContainSubstring("Installing"))

	By("Reboot to active after cos-reset")
	s.Reboot()
	ExpectWithOffset(1, s.BootFrom()).To(Equal(Active))
}

// BootFrom returns the booting partition of the SUT
func (s *SUT) BootFrom() int {
	out, err := s.command("cat /proc/cmdline", false)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	switch {
	case strings.Contains(out, "COS_ACTIVE"):
		return Active
	case strings.Contains(out, "COS_PASSIVE"):
		return Passive
	case strings.Contains(out, "COS_RECOVERY"), strings.Contains(out, "COS_SYSTEM"):
		return Recovery
	default:
		return UnknownBoot
	}
}

// SquashFSRecovery returns true if we are in recovery mode and booting from squashfs
func (s *SUT) SquashFSRecovery() bool {
	out, err := s.command("cat /proc/cmdline", false)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	return strings.Contains(out, "rd.live.squashimg")
}

func (s *SUT) GetOSRelease(ss string) string {
	out, err := s.Command(fmt.Sprintf("source /etc/os-release && echo $%s", ss))
	Expect(err).ToNot(HaveOccurred())
	Expect(out).ToNot(Equal(""))

	return out
}

func (s *SUT) EventuallyConnects(t ...int) {
	dur := 180
	if len(t) > 0 {
		dur = t[0]
	}
	Eventually(func() error {
		if s.Arch == arm64 {
			// Reload ssh config from vagrant ssh-config
			// As after a reboot it wont change for a while, we need to keep getting it
			host, _ := getVagrantIP(s.Arch)
			s.Host = fmt.Sprintf("%s:22", host)
		}
		out, err := s.command("echo ping", true)
		if out == "ping\n" {
			return nil
		}
		return err
	}, time.Duration(time.Duration(dur)*time.Second), time.Duration(5*time.Second)).ShouldNot(HaveOccurred())
}

// Command sends a command to the SUIT and waits for reply
func (s *SUT) Command(cmd string) (string, error) {
	return s.command(cmd, false)
}

func (s *SUT) command(cmd string, timeout bool) (string, error) {
	client, err := s.connectToHost(timeout)
	if err != nil {
		return "", err
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return "", err
	}

	out, err := session.CombinedOutput(cmd)
	if err != nil {
		return string(out), errors.Wrap(err, string(out))
	}

	return string(out), err
}

// Reboot reboots the system under test
func (s *SUT) Reboot(t ...int) {
	By("Rebooting")
	_, _ = s.command("reboot", true)
	time.Sleep(10 * time.Second)
	s.EventuallyConnects(t...)
}

func (s *SUT) clientConfig() *ssh.ClientConfig {
	sshConfig := &ssh.ClientConfig{
		User:    s.Username,
		Auth:    []ssh.AuthMethod{ssh.Password(s.Password)},
		Timeout: 30 * time.Second, // max time to establish connection
	}
	sshConfig.HostKeyCallback = ssh.InsecureIgnoreHostKey()

	return sshConfig
}

func (s *SUT) SendFile(src, dst, permission string) error {
	sshConfig := s.clientConfig()
	scpClient := scp.NewClientWithTimeout(s.Host, sshConfig, 10*time.Second)
	defer scpClient.Close()

	if err := scpClient.Connect(); err != nil {
		return err
	}

	f, err := os.Open(src)
	if err != nil {
		return err
	}

	defer scpClient.Close()
	defer f.Close()

	if err := scpClient.CopyFile(f, dst, permission); err != nil {
		return err
	}
	return nil
}

func (s *SUT) connectToHost(timeout bool) (*ssh.Client, error) {
	sshConfig := s.clientConfig()

	client, err := DialWithDeadline("tcp", s.Host, sshConfig, timeout)
	if err != nil {
		return nil, err
	}

	return client, nil
}

// GatherAllLogs will try to gather as much info from the system as possible, including services, dmesg and os related info
func (s SUT) GatherAllLogs()  {
	services := []string{
		"cos-setup-boot",
		"cos-setup-fs",
		"cos-setup-initramfs",
		"cos-setup-network",
		"cos-setup-reconcile",
		"cos-setup-rootfs",
	}

	logFiles := []string{
		"/tmp/image-mtree-check.log",
		"/tmp/luet_mtree_failures.log",
		"/tmp/luet_mtree.log",
		"/tmp/luet.log",
	}

	// services
	for _, ser := range services {
		out, err := s.command(fmt.Sprintf("journalctl -u %s -o short-iso >> /tmp/%s.log", ser, ser), true)
		if err != nil {
			fmt.Printf("Error getting journal for service %s: %s\n", ser, err.Error())
			fmt.Printf("Output from command: %s\n", out)
		}
		s.GatherLog(fmt.Sprintf("/tmp/%s.log", ser))
	}

	// log files
	for _, file := range logFiles {
		s.GatherLog(file)
	}

	// dmesg
	out, err := s.command("dmesg > /tmp/dmesg", true)
	if err != nil {
		fmt.Printf("Error getting dmesg : %s\n", err.Error())
		fmt.Printf("Output from command: %s\n", out)
	}
	s.GatherLog("/tmp/dmesg")

	// grab full journal
	out, err = s.command("journalctl -o short-iso > /tmp/journal.log", true)
	if err != nil {
		fmt.Printf("Error getting full journalctl info : %s\n", err.Error())
		fmt.Printf("Output from command: %s\n", out)
	}
	s.GatherLog("/tmp/journal.log")

	// uname
	out, err = s.command("uname -a > /tmp/uname.log", true)
	if err != nil {
		fmt.Printf("Error getting uname info : %s\n", err.Error())
		fmt.Printf("Output from command: %s\n", out)
	}
	s.GatherLog("/tmp/uname.log")

	// disk info
	out, err = s.command("lsblk -a >> /tmp/disks.log", true)
	if err != nil {
		fmt.Printf("Error getting disk info : %s\n", err.Error())
		fmt.Printf("Output from command: %s\n", out)
	}
	out, err = s.command("blkid >> /tmp/disks.log", true)
	if err != nil {
		fmt.Printf("Error getting disk info : %s\n", err.Error())
		fmt.Printf("Output from command: %s\n", out)
	}
	s.GatherLog("/tmp/disks.log")

	// Grab users
	s.GatherLog("/etc/passwd")
	// Grab system info
	s.GatherLog("/etc/os-release")


}

// GatherLog will try to scp the given log from the machine to a local file
func (s SUT) GatherLog(logPath string) {
	fmt.Printf("Trying to get file: %s\n", logPath)
	sshConfig := s.clientConfig()
	scpClient := scp.NewClientWithTimeout(s.Host, sshConfig, 10*time.Second)

	err := scpClient.Connect()
	if err != nil {
		scpClient.Close()
		fmt.Println("Couldn't establish a connection to the remote server ", err)
		return
	}

	fmt.Printf("Connection to %s established!\n", s.Host)
	baseName := filepath.Base(logPath)
	_ = os.Mkdir("logs", 0755)

	f, _ := os.Create(fmt.Sprintf("logs/%s", baseName))
	// Close the file after it has been copied
	// Close client connection after the file has been copied
	defer scpClient.Close()
	defer f.Close()

	err = scpClient.CopyFromRemote(f, logPath)

	if err != nil {
		fmt.Printf("Error while copying file: %s\n", err.Error())
		return
	}
	// Change perms so its world readable
	_ = os.Chmod(fmt.Sprintf("logs/%s", baseName), 0666)
	fmt.Printf("File %s copied!\n", baseName)

}

// DialWithDeadline Dials SSH with a deadline to avoid Read timeouts
func DialWithDeadline(network string, addr string, config *ssh.ClientConfig, timeout bool) (*ssh.Client, error) {
	conn, err := net.DialTimeout(network, addr, config.Timeout)
	if err != nil {
		return nil, err
	}
	if config.Timeout > 0 {
		conn.SetReadDeadline(time.Now().Add(config.Timeout))
		conn.SetWriteDeadline(time.Now().Add(config.Timeout))
	}
	c, chans, reqs, err := ssh.NewClientConn(conn, addr, config)
	if err != nil {
		return nil, err
	}
	if !timeout {
		conn.SetReadDeadline(time.Time{})
		conn.SetWriteDeadline(time.Time{})
	}

	go func() {
		t := time.NewTicker(2 * time.Second)
		defer t.Stop()
		for range t.C {
			_, _, err := c.SendRequest("keepalive@golang.org", true, nil)
			if err != nil {
				return
			}
		}
	}()
	return ssh.NewClient(c, chans, reqs), nil
}

func getVagrantIP(arch string)  (string, error){
	By("After reboot, getting the IP from vagrant")
	cmd := exec.Command("vagrant", "ssh-config", "cos-arm64")
	out, _ := cmd.Output()
	rx, _ := regexp.Compile("\\s([0-9]{1,3}\\.[0-9]{1,3}\\.[0-9]{1,3}\\.[0-9]{1,3})\\n")
	ip := rx.Find(out)
	return strings.TrimSpace(string(ip)), nil
}