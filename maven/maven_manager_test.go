package maven_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/buildpacks/libcnb"
	. "github.com/onsi/gomega"
	"github.com/paketo-buildpacks/libpak"
	"github.com/paketo-buildpacks/maven/v6/maven"
	"github.com/sclevine/spec"
)

func testMavenManager(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		ctx          libcnb.BuildContext
		mavenManager maven.MavenManager
		mvnwFilepath string
	)

	it.Before(func() {
		var err error

		ctx.Application.Path, err = os.MkdirTemp("", "build-application")
		Expect(err).NotTo(HaveOccurred())

		ctx.Layers.Path, err = os.MkdirTemp("", "build-layers")
		Expect(err).NotTo(HaveOccurred())

		mvnwFilepath = filepath.Join(ctx.Application.Path, "mvnw")
	})

	it.After(func() {
		Expect(os.RemoveAll(ctx.Application.Path)).To(Succeed())
		Expect(os.RemoveAll(ctx.Layers.Path)).To(Succeed())
	})

	context("StandardMavenManager", func() {
		it.Before(func() {
			dep := libpak.BuildpackDependency{
				URI:     "https://localhost/stub-maven-distribution.tar.gz",
				SHA256:  "31ba45356e22aff670af88170f43ff82328e6f323c3ce891ba422bd1031e3308",
				Version: "1.1.1",
				ID:      "maven",
				Name:    "Maven",
			}
			dc := libpak.DependencyCache{CachePath: "testdata"}

			mavenManager = maven.NewStandardMavenManager(
				ctx.Application.Path,
				libpak.ConfigurationResolver{},
				libpak.DependencyResolver{
					Dependencies: []libpak.BuildpackDependency{dep},
					StackID:      "test-stack",
				},
				dc,
				"/layers")
		})

		it("shouldn't install", func() {
			Expect(os.WriteFile(mvnwFilepath, []byte{}, 0644)).ToNot(HaveOccurred())
			Expect(mavenManager.ShouldInstall()).To(BeFalse())
		})

		it("should install", func() {
			t.Setenv("PATH", "/doesnt-exist")
			Expect(mavenManager.ShouldInstall()).To(BeTrue())
		})

		it("installs OK", func() {
			cmd, layerContrib, _, err := mavenManager.Install()
			Expect(err).NotTo(HaveOccurred())

			Expect(cmd).To(Equal("/layers/maven/bin/mvn"))

			Expect(layerContrib.Name()).To(Equal("maven"))
			Expect(layerContrib.(maven.Distribution)).ToNot(BeNil())
		})
	})

	context("DaemonMavenManager", func() {
		var cfg libpak.BuildpackConfiguration

		it.Before(func() {
			dep := libpak.BuildpackDependency{
				URI:     "https://localhost/stub-mvnd-distribution.zip",
				SHA256:  "75458bf0354fde2c9762366e7d952489587e9d618630100b432a5486c4d22664",
				Version: "1.1.1",
				ID:      "mvnd",
				Name:    "Maven",
			}
			dc := libpak.DependencyCache{CachePath: "testdata"}
			cfg = libpak.BuildpackConfiguration{
				Name:    "BP_MAVEN_DAEMON_ENABLED",
				Default: "true",
				Build:   true,
			}

			mavenManager = maven.NewDaemonMavenManager(
				libpak.ConfigurationResolver{
					Configurations: []libpak.BuildpackConfiguration{cfg},
				},
				libpak.DependencyResolver{
					Dependencies: []libpak.BuildpackDependency{dep},
					StackID:      "test-stack",
				},
				dc,
				"/layers")
		})

		it("should install", func() {
			Expect(mavenManager.ShouldInstall()).To(BeTrue())
		})

		it("shouldn't install", func() {
			cfg.Default = "false"
			Expect(mavenManager.ShouldInstall()).To(BeTrue())
		})

		it("installs OK", func() {
			cmd, layerContrib, _, err := mavenManager.Install()
			Expect(err).NotTo(HaveOccurred())

			Expect(cmd).To(Equal("/layers/mvnd/bin/mvnd"))

			Expect(layerContrib.Name()).To(Equal("mvnd"))
			Expect(layerContrib.(maven.MvndDistribution)).ToNot(BeNil())
		})
	})

	context("WrapperMavenManager", func() {
		it.Before(func() {
			dc := libpak.DependencyCache{CachePath: "testdata"}

			mavenManager = maven.NewWrapperMavenManager(
				ctx.Application.Path,
				libpak.ConfigurationResolver{},
				libpak.DependencyResolver{},
				dc)
		})

		it("should install", func() {
			Expect(os.WriteFile(mvnwFilepath, []byte{}, 0644)).ToNot(HaveOccurred())
			Expect(mavenManager.ShouldInstall()).To(BeTrue())
		})

		it("shouldn't install", func() {
			Expect(mavenManager.ShouldInstall()).To(BeFalse())
		})

		it("installs OK", func() {
			Expect(os.WriteFile(mvnwFilepath, []byte{}, 0644)).ToNot(HaveOccurred())
			cmd, layerContrib, _, err := mavenManager.Install()
			Expect(err).NotTo(HaveOccurred())

			Expect(cmd).To(Equal(mvnwFilepath))

			Expect(layerContrib).To(BeNil())

			// makes sure that mvnw is executable
			fi, err := os.Stat(mvnwFilepath)
			Expect(err).NotTo(HaveOccurred())
			Expect(fi.Mode()).To(BeEquivalentTo(0755))
		})

		it("proceeds without error if mvnw could not have been made executable", func() {
			if _, err := os.Stat("/dev/null"); errors.Is(err, os.ErrNotExist) {
				t.Skip("No /dev/null thus not a unix system. Skipping chmod test.")
			}
			Expect(os.Symlink("/dev/null", mvnwFilepath)).To(Succeed())
			fi, err := os.Stat(mvnwFilepath)
			Expect(err).NotTo(HaveOccurred())
			originalMode := fi.Mode()
			Expect(originalMode).ToNot(BeEquivalentTo(0755))
			ctx.StackID = "test-stack-id"

			_, _, _, err = mavenManager.Install()
			Expect(err).NotTo(HaveOccurred())

			fi, err = os.Stat(mvnwFilepath)
			Expect(err).NotTo(HaveOccurred())
			Expect(fi.Mode()).To(BeEquivalentTo(originalMode))
		})

		it("converts CRLF formatting in the mvnw file to LF (unix) if present", func() {
			Expect(os.WriteFile(mvnwFilepath, []byte("test\r\n"), 0644)).To(Succeed())
			ctx.StackID = "test-stack-id"

			_, _, _, err := mavenManager.Install()
			Expect(err).NotTo(HaveOccurred())

			contents, err := os.ReadFile(mvnwFilepath)
			Expect(err).NotTo(HaveOccurred())
			Expect(bytes.Compare(contents, []byte("test\n"))).To(Equal(0))

		})

		it("does not perform format conversion in the mvnw file if not required", func() {
			Expect(os.WriteFile(mvnwFilepath, []byte("test\n"), 0644)).To(Succeed())
			ctx.StackID = "test-stack-id"

			_, _, _, err := mavenManager.Install()
			Expect(err).NotTo(HaveOccurred())

			contents, err := os.ReadFile(mvnwFilepath)
			Expect(err).NotTo(HaveOccurred())
			Expect(bytes.Compare(contents, []byte("test\n"))).To(Equal(0))
		})
	})

	context("NoopMavenManager", func() {
		var addToPath string
		var mvnFilePath string

		it.Before(func() {
			var err error
			addToPath, err = os.MkdirTemp("", "add-to-path")
			Expect(err).NotTo(HaveOccurred())

			t.Setenv("PATH", addToPath)

			mvnFilePath = filepath.Join(addToPath, "mvn")

			mavenManager = maven.NewNoopMavenManager()
		})

		it("should install", func() {
			Expect(os.WriteFile(mvnFilePath, []byte{}, 0755)).ToNot(HaveOccurred())
			Expect(mavenManager.ShouldInstall()).To(BeTrue())
		})

		it("shouldn't install", func() {
			Expect(mavenManager.ShouldInstall()).To(BeFalse())
		})

		it("installs OK", func() {
			Expect(os.WriteFile(mvnFilePath, []byte{}, 0755)).ToNot(HaveOccurred())
			cmd, layerContrib, _, err := mavenManager.Install()
			Expect(err).NotTo(HaveOccurred())

			Expect(cmd).To(Equal(mvnFilePath))

			Expect(layerContrib).To(BeNil())

			// makes sure that mvn is executable
			fi, err := os.Stat(mvnFilePath)
			Expect(err).NotTo(HaveOccurred())
			Expect(fi.Mode()).To(BeEquivalentTo(0755))
		})
	})
}
