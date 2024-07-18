/*
   Copyright © 2021 SUSE LLC

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package elemental_test

import (
	"errors"
	"fmt"
	iofs "io/fs"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/twpayne/go-vfs/v4"
	"github.com/twpayne/go-vfs/v4/vfst"

	conf "github.com/rancher/elemental-toolkit/v2/pkg/config"
	"github.com/rancher/elemental-toolkit/v2/pkg/constants"
	"github.com/rancher/elemental-toolkit/v2/pkg/elemental"
	"github.com/rancher/elemental-toolkit/v2/pkg/mocks"
	"github.com/rancher/elemental-toolkit/v2/pkg/types"
	"github.com/rancher/elemental-toolkit/v2/pkg/utils"
)

const printOutput = `BYT;
/dev/loop0:50593792s:loopback:512:512:gpt:Loopback device:;`
const partTmpl = `
%d:%ss:%ss:2048s:ext4::type=83;`

func TestElementalSuite(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Elemental test suite")
}

var _ = Describe("Elemental", Label("elemental"), func() {
	var config *types.Config
	var runner *mocks.FakeRunner
	var logger types.Logger
	var syscall types.SyscallInterface
	var client *mocks.FakeHTTPClient
	var mounter *mocks.FakeMounter
	var extractor *mocks.FakeImageExtractor
	var fs *vfst.TestFS
	var cleanup func()
	BeforeEach(func() {
		runner = mocks.NewFakeRunner()
		syscall = &mocks.FakeSyscall{}
		mounter = mocks.NewFakeMounter()
		client = &mocks.FakeHTTPClient{}
		logger = types.NewNullLogger()
		extractor = mocks.NewFakeImageExtractor(logger)
		fs, cleanup, _ = vfst.NewTestFS(nil)
		config = conf.NewConfig(
			conf.WithFs(fs),
			conf.WithRunner(runner),
			conf.WithLogger(logger),
			conf.WithMounter(mounter),
			conf.WithSyscall(syscall),
			conf.WithClient(client),
			conf.WithImageExtractor(extractor),
		)
	})
	AfterEach(func() { cleanup() })
	Describe("MountRWPartition", Label("mount"), func() {
		var parts types.ElementalPartitions
		BeforeEach(func() {
			parts = conf.NewInstallElementalPartitions()
			err := utils.MkdirAll(fs, "/some", constants.DirPerm)
			Expect(err).ToNot(HaveOccurred())
			_, err = fs.Create("/some/device")
			Expect(err).ToNot(HaveOccurred())

			parts.OEM.Path = "/dev/device1"
		})

		It("Mounts and umounts a partition with RW", func() {
			umount, err := elemental.MountRWPartition(*config, parts.OEM)
			Expect(err).To(BeNil())
			lst, _ := mounter.List()
			Expect(len(lst)).To(Equal(1))
			Expect(lst[0].Opts).To(Equal([]string{"rw"}))

			Expect(umount()).ShouldNot(HaveOccurred())
			lst, _ = mounter.List()
			Expect(len(lst)).To(Equal(0))
		})
		It("Remounts a partition with RW", func() {
			err := elemental.MountPartition(*config, parts.OEM)
			Expect(err).To(BeNil())
			lst, _ := mounter.List()
			Expect(len(lst)).To(Equal(1))

			umount, err := elemental.MountRWPartition(*config, parts.OEM)
			Expect(err).To(BeNil())
			lst, _ = mounter.List()
			// fake mounter is not merging remounts it just appends
			Expect(len(lst)).To(Equal(2))
			Expect(lst[1].Opts).To(Equal([]string{"remount", "rw"}))

			Expect(umount()).ShouldNot(HaveOccurred())
			lst, _ = mounter.List()
			// Increased once more to remount read-onply
			Expect(len(lst)).To(Equal(3))
			Expect(lst[2].Opts).To(Equal([]string{"remount", "ro"}))
		})
		It("Fails to mount a partition", func() {
			mounter.ErrorOnMount = true
			_, err := elemental.MountRWPartition(*config, parts.OEM)
			Expect(err).Should(HaveOccurred())
		})
		It("Fails to remount a partition", func() {
			err := elemental.MountPartition(*config, parts.OEM)
			Expect(err).To(BeNil())
			lst, _ := mounter.List()
			Expect(len(lst)).To(Equal(1))

			mounter.ErrorOnMount = true
			_, err = elemental.MountRWPartition(*config, parts.OEM)
			Expect(err).Should(HaveOccurred())
			lst, _ = mounter.List()
			Expect(len(lst)).To(Equal(1))
		})
	})
	Describe("IsMounted", Label("ismounted"), func() {
		It("checks a mounted partition", func() {
			part := &types.Partition{
				MountPoint: "/some/mountpoint",
			}
			err := mounter.Mount("/some/device", "/some/mountpoint", "auto", []string{})
			Expect(err).ShouldNot(HaveOccurred())
			mnt, err := elemental.IsMounted(*config, part)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(mnt).To(BeTrue())
		})
		It("checks a not mounted partition", func() {
			part := &types.Partition{
				MountPoint: "/some/mountpoint",
			}
			mnt, err := elemental.IsMounted(*config, part)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(mnt).To(BeFalse())
		})
		It("checks a partition without mountpoint", func() {
			part := &types.Partition{}
			mnt, err := elemental.IsMounted(*config, part)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(mnt).To(BeFalse())
		})
		It("checks a nil partitiont", func() {
			mnt, err := elemental.IsMounted(*config, nil)
			Expect(err).Should(HaveOccurred())
			Expect(mnt).To(BeFalse())
		})
	})
	Describe("MountPartitions", Label("MountPartitions", "disk", "partition", "mount"), func() {
		var parts types.ElementalPartitions
		BeforeEach(func() {
			parts = conf.NewInstallElementalPartitions()

			err := utils.MkdirAll(fs, "/some", constants.DirPerm)
			Expect(err).ToNot(HaveOccurred())
			_, err = fs.Create("/some/device")
			Expect(err).ToNot(HaveOccurred())

			parts.EFI.Path = "/dev/device1"
			parts.OEM.Path = "/dev/device2"
			parts.Recovery.Path = "/dev/device3"
			parts.State.Path = "/dev/device4"
			parts.Persistent.Path = "/dev/device5"
		})

		It("Mounts disk partitions", func() {
			// Ignores an already unmounted parition
			Expect(elemental.MountPartition(*config, parts.PartitionsByMountPoint(false)[0])).To(Succeed())
			lst, _ := mounter.List()
			Expect(len(lst)).To(Equal(1))

			err := elemental.MountPartitions(*config, parts.PartitionsByMountPoint(false))
			Expect(err).To(BeNil())
			lst, _ = mounter.List()
			Expect(len(lst)).To(Equal(len(parts.PartitionsByMountPoint(false))))
		})

		It("Ignores partitions with undefiend mountpoints", func() {
			parts.EFI.MountPoint = ""

			err := elemental.MountPartitions(*config, parts.PartitionsByMountPoint(false))
			Expect(err).To(BeNil())
			lst, _ := mounter.List()
			for _, i := range lst {
				Expect(i.Path).NotTo(Equal("/dev/device1"))
			}
		})

		It("Mounts disk partitions excluding recovery", func() {
			err := elemental.MountPartitions(*config, parts.PartitionsByMountPoint(false, parts.Recovery))
			Expect(err).To(BeNil())
			lst, _ := mounter.List()
			for _, i := range lst {
				Expect(i.Path).NotTo(Equal("/dev/device3"))
			}
		})

		It("Fails if some partition resists to mount ", func() {
			mounter.ErrorOnMount = true
			err := elemental.MountPartitions(*config, parts.PartitionsByMountPoint(false))
			Expect(err).NotTo(BeNil())
		})

		It("does not mount partitions without a mountpoint ", func() {
			parts.OEM.MountPoint = ""
			err := elemental.MountPartitions(*config, parts.PartitionsByMountPoint(false))
			Expect(err).To(BeNil())
			lst, _ := mounter.List()
			for _, i := range lst {
				Expect(i.Path).NotTo(Equal("/dev/device2"))
			}
		})

		It("fails to mount missing partitions", func() {
			// Without path tries to get devices by label
			parts.OEM.Path = ""
			err := elemental.MountPartitions(*config, parts.PartitionsByMountPoint(false))
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("UnmountPartitions", Label("UnmountPartitions", "disk", "partition", "unmount"), func() {
		var parts types.ElementalPartitions

		BeforeEach(func() {
			parts = conf.NewInstallElementalPartitions()

			err := utils.MkdirAll(fs, "/some", constants.DirPerm)
			Expect(err).ToNot(HaveOccurred())
			_, err = fs.Create("/some/device")
			Expect(err).ToNot(HaveOccurred())

			parts.EFI.Path = "/dev/device1"
			parts.OEM.Path = "/dev/device2"
			parts.Recovery.Path = "/dev/device3"
			parts.State.Path = "/dev/device4"
			parts.Persistent.Path = "/dev/device5"

			err = elemental.MountPartitions(*config, parts.PartitionsByMountPoint(false))
			Expect(err).ToNot(HaveOccurred())
			lst, _ := mounter.List()
			Expect(len(lst)).To(Equal(len(parts.PartitionsByMountPoint(false))))
		})

		It("Unmounts disk partitions", func() {
			Expect(elemental.UnmountPartition(*config, parts.PartitionsByMountPoint(false)[0])).To(Succeed())
			lst, _ := mounter.List()
			Expect(len(lst)).To(Equal(len(parts.PartitionsByMountPoint(false)) - 1))

			err := elemental.UnmountPartitions(*config, parts.PartitionsByMountPoint(true))
			Expect(err).To(BeNil())
			lst, _ = mounter.List()
			Expect(len(lst)).To(Equal(0))
		})

		It("Ignores partitions with undefiend mountpoints", func() {
			parts.EFI.MountPoint = ""

			err := elemental.UnmountPartitions(*config, parts.PartitionsByMountPoint(true))
			Expect(err).To(BeNil())
			lst, _ := mounter.List()
			Expect(len(lst)).To(Equal(1))
		})

		It("Fails to unmount disk partitions", func() {
			mounter.ErrorOnUnmount = true
			err := elemental.UnmountPartitions(*config, parts.PartitionsByMountPoint(true))
			Expect(err).NotTo(BeNil())
		})
	})

	Describe("MountImage", Label("MountImage", "mount", "image"), func() {
		var img *types.Image
		BeforeEach(func() {
			img = &types.Image{MountPoint: "/some/mountpoint"}
		})

		It("Mounts file system image", func() {
			runner.ReturnValue = []byte("/dev/loop")
			Expect(elemental.MountFileSystemImage(*config, img)).To(BeNil())
			Expect(img.LoopDevice).To(Equal("/dev/loop"))
		})

		It("Fails to set a loop device", Label("loop"), func() {
			runner.ReturnError = errors.New("failed to set a loop device")
			Expect(elemental.MountFileSystemImage(*config, img)).NotTo(BeNil())
			Expect(img.LoopDevice).To(Equal(""))
		})

		It("Fails to mount a loop device", Label("loop"), func() {
			runner.ReturnValue = []byte("/dev/loop")
			mounter.ErrorOnMount = true
			Expect(elemental.MountFileSystemImage(*config, img)).NotTo(BeNil())
			Expect(img.LoopDevice).To(Equal(""))
		})
	})

	Describe("UnmountImage", Label("UnmountImage", "mount", "image"), func() {
		var img *types.Image
		BeforeEach(func() {
			runner.ReturnValue = []byte("/dev/loop")
			img = &types.Image{MountPoint: "/some/mountpoint"}
			Expect(elemental.MountFileSystemImage(*config, img)).To(BeNil())
			Expect(img.LoopDevice).To(Equal("/dev/loop"))
		})

		It("Unmounts file system image", func() {
			Expect(elemental.UnmountFileSystemImage(*config, img)).To(BeNil())
			Expect(img.LoopDevice).To(Equal(""))
		})

		It("Fails to unmount a mountpoint", func() {
			mounter.ErrorOnUnmount = true
			Expect(elemental.UnmountFileSystemImage(*config, img)).NotTo(BeNil())
		})

		It("Fails to unset a loop device", Label("loop"), func() {
			runner.ReturnError = errors.New("failed to unset a loop device")
			Expect(elemental.UnmountFileSystemImage(*config, img)).NotTo(BeNil())
		})
	})

	Describe("CreateFileSystemImage", Label("CreateFileSystemImage", "image"), func() {
		var img *types.Image
		BeforeEach(func() {
			img = &types.Image{
				Label:      "SOME_LABEL",
				Size:       32,
				File:       filepath.Join(constants.StateDir, "some.img"),
				FS:         constants.LinuxImgFs,
				MountPoint: constants.TransitionDir,
				Source:     types.NewDirSrc(constants.ISOBaseTree),
			}
			_ = utils.MkdirAll(fs, constants.ISOBaseTree, constants.DirPerm)
		})

		It("Creates a new file system image", func() {
			_, err := fs.Stat(img.File)
			Expect(err).NotTo(BeNil())
			err = elemental.CreateFileSystemImage(*config, img, "", false)
			Expect(err).To(BeNil())
			stat, err := fs.Stat(img.File)
			Expect(err).To(BeNil())
			Expect(stat.Size()).To(Equal(int64(32 * 1024 * 1024)))
		})

		It("Fails formatting a file system image", Label("format"), func() {
			runner.ReturnError = errors.New("run error")
			_, err := fs.Stat(img.File)
			Expect(err).NotTo(BeNil())
			err = elemental.CreateFileSystemImage(*config, img, "", false)
			Expect(err).NotTo(BeNil())
			_, err = fs.Stat(img.File)
			Expect(err).NotTo(BeNil())
		})
	})

	Describe("FormatPartition", Label("FormatPartition", "partition", "format"), func() {
		It("Reformats an already existing partition", func() {
			part := &types.Partition{
				Path:            "/dev/device1",
				FS:              "ext4",
				FilesystemLabel: "MY_LABEL",
			}
			Expect(elemental.FormatPartition(*config, part)).To(BeNil())
		})

	})
	Describe("PartitionAndFormatDevice", Label("PartitionAndFormatDevice", "partition", "format"), func() {
		var cInit *mocks.FakeCloudInitRunner
		var partNum int
		var printOut string
		var failPart bool
		var install *types.InstallSpec

		BeforeEach(func() {
			cInit = &mocks.FakeCloudInitRunner{ExecStages: []string{}, Error: false}
			config.CloudInitRunner = cInit
			install = conf.NewInstallSpec(*config)
			install.Target = "/some/device"

			err := utils.MkdirAll(fs, "/some", constants.DirPerm)
			Expect(err).ToNot(HaveOccurred())
			_, err = fs.Create("/some/device")
			Expect(err).ToNot(HaveOccurred())
		})

		Describe("Successful run", func() {
			var runFunc func(cmd string, args ...string) ([]byte, error)
			var efiPartCmds, partCmds, biosPartCmds [][]string
			BeforeEach(func() {
				partNum, printOut = 0, printOutput
				err := utils.MkdirAll(fs, "/some", constants.DirPerm)
				Expect(err).To(BeNil())
				efiPartCmds = [][]string{
					{
						"parted", "--script", "--machine", "--", "/some/device", "unit", "s",
						"mklabel", "gpt",
					}, {
						"parted", "--script", "--machine", "--", "/some/device", "unit", "s",
						"mkpart", "efi", "fat32", "2048", "133119", "set", "1", "esp", "on",
					}, {"mkfs.vfat", "-n", "COS_GRUB", "/some/device1"},
				}
				biosPartCmds = [][]string{
					{
						"parted", "--script", "--machine", "--", "/some/device", "unit", "s",
						"mklabel", "gpt",
					}, {
						"parted", "--script", "--machine", "--", "/some/device", "unit", "s",
						"mkpart", "bios", "", "2048", "4095", "set", "1", "bios_grub", "on",
					}, {"wipefs", "--all", "/some/device1"},
				}
				// These commands are only valid for EFI case
				partCmds = [][]string{
					{
						"parted", "--script", "--machine", "--", "/some/device", "unit", "s",
						"mkpart", "oem", "ext4", "133120", "264191",
					}, {"mkfs.ext4", "-L", "COS_OEM", "/some/device2"}, {
						"parted", "--script", "--machine", "--", "/some/device", "unit", "s",
						"mkpart", "recovery", "ext4", "264192", "8652799",
					}, {"mkfs.ext4", "-L", "COS_RECOVERY", "/some/device3"}, {
						"parted", "--script", "--machine", "--", "/some/device", "unit", "s",
						"mkpart", "state", "ext4", "8652800", "25430015",
					}, {"mkfs.ext4", "-L", "COS_STATE", "/some/device4"}, {
						"parted", "--script", "--machine", "--", "/some/device", "unit", "s",
						"mkpart", "persistent", "ext4", "25430016", "100%",
					}, {"mkfs.ext4", "-L", "COS_PERSISTENT", "/some/device5"},
				}

				runFunc = func(cmd string, args ...string) ([]byte, error) {
					switch cmd {
					case "parted":
						idx := 0
						for i, arg := range args {
							if arg == "mkpart" {
								idx = i
								break
							}
						}
						if idx > 0 {
							partNum++
							printOut += fmt.Sprintf(partTmpl, partNum, args[idx+3], args[idx+4])
							_, _ = fs.Create(fmt.Sprintf("/some/device%d", partNum))
						}
						return []byte(printOut), nil
					default:
						return []byte{}, nil
					}
				}
				runner.SideEffect = runFunc
			})

			It("Successfully creates partitions and formats them, EFI boot", func() {
				install.PartTable = types.GPT
				install.Firmware = types.EFI
				install.Partitions.SetFirmwarePartitions(types.EFI, types.GPT)
				Expect(elemental.PartitionAndFormatDevice(*config, install)).To(BeNil())
				Expect(runner.MatchMilestones(append(efiPartCmds, partCmds...))).To(BeNil())
			})

			It("Successfully creates partitions and formats them, BIOS boot", func() {
				install.PartTable = types.GPT
				install.Firmware = types.BIOS
				install.Partitions.SetFirmwarePartitions(types.BIOS, types.GPT)
				Expect(elemental.PartitionAndFormatDevice(*config, install)).To(BeNil())
				Expect(runner.MatchMilestones(biosPartCmds)).To(BeNil())
			})
		})

		Describe("Run with failures", func() {
			var runFunc func(cmd string, args ...string) ([]byte, error)
			BeforeEach(func() {
				err := utils.MkdirAll(fs, "/some", constants.DirPerm)
				Expect(err).To(BeNil())
				partNum, printOut = 0, printOutput
				runFunc = func(cmd string, args ...string) ([]byte, error) {
					switch cmd {
					case "parted":
						idx := 0
						for i, arg := range args {
							if arg == "mkpart" {
								idx = i
								break
							}
						}
						if idx > 0 {
							partNum++
							printOut += fmt.Sprintf(partTmpl, partNum, args[idx+3], args[idx+4])
							if failPart {
								return []byte{}, errors.New("Failure")
							}
							_, _ = fs.Create(fmt.Sprintf("/some/device%d", partNum))
						}
						return []byte(printOut), nil
					case "mkfs.ext4", "wipefs", "mkfs.vfat":
						return []byte{}, errors.New("Failure")
					default:
						return []byte{}, nil
					}
				}
				runner.SideEffect = runFunc
			})

			It("Fails creating efi partition", func() {
				failPart = true
				Expect(elemental.PartitionAndFormatDevice(*config, install)).NotTo(BeNil())
				// Failed to create first partition
				Expect(partNum).To(Equal(1))
			})

			It("Fails formatting efi partition", func() {
				failPart = false
				Expect(elemental.PartitionAndFormatDevice(*config, install)).NotTo(BeNil())
				// Failed to format first partition
				Expect(partNum).To(Equal(1))
			})
		})
	})
	Describe("MirrorRoot", func() {
		var destDir string
		var syncFunc func(l types.Logger, r types.Runner, f types.FS, src string, dst string, excl ...string) error
		var fErr error
		BeforeEach(func() {
			var err error
			destDir, err = utils.TempDir(fs, "", "elemental")
			Expect(err).ShouldNot(HaveOccurred())
			syncFunc = func(_ types.Logger, _ types.Runner, _ types.FS, src string, dst string, _ ...string) error {
				return fErr
			}
		})
		It("Unpacks a docker image to target", Label("docker"), func() {
			dockerSrc := types.NewDockerSrc("docker/image:latest")
			err := elemental.DumpSource(*config, destDir, dockerSrc, syncFunc)
			Expect(dockerSrc.GetDigest()).To(Equal("fakeDigest"))
			Expect(err).To(BeNil())
		})
		It("Fails to mirror data", func() {
			fErr = errors.New("fake synching failure")
			err := elemental.DumpSource(*config, destDir, types.NewDirSrc("/source"), syncFunc)
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring("fake synching failure"))
		})
	})
	Describe("DumpSource", Label("dump"), func() {
		var destDir string
		var syncFunc func(l types.Logger, r types.Runner, f types.FS, src string, dst string, excl ...string) error
		var fErr error
		var src, dst string
		BeforeEach(func() {
			var err error
			src = ""
			dst = ""
			destDir, err = utils.TempDir(fs, "", "elemental")
			Expect(err).ShouldNot(HaveOccurred())
			syncFunc = func(_ types.Logger, _ types.Runner, _ types.FS, s string, d string, _ ...string) error {
				src = s
				dst = d
				return fErr
			}
		})
		It("Copies files from a directory source", func() {
			Expect(elemental.DumpSource(*config, "/dest", types.NewDirSrc("/source"), syncFunc)).To(Succeed())
			Expect(src).To(Equal("/source"))
			Expect(dst).To(Equal("/dest"))
		})
		It("Unpacks a docker image to target", Label("docker"), func() {
			dockerSrc := types.NewDockerSrc("docker/image:latest")
			err := elemental.DumpSource(*config, destDir, dockerSrc, syncFunc)
			Expect(dockerSrc.GetDigest()).To(Equal("fakeDigest"))
			Expect(err).To(BeNil())

			// SyncFunc is not used
			Expect(src).To(BeEmpty())
			Expect(dst).To(BeEmpty())
		})
		It("Unpacks a docker image to target with cosign validation", Label("docker", "cosign"), func() {
			config.Cosign = true
			err := elemental.DumpSource(*config, destDir, types.NewDockerSrc("docker/image:latest"), nil)
			Expect(err).To(BeNil())
			Expect(runner.CmdsMatch([][]string{{"cosign", "verify", "docker/image:latest"}}))
		})
		It("Fails cosign validation", Label("cosign"), func() {
			runner.ReturnError = errors.New("cosign error")
			config.Cosign = true
			err := elemental.DumpSource(*config, destDir, types.NewDockerSrc("docker/image:latest"), nil)
			Expect(err).NotTo(BeNil())
			Expect(runner.CmdsMatch([][]string{{"cosign", "verify", "docker/image:latest"}}))
		})
		It("Fails to unpack a docker image to target", Label("docker"), func() {
			unpackErr := errors.New("failed to unpack")
			extractor.SideEffect = func(_, _, _ string, _, _ bool) (string, error) { return "", unpackErr }
			err := elemental.DumpSource(*config, destDir, types.NewDockerSrc("docker/image:latest"), nil)
			Expect(err).To(Equal(unpackErr))
		})
		It("Copies image file to target", func() {
			sourceImg := "/source.img"
			destFile := filepath.Join(destDir, "active.img")

			err := elemental.DumpSource(*config, destFile, types.NewFileSrc(sourceImg), syncFunc)
			Expect(err).To(BeNil())
			Expect(dst).To(Equal(destFile))
			Expect(src).To(Equal(constants.ImgSrcDir))
		})
		It("Fails to copy, source can't be mounted", func() {
			mounter.ErrorOnMount = true
			err := elemental.DumpSource(*config, "whatever", types.NewFileSrc("/source.img"), nil)
			Expect(err).To(HaveOccurred())
		})
	})
	Describe("CreateImageFromTree", Label("createImg"), func() {
		var imgFile, root string
		var img *types.Image
		var cleaned bool

		BeforeEach(func() {
			cleaned = false
			destDir, err := utils.TempDir(fs, "", "test")
			Expect(err).ShouldNot(HaveOccurred())
			root, err = utils.TempDir(fs, "", "test")
			Expect(err).ShouldNot(HaveOccurred())

			imgFile = filepath.Join(destDir, "dst.img")
			sf, err := fs.Create(filepath.Join(root, "somefile"))
			Expect(err).ShouldNot(HaveOccurred())
			Expect(sf.Truncate(32 * 1024 * 1024)).To(Succeed())
			Expect(sf.Close()).To(Succeed())

			Expect(err).ShouldNot(HaveOccurred())
			img = &types.Image{
				FS:         constants.LinuxImgFs,
				File:       imgFile,
				MountPoint: "/some/mountpoint",
			}
		})
		It("Creates an image including including the root tree contents", func() {
			cleaner := func() error {
				cleaned = true
				return nil
			}
			err := elemental.CreateImageFromTree(*config, img, root, false, cleaner)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(img.Size).To(Equal(32 + constants.ImgOverhead + 1))
			Expect(cleaned).To(BeTrue())
		})
		It("Creates an squashfs image", func() {
			img.FS = constants.SquashFs
			err := elemental.CreateImageFromTree(*config, img, root, false)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(img.Size).To(Equal(uint(0)))
			Expect(runner.IncludesCmds([][]string{{"mksquashfs"}}))
		})
		It("Creates an image of an specific size including including the root tree contents", func() {
			img.Size = 64
			err := elemental.CreateImageFromTree(*config, img, root, false)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(img.Size).To(Equal(uint(64)))
			Expect(runner.IncludesCmds([][]string{{"rsync"}}))
		})
		It("Fails to mount created filesystem image", func() {
			mounter.ErrorOnUnmount = true
			err := elemental.CreateImageFromTree(*config, img, root, false)
			Expect(err).Should(HaveOccurred())
			Expect(img.Size).To(Equal(32 + constants.ImgOverhead + 1))
			Expect(cleaned).To(BeFalse())
			Expect(runner.IncludesCmds([][]string{{"rsync"}}))
		})
	})
	Describe("CopyImgFile", Label("copyimg"), func() {
		var imgFile, srcFile string
		var img *types.Image
		var fileContent []byte
		BeforeEach(func() {
			destDir, err := utils.TempDir(fs, "", "test")
			Expect(err).ShouldNot(HaveOccurred())
			imgFile = filepath.Join(destDir, "dst.img")
			srcFile = filepath.Join(destDir, "src.img")
			fileContent = []byte("imagefile")
			err = fs.WriteFile(srcFile, fileContent, constants.FilePerm)
			Expect(err).ShouldNot(HaveOccurred())
			img = &types.Image{
				Label:  "myLabel",
				FS:     constants.LinuxImgFs,
				File:   imgFile,
				Source: types.NewFileSrc(srcFile),
			}
		})
		It("Copies image file and sets new label", func() {
			err := elemental.CopyFileImg(*config, img)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(runner.IncludesCmds([][]string{{"tune2fs", "-L", img.Label, img.File}})).To(BeNil())
			data, err := fs.ReadFile(imgFile)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(data).To(Equal(fileContent))
		})
		It("Copies image file and without setting a new label", func() {
			img.FS = constants.SquashFs
			err := elemental.CopyFileImg(*config, img)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(runner.IncludesCmds([][]string{{"tune2fs", "-L", img.Label, img.File}})).NotTo(BeNil())
			data, err := fs.ReadFile(imgFile)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(data).To(Equal(fileContent))
		})
		It("Fails to copy image if source is not of file type", func() {
			img.Source = types.NewEmptySrc()
			err := elemental.CopyFileImg(*config, img)
			Expect(err).Should(HaveOccurred())
		})
		It("Fails to copy image if source does not exist", func() {
			img.Source = types.NewFileSrc("whatever")
			err := elemental.CopyFileImg(*config, img)
			Expect(err).Should(HaveOccurred())
		})
		It("Fails to copy image if it can't create target dir", func() {
			img.File = "/new/path.img"
			config.Fs = vfs.NewReadOnlyFS(fs)
			err := elemental.CopyFileImg(*config, img)
			Expect(err).Should(HaveOccurred())
		})
		It("Fails to copy image if it can't write a new file", func() {
			config.Fs = vfs.NewReadOnlyFS(fs)
			err := elemental.CopyFileImg(*config, img)
			Expect(err).Should(HaveOccurred())
		})
	})
	Describe("CheckActiveDeployment", Label("check"), func() {
		BeforeEach(func() {
			Expect(utils.MkdirAll(config.Fs, filepath.Dir(constants.ActiveMode), constants.DirPerm)).To(Succeed())
		})
		It("deployment found on active", func() {
			Expect(fs.WriteFile(constants.ActiveMode, []byte("1"), constants.FilePerm)).To(Succeed())
			Expect(elemental.CheckActiveDeployment(*config)).To(BeTrue())
		})

		It("deployment found on passive", func() {
			Expect(fs.WriteFile(constants.PassiveMode, []byte("1"), constants.FilePerm)).To(Succeed())
			Expect(elemental.CheckActiveDeployment(*config)).To(BeTrue())
		})

		It("deployment found on recovery", func() {
			Expect(fs.WriteFile(constants.RecoveryMode, []byte("1"), constants.FilePerm)).To(Succeed())
			Expect(elemental.CheckActiveDeployment(*config)).To(BeTrue())
		})

		It("Should not error out", func() {
			Expect(elemental.CheckActiveDeployment(*config)).To(BeFalse())
		})
	})
	Describe("SelinuxRelabel", Label("SelinuxRelabel", "selinux"), func() {
		var policyFile string
		var relabelCmd []string
		BeforeEach(func() {
			// to mock the existance of setfiles command on non selinux hosts
			err := utils.MkdirAll(fs, "/usr/sbin", constants.DirPerm)
			Expect(err).ShouldNot(HaveOccurred())
			sbin, err := fs.RawPath("/usr/sbin")
			Expect(err).ShouldNot(HaveOccurred())

			path := os.Getenv("PATH")
			os.Setenv("PATH", fmt.Sprintf("%s:%s", sbin, path))
			_, err = fs.Create("/usr/sbin/setfiles")
			Expect(err).ShouldNot(HaveOccurred())
			err = fs.Chmod("/usr/sbin/setfiles", 0777)
			Expect(err).ShouldNot(HaveOccurred())

			// to mock SELinux policy files
			policyFile = filepath.Join(constants.SELinuxTargetedPolicyPath, "policy.31")
			err = utils.MkdirAll(fs, filepath.Dir(constants.SELinuxTargetedContextFile), constants.DirPerm)
			Expect(err).ShouldNot(HaveOccurred())
			_, err = fs.Create(constants.SELinuxTargetedContextFile)
			Expect(err).ShouldNot(HaveOccurred())
			err = utils.MkdirAll(fs, constants.SELinuxTargetedPolicyPath, constants.DirPerm)
			Expect(err).ShouldNot(HaveOccurred())
			_, err = fs.Create(policyFile)
			Expect(err).ShouldNot(HaveOccurred())

			relabelCmd = []string{
				"setfiles", "-c", policyFile, "-e", "/dev", "-e", "/proc", "-e", "/sys",
				"-F", constants.SELinuxTargetedContextFile, "/",
			}
		})
		It("does nothing if the context file is not found", func() {
			err := fs.Remove(constants.SELinuxTargetedContextFile)
			Expect(err).ShouldNot(HaveOccurred())

			Expect(elemental.SelinuxRelabel(*config, "/", true)).To(BeNil())
			Expect(runner.CmdsMatch([][]string{{}}))
		})
		It("does nothing if the policy file is not found", func() {
			err := fs.Remove(policyFile)
			Expect(err).ShouldNot(HaveOccurred())

			Expect(elemental.SelinuxRelabel(*config, "/", true)).To(BeNil())
			Expect(runner.CmdsMatch([][]string{{}}))
		})
		It("relabels the current root", func() {
			Expect(elemental.SelinuxRelabel(*config, "", true)).To(BeNil())
			Expect(runner.CmdsMatch([][]string{relabelCmd})).To(BeNil())

			runner.ClearCmds()
			Expect(elemental.SelinuxRelabel(*config, "/", true)).To(BeNil())
			Expect(runner.CmdsMatch([][]string{relabelCmd})).To(BeNil())
		})
		It("fails to relabel the current root", func() {
			runner.ReturnError = errors.New("setfiles failure")
			Expect(elemental.SelinuxRelabel(*config, "", true)).NotTo(BeNil())
			Expect(runner.CmdsMatch([][]string{relabelCmd})).To(BeNil())
		})
		It("ignores relabel failures", func() {
			runner.ReturnError = errors.New("setfiles failure")
			Expect(elemental.SelinuxRelabel(*config, "", false)).To(BeNil())
			Expect(runner.CmdsMatch([][]string{relabelCmd})).To(BeNil())
		})
		It("relabels the given root-tree path", func() {
			contextFile := filepath.Join("/root", constants.SELinuxTargetedContextFile)
			err := utils.MkdirAll(fs, filepath.Dir(contextFile), constants.DirPerm)
			Expect(err).ShouldNot(HaveOccurred())
			_, err = fs.Create(contextFile)
			Expect(err).ShouldNot(HaveOccurred())
			policyFile = filepath.Join("/root", policyFile)
			err = utils.MkdirAll(fs, filepath.Join("/root", constants.SELinuxTargetedPolicyPath), constants.DirPerm)
			Expect(err).ShouldNot(HaveOccurred())
			_, err = fs.Create(policyFile)
			Expect(err).ShouldNot(HaveOccurred())

			relabelCmd = []string{
				"setfiles", "-c", policyFile, "-F", "-r", "/root", contextFile, "/root",
			}

			Expect(elemental.SelinuxRelabel(*config, "/root", true)).To(BeNil())
			Expect(runner.CmdsMatch([][]string{relabelCmd})).To(BeNil())
		})
	})
	Describe("GetIso", Label("GetIso", "iso"), func() {
		It("Gets the iso, mounts it and updates image source", func() {
			tmpDir, err := utils.TempDir(fs, "", "elemental-test")
			Expect(err).To(BeNil())
			err = fs.WriteFile(fmt.Sprintf("%s/fake.iso", tmpDir), []byte("Hi"), constants.FilePerm)
			Expect(err).To(BeNil())
			iso := fmt.Sprintf("%s/fake.iso", tmpDir)

			// Create the internal ISO file structure
			rootfsImg := filepath.Join(os.TempDir(), "/elemental/iso", constants.ISORootFile)
			Expect(utils.MkdirAll(fs, filepath.Dir(rootfsImg), constants.DirPerm)).To(Succeed())
			Expect(fs.WriteFile(rootfsImg, []byte{}, constants.FilePerm)).To(Succeed())

			source, isoClean, err := elemental.SourceFormISO(*config, iso)
			Expect(err).To(BeNil())
			Expect(source.IsFile()).To(BeTrue())
			Expect(isoClean()).To(Succeed())
		})
		It("Fails if it cant find the iso", func() {
			iso := "whatever"
			_, isoClean, err := elemental.SourceFormISO(*config, iso)
			Expect(err).ToNot(BeNil())
			Expect(isoClean()).To(Succeed())
		})
		It("Fails creating the mountpoint", func() {
			tmpDir, err := utils.TempDir(fs, "", "elemental-test")
			Expect(err).To(BeNil())
			err = fs.WriteFile(fmt.Sprintf("%s/fake.iso", tmpDir), []byte("Hi"), constants.FilePerm)
			Expect(err).To(BeNil())
			iso := fmt.Sprintf("%s/fake.iso", tmpDir)
			config.Fs = vfs.NewReadOnlyFS(fs)

			_, isoClean, err := elemental.SourceFormISO(*config, iso)
			Expect(err).ToNot(BeNil())
			Expect(isoClean()).To(Succeed())
		})
		It("Fails if it cannot mount the iso", func() {
			mounter.ErrorOnMount = true
			tmpDir, err := utils.TempDir(fs, "", "elemental-test")
			Expect(err).To(BeNil())
			err = fs.WriteFile(fmt.Sprintf("%s/fake.iso", tmpDir), []byte("Hi"), constants.FilePerm)
			Expect(err).To(BeNil())
			iso := fmt.Sprintf("%s/fake.iso", tmpDir)
			_, isoClean, err := elemental.SourceFormISO(*config, iso)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("mount error"))
			Expect(isoClean()).To(Succeed())
		})
	})
	Describe("CloudConfig", Label("CloudConfig", "cloud-config"), func() {
		var parts types.ElementalPartitions
		BeforeEach(func() {
			parts = conf.NewInstallElementalPartitions()
		})
		It("Copies the cloud config file", func() {
			testString := "In a galaxy far far away..."
			cloudInit := []string{"/config.yaml"}
			err := fs.WriteFile(cloudInit[0], []byte(testString), constants.FilePerm)
			Expect(err).To(BeNil())
			Expect(err).To(BeNil())

			err = elemental.CopyCloudConfig(*config, parts.GetConfigStorage(), cloudInit)
			Expect(err).To(BeNil())
			copiedFile, err := fs.ReadFile(fmt.Sprintf("%s/90_custom.yaml", constants.OEMDir))
			Expect(err).To(BeNil())
			Expect(copiedFile).To(ContainSubstring(testString))
		})
		It("Doesnt do anything if the config file is not set", func() {
			err := elemental.CopyCloudConfig(*config, parts.GetConfigStorage(), []string{})
			Expect(err).To(BeNil())
		})
		It("Doesnt do anything if the OEM partition has no mount point", func() {
			parts.OEM.MountPoint = ""
			err := elemental.CopyCloudConfig(*config, parts.GetConfigStorage(), []string{})
			Expect(err).To(BeNil())
		})
	})
	Describe("DeactivateDevices", Label("blkdeactivate"), func() {
		It("calls blkdeactivat", func() {
			err := elemental.DeactivateDevices(*config)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(runner.CmdsMatch([][]string{{
				"blkdeactivate", "--lvmoptions", "retry,wholevg",
				"--dmoptions", "force,retry", "--errors",
			}})).To(BeNil())
		})
	})
	Describe("DeployRecoverySystem", Label("recovery"), func() {
		BeforeEach(func() {
			extractor.SideEffect = func(_, destination, platform string, _, _ bool) (string, error) {
				bootDir := filepath.Join(destination, "boot")
				logger.Debugf("Creating %s", bootDir)
				err := utils.MkdirAll(fs, bootDir, constants.DirPerm)
				Expect(err).ShouldNot(HaveOccurred())

				libDir := "/lib/modules/6.4"
				err = utils.MkdirAll(fs, filepath.Join(destination, libDir), constants.DirPerm)
				Expect(err).ShouldNot(HaveOccurred())

				kernelPath := filepath.Join(libDir, "vmlinuz")
				_, err = fs.Create(filepath.Join(destination, kernelPath))
				Expect(err).ShouldNot(HaveOccurred())

				err = fs.Symlink(filepath.Join(libDir, "vmlinuz"), filepath.Join(bootDir, "vmlinuz-6.4"))
				Expect(err).ShouldNot(HaveOccurred())

				_, err = fs.Create(filepath.Join(bootDir, "initrd"))
				Expect(err).ShouldNot(HaveOccurred())
				return mocks.FakeDigest, err
			}
		})
		It("deploys a recovery system", func() {
			Expect(fs.Mkdir("/recovery", constants.DirPerm)).To(Succeed())
			Expect(fs.Mkdir("/recovery/boot", constants.DirPerm)).To(Succeed())

			img := &types.Image{
				File:   filepath.Join("/recovery", constants.BootDir, constants.RecoveryImgFile),
				Source: types.NewDockerSrc("elemental:latest"),
				FS:     constants.SquashFs,
			}
			err := elemental.DeployRecoverySystem(*config, img)
			Expect(err).ShouldNot(HaveOccurred())

			info, err := fs.Stat("/recovery/boot/vmlinuz")
			Expect(err).ShouldNot(HaveOccurred())
			Expect(info.Mode() & iofs.ModeSymlink).To(BeZero())
		})
	})
})

// PathInMountPoints will check if the given path is in the mountPoints list
func pathInMountPoints(mounter *mocks.FakeMounter, path string) bool {
	mountPoints, _ := mounter.List()
	for _, m := range mountPoints {
		if path == m.Path {
			return true
		}
	}
	return false
}
