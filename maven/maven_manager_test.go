package maven_test

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/buildpacks/libcnb"
	. "github.com/onsi/gomega"
	"github.com/paketo-buildpacks/libpak"
	"github.com/paketo-buildpacks/libpak/bard"
	"github.com/paketo-buildpacks/maven/v6/maven"
	"github.com/sclevine/spec"
)

func testMavenManager(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		ctx          libcnb.BuildContext
		mavenManager maven.MavenManager
		mvnwFilepath string
		mvnwPropsPath string
	)

	it.Before(func() {
		var err error

		ctx.Application.Path, err = os.MkdirTemp("", "build-application")
		Expect(err).NotTo(HaveOccurred())

		ctx.Layers.Path, err = os.MkdirTemp("", "build-layers")
		Expect(err).NotTo(HaveOccurred())

		mvnwFilepath = filepath.Join(ctx.Application.Path, "mvnw")
		mvnwPropsPath = filepath.Join(ctx.Application.Path, ".mvn/wrapper/") 
		
		err = os.MkdirAll(mvnwPropsPath, 0755)
		Expect(err).NotTo(HaveOccurred())
	})

	it.After(func() {
		Expect(os.RemoveAll(ctx.Application.Path)).To(Succeed())
		Expect(os.RemoveAll(ctx.Layers.Path)).To(Succeed())
	})

	context("StandardMavenManager", func() {
		var dc libpak.DependencyCache
		var dep3, dep4, dep5 libpak.BuildpackDependency

		it.Before(func() {
			dep3 = libpak.BuildpackDependency{
				URI:     "https://localhost/stub-maven-distribution.tar.gz",
				SHA256:  "31ba45356e22aff670af88170f43ff82328e6f323c3ce891ba422bd1031e3308",
				Version: "3.3.3",
				ID:      "maven",
				Name:    "Maven",
			}
			dep4 = libpak.BuildpackDependency{
				URI:     "https://localhost/stub-maven-distribution.tar.gz",
				SHA256:  "31ba45356e22aff670af88170f43ff82328e6f323c3ce891ba422bd1031e3308",
				Version: "4.4.4",
				ID:      "maven",
				Name:    "Maven",
			}
			dep5 = libpak.BuildpackDependency{
				URI:     "https://localhost/stub-maven-distribution.tar.gz",
				SHA256:  "31ba45356e22aff670af88170f43ff82328e6f323c3ce891ba422bd1031e3308",
				Version: "5.5.5",
				ID:      "maven",
				Name:    "Maven",
			}
			dc = libpak.DependencyCache{CachePath: "testdata"}

			mavenManager = maven.NewStandardMavenManager(
				ctx.Application.Path,
				libpak.ConfigurationResolver{
					Configurations: []libpak.BuildpackConfiguration{
						{
							Build:   true,
							Launch:  false,
							Default: "3",
							Name:    "BP_MAVEN_VERSION",
						},
					},
				},
				libpak.DependencyResolver{
					Dependencies: []libpak.BuildpackDependency{dep3, dep5},
					StackID:      "test-stack",
				},
				dc,
				"/layers",
				bard.NewLogger(io.Discard))
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

		context("user sets version to 4", func() {
			it.Before(func() {
				t.Setenv("BP_MAVEN_VERSION", "4")

				mavenManager = maven.NewStandardMavenManager(
					ctx.Application.Path,
					libpak.ConfigurationResolver{},
					libpak.DependencyResolver{
						Dependencies: []libpak.BuildpackDependency{dep4, dep5},
						StackID:      "test-stack",
					},
					dc,
					"/layers",
					bard.NewLogger(io.Discard))
			})

			it("installs a specific version", func() {
				cmd, layerContrib, _, err := mavenManager.Install()
				Expect(err).NotTo(HaveOccurred())

				Expect(cmd).To(Equal("/layers/maven/bin/mvn"))

				Expect(layerContrib.Name()).To(Equal("maven"))
				Expect(layerContrib.(maven.Distribution)).ToNot(BeNil())
			})
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
				"/layers",
				bard.NewLogger(io.Discard))
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
			mavenManager = maven.NewWrapperMavenManager(
				ctx.Application.Path,
				bard.NewLogger(io.Discard))
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

		it("converts CRLF formatting in the maven-wrapper.properties file to LF (unix) if present", func() {
			propsFile := filepath.Join(mvnwPropsPath, "maven-wrapper.properties")
			Expect(os.WriteFile(propsFile, []byte("test\r\n"), 0755)).To(Succeed())
	
			_, _, _, err := mavenManager.Install()
                        Expect(err).NotTo(HaveOccurred())

                        contents, err := os.ReadFile(propsFile)
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

			mavenManager = maven.NewNoopMavenManager(bard.NewLogger(io.Discard))
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
