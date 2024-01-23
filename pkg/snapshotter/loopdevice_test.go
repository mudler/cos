/*
Copyright © 2022 - 2024 SUSE LLC

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

package snapshotter_test

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	conf "github.com/rancher/elemental-toolkit/pkg/config"
	"github.com/rancher/elemental-toolkit/pkg/constants"
	v1mock "github.com/rancher/elemental-toolkit/pkg/mocks"
	"github.com/rancher/elemental-toolkit/pkg/snapshotter"
	v1 "github.com/rancher/elemental-toolkit/pkg/types/v1"
	"github.com/rancher/elemental-toolkit/pkg/utils"
	"github.com/twpayne/go-vfs/v4"
	"github.com/twpayne/go-vfs/v4/vfst"
)

var _ = Describe("LoopDevice", Label("snapshotter", "loopdevice"), func() {
	var cfg v1.Config
	var runner *v1mock.FakeRunner
	var fs vfs.FS
	var logger v1.Logger
	var mounter *v1mock.FakeMounter
	var cleanup func()
	var bootloader *v1mock.FakeBootloader
	var memLog *bytes.Buffer
	var snapCfg v1.SnapshotterConfig
	var rootDir string

	BeforeEach(func() {
		rootDir = "/some/root"
		runner = v1mock.NewFakeRunner()
		mounter = v1mock.NewFakeMounter()
		bootloader = &v1mock.FakeBootloader{}
		memLog = &bytes.Buffer{}
		logger = v1.NewBufferLogger(memLog)
		logger.SetLevel(v1.DebugLevel())

		var err error
		fs, cleanup, err = vfst.NewTestFS(map[string]interface{}{})
		Expect(err).Should(BeNil())

		cfg = *conf.NewConfig(
			conf.WithFs(fs),
			conf.WithRunner(runner),
			conf.WithLogger(logger),
			conf.WithMounter(mounter),
			conf.WithPlatform("linux/amd64"),
		)
		snapCfg = v1.SnapshotterConfig{
			Type:     constants.LoopDeviceSnapshotterType,
			MaxSnaps: constants.MaxSnaps,
			Config:   v1.NewLoopDeviceConfig(),
		}

		Expect(utils.MkdirAll(fs, rootDir, constants.DirPerm)).To(Succeed())
	})

	AfterEach(func() {
		cleanup()
	})

	It("creates a new LoopDevice snapshotter instance", func() {
		Expect(snapshotter.NewLoopDeviceSnapshotter(cfg, snapCfg, bootloader)).Error().NotTo(HaveOccurred())

		// Invalid snapshotter type
		snapCfg.Type = "invalid"
		Expect(snapshotter.NewLoopDeviceSnapshotter(cfg, snapCfg, bootloader)).Error().To(HaveOccurred())

		// Invalid snapshotter type
		snapCfg.Type = constants.LoopDeviceSnapshotterType
		snapCfg.Config = map[string]string{}
		Expect(snapshotter.NewLoopDeviceSnapshotter(cfg, snapCfg, bootloader)).Error().To(HaveOccurred())
	})

	It("inits a snapshotter", func() {
		lp, err := snapshotter.NewLoopDeviceSnapshotter(cfg, snapCfg, bootloader)
		Expect(err).NotTo(HaveOccurred())

		Expect(utils.Exists(fs, filepath.Join(rootDir, ".snapshots"))).To(BeFalse())
		Expect(lp.InitSnapshotter(rootDir)).To(Succeed())
		Expect(utils.Exists(fs, filepath.Join(rootDir, ".snapshots"))).To(BeTrue())
	})

	It("fails to init if it can't create working directories", func() {
		cfg.Fs = vfs.NewReadOnlyFS(fs)
		lp, err := snapshotter.NewLoopDeviceSnapshotter(cfg, snapCfg, bootloader)
		Expect(err).NotTo(HaveOccurred())

		Expect(utils.Exists(fs, filepath.Join(rootDir, ".snapshots"))).To(BeFalse())
		Expect(lp.InitSnapshotter(rootDir)).NotTo(Succeed())
		Expect(utils.Exists(fs, filepath.Join(rootDir, ".snapshots"))).To(BeFalse())
	})

	It("starts a transaction", func() {
		lp, err := snapshotter.NewLoopDeviceSnapshotter(cfg, snapCfg, bootloader)
		Expect(err).NotTo(HaveOccurred())

		Expect(lp.InitSnapshotter(rootDir)).To(Succeed())

		snap, err := lp.StartTransaction()
		Expect(err).NotTo(HaveOccurred())
		Expect(snap.ID).To(Equal(1))
		Expect(snap.InProgress).To(BeTrue())
		Expect(snap.Path).To(Equal(filepath.Join(rootDir, ".snapshots/1/snapshot.img")))
	})

	It("fails to start a transaction without being initiated first", func() {
		lp, err := snapshotter.NewLoopDeviceSnapshotter(cfg, snapCfg, bootloader)
		Expect(err).NotTo(HaveOccurred())

		Expect(lp.StartTransaction()).Error().To(HaveOccurred())
	})

	It("fails to start a transaction if working directory bind mount fails", func() {
		lp, err := snapshotter.NewLoopDeviceSnapshotter(cfg, snapCfg, bootloader)
		Expect(err).NotTo(HaveOccurred())

		mounter.ErrorOnMount = true

		Expect(lp.InitSnapshotter(rootDir)).To(Succeed())
		Expect(lp.StartTransaction()).Error().To(HaveOccurred())
	})

	It("fails to get available snapshots on a not initated system", func() {
		lp, err := snapshotter.NewLoopDeviceSnapshotter(cfg, snapCfg, bootloader)
		Expect(err).NotTo(HaveOccurred())

		Expect(lp.GetSnapshots()).Error().To(HaveOccurred())
	})

	Describe("using loopdevice on sixth snapshot", func() {
		var err error
		var lp *snapshotter.LoopDevice
		var snapshotsPrefix string

		BeforeEach(func() {
			var snapshotFile string
			var i int
			snapshotsPrefix = filepath.Join(rootDir, ".snapshots")
			for i = 1; i < 6; i++ {
				Expect(utils.MkdirAll(cfg.Fs, filepath.Join(rootDir, ".snapshots", strconv.Itoa(i)), constants.DirPerm)).To(Succeed())
				snapshotFile = filepath.Join(snapshotsPrefix, strconv.Itoa(i), "snapshot.img")
				Expect(fs.WriteFile(snapshotFile, []byte(fmt.Sprintf("This is snapshot %d", i)), constants.FilePerm)).To(Succeed())
			}
			Expect(fs.Symlink(filepath.Join(strconv.Itoa(5), "snapshot.img"), filepath.Join(snapshotsPrefix, constants.ActiveSnapshot))).To(Succeed())
			passivesPath := filepath.Join(snapshotsPrefix, "passives")
			Expect(utils.MkdirAll(fs, passivesPath, constants.DirPerm))
			for i = 1; i < 5; i++ {
				snapshotFile = filepath.Join("..", strconv.Itoa(i), "snapshot.img")
				Expect(fs.Symlink(snapshotFile, filepath.Join(passivesPath, fmt.Sprintf(constants.PassiveSnapshot, i)))).To(Succeed())
			}

			runner.SideEffect = func(cmd string, args ...string) ([]byte, error) {
				if cmd == "losetup" {
					return []byte(".snapshots/5/snapshot.img"), nil
				}
				return []byte(""), nil
			}

			lp, err = snapshotter.NewLoopDeviceSnapshotter(cfg, snapCfg, bootloader)
			Expect(err).NotTo(HaveOccurred())
			Expect(lp.InitSnapshotter(rootDir)).To(Succeed())
		})

		It("gets current snapshots", func() {
			Expect(lp.GetSnapshots()).To(Equal([]int{1, 2, 3, 4, 5}))
		})

		It("starts a transaction with the expected snapshot values", func() {
			snap, err := lp.StartTransaction()
			Expect(err).NotTo(HaveOccurred())
			Expect(snap.ID).To(Equal(6))
			Expect(snap.InProgress).To(BeTrue())
		})

		It("fails to start a transaction if active snapshot can't be detected", func() {
			// delete current active symlink and create a broken one
			activeLink := filepath.Join(snapshotsPrefix, constants.ActiveSnapshot)
			Expect(fs.Remove(activeLink)).To(Succeed())
			Expect(fs.Symlink("nonExistingFile", activeLink)).To(Succeed())

			_, err = lp.StartTransaction()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("nonExistingFile: no such file or directory"))
		})

		It("closes a transaction on error with a nil snapshot", func() {
			_, err := lp.StartTransaction()
			Expect(err).NotTo(HaveOccurred())
			Expect(lp.CloseTransactionOnError(nil)).To(Succeed())
		})

		It("closes a transaction on error", func() {
			snap, err := lp.StartTransaction()
			Expect(err).NotTo(HaveOccurred())
			Expect(lp.CloseTransactionOnError(snap)).To(Succeed())
		})

		It("closes a transaction on error and errors out umounting snapshot", func() {
			mounter.ErrorOnUnmount = true
			snap, err := lp.StartTransaction()
			Expect(err).NotTo(HaveOccurred())
			Expect(lp.CloseTransactionOnError(snap)).NotTo(Succeed())
		})

		It("deletes a passiev snapshot", func() {
			Expect(lp.DeleteSnapshot(4)).To(Succeed())
			Expect(lp.GetSnapshots()).To(Equal([]int{1, 2, 3, 5}))
		})

		It("fails to delete current snapshot", func() {
			Expect(lp.DeleteSnapshot(5)).NotTo(Succeed())
		})

		It("deletes nothing for non existing snapshots", func() {
			Expect(lp.DeleteSnapshot(99)).To(Succeed())
			Expect(memLog.String()).To(ContainSubstring("nothing to delete"))
		})

		It("closes a started transaction and cleans old snapshots", func() {
			Expect(lp.GetSnapshots()).To(Equal([]int{1, 2, 3, 4, 5}))
			snap, err := lp.StartTransaction()
			Expect(err).NotTo(HaveOccurred())
			Expect(snap.ID).To(Equal(6))
			Expect(snap.InProgress).To(BeTrue())
			Expect(lp.CloseTransaction(snap)).To(Succeed())
			Expect(lp.GetSnapshots()).To(Equal([]int{3, 4, 5, 6}))
		})

		It("closes a started transaction and cleans old snapshots up to current active", func() {
			// Snapshot 2 is the current one
			runner.SideEffect = func(cmd string, args ...string) ([]byte, error) {
				if cmd == "losetup" {
					return []byte(".snapshots/2/snapshot.img"), nil
				}
				return []byte(""), nil
			}

			Expect(lp.GetSnapshots()).To(Equal([]int{1, 2, 3, 4, 5}))
			snap, err := lp.StartTransaction()
			Expect(err).NotTo(HaveOccurred())
			Expect(snap.ID).To(Equal(6))
			Expect(snap.InProgress).To(BeTrue())
			Expect(lp.CloseTransaction(snap)).To(Succeed())

			// Could not delete 2 as it is in use and stopped cleaning
			Expect(lp.GetSnapshots()).To(Equal([]int{2, 3, 4, 5, 6}))
		})

		It("closes and drops a started transaction if snapshot is not in progress", func() {
			Expect(lp.GetSnapshots()).To(Equal([]int{1, 2, 3, 4, 5}))
			snap, err := lp.StartTransaction()
			Expect(err).NotTo(HaveOccurred())
			Expect(snap.ID).To(Equal(6))
			Expect(snap.InProgress).To(BeTrue())

			snap.InProgress = false
			Expect(lp.CloseTransaction(snap)).To(Succeed())
			Expect(lp.GetSnapshots()).To(Equal([]int{1, 2, 3, 4, 5}))
		})

		It("fails closing a transaction, can't unmount snapshot", func() {
			Expect(lp.GetSnapshots()).To(Equal([]int{1, 2, 3, 4, 5}))
			snap, err := lp.StartTransaction()
			Expect(err).NotTo(HaveOccurred())
			Expect(snap.ID).To(Equal(6))
			Expect(snap.InProgress).To(BeTrue())

			mounter.ErrorOnUnmount = true

			Expect(lp.CloseTransaction(snap)).NotTo(Succeed())
			Expect(lp.GetSnapshots()).To(Equal([]int{1, 2, 3, 4, 5}))
		})

		It("fails closing a transaction, can't create image from tree", func() {
			Expect(lp.GetSnapshots()).To(Equal([]int{1, 2, 3, 4, 5}))
			snap, err := lp.StartTransaction()
			Expect(err).NotTo(HaveOccurred())
			Expect(snap.ID).To(Equal(6))
			Expect(snap.InProgress).To(BeTrue())

			snap.WorkDir = "nonExistingPath"

			Expect(lp.CloseTransaction(snap)).NotTo(Succeed())
			Expect(lp.GetSnapshots()).To(Equal([]int{1, 2, 3, 4, 5}))
		})
	})
})
