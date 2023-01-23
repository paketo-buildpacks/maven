package maven

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/buildpacks/libcnb"
	"github.com/paketo-buildpacks/libpak"
	"github.com/paketo-buildpacks/libpak/bard"
)

// MavenManager manages the lifecycle of a Maven distribution
type MavenManager interface {
	ShouldInstall() bool
	Install() (string, libcnb.LayerContributor, *libcnb.BOMEntry, error)
}

// DaemonMavenManager provides the Maven daemon based Maven distribution
type DaemonMavenManager struct {
	configResolver libpak.ConfigurationResolver
	depCache       libpak.DependencyCache
	depResolver    libpak.DependencyResolver
	layersPath     string
	logger         bard.Logger
}

func NewDaemonMavenManager(configResolver libpak.ConfigurationResolver, depResolver libpak.DependencyResolver, depCache libpak.DependencyCache, layersPath string, logger bard.Logger) DaemonMavenManager {
	return DaemonMavenManager{
		configResolver: configResolver,
		depResolver:    depResolver,
		depCache:       depCache,
		layersPath:     layersPath,
		logger:         logger,
	}
}

// ShouldInstall the Maven Daemon
func (d DaemonMavenManager) ShouldInstall() bool {
	return d.configResolver.ResolveBool("BP_MAVEN_DAEMON_ENABLED")
}

// Install the Maven daemon tool
func (d DaemonMavenManager) Install() (string, libcnb.LayerContributor, *libcnb.BOMEntry, error) {
	dep, err := d.depResolver.Resolve("mvnd", "")
	if err != nil {
		return "", nil, nil, fmt.Errorf("unable to find dependency\n%w", err)
	}

	dist, be := NewMvndDistribution(dep, d.depCache)
	dist.Logger = d.logger

	command := filepath.Join(d.layersPath, dist.Name(), "bin", "mvnd")

	return command, dist, &be, nil
}

// StandardMavenManager provides the standard JVM-based Maven distribution
type StandardMavenManager struct {
	appPath        string
	configResolver libpak.ConfigurationResolver
	depCache       libpak.DependencyCache
	depResolver    libpak.DependencyResolver
	layersPath     string
	logger         bard.Logger
}

func NewStandardMavenManager(appPath string, configResolver libpak.ConfigurationResolver, depResolver libpak.DependencyResolver, depCache libpak.DependencyCache, layersPath string, logger bard.Logger) StandardMavenManager {
	return StandardMavenManager{
		appPath:        appPath,
		configResolver: configResolver,
		depResolver:    depResolver,
		depCache:       depCache,
		layersPath:     layersPath,
		logger:         logger,
	}
}

// ShouldInstall the standard JVM-based Maven distribution
func (s StandardMavenManager) ShouldInstall() bool {
	command := filepath.Join(s.appPath, "mvnw")
	_, err := os.Stat(command)
	mvnwNotFound := os.IsNotExist(err)

	_, err = exec.LookPath("mvn")
	mvnNotOnPath := err != nil // or lookup failure

	return mvnwNotFound && mvnNotOnPath
}

// Install the standard JVM-based Maven distribution
func (s StandardMavenManager) Install() (string, libcnb.LayerContributor, *libcnb.BOMEntry, error) {
	version, _ := s.configResolver.Resolve("BP_MAVEN_VERSION")

	dep, err := s.depResolver.Resolve("maven", version)
	if err != nil {
		return "", nil, nil, fmt.Errorf("unable to find dependency\n%w", err)
	}

	dist, be := NewDistribution(dep, s.depCache)
	dist.Logger = s.logger

	command := filepath.Join(s.layersPath, dist.Name(), "bin", "mvn")

	return command, dist, &be, nil
}

// WrapperMavenManager provides Maven through the Maven Wrapper
type WrapperMavenManager struct {
	appPath string
	logger  bard.Logger
}

func NewWrapperMavenManager(appPath string, logger bard.Logger) WrapperMavenManager {
	return WrapperMavenManager{
		appPath: appPath,
		logger:  logger,
	}
}

// ShouldInstall the Maven Wrapper
func (w WrapperMavenManager) ShouldInstall() bool {
	command := filepath.Join(w.appPath, "mvnw")
	_, err := os.Stat(command)
	return err == nil
}

// Install the Maven wrapper tool
// Slightly misleading as this doesn't install anything, it just makes sure the wrapper can be run
// The wrapper itself handles any installation, if it's necessary
func (w WrapperMavenManager) Install() (string, libcnb.LayerContributor, *libcnb.BOMEntry, error) {
	command := filepath.Join(w.appPath, "mvnw")

	if err := os.Chmod(command, 0755); err != nil {
		w.logger.Bodyf("WARNING: unable to chmod %s:\n%s", command, err)
	}

	if err := w.cleanMvnWrapper(command); err != nil {
		w.logger.Bodyf("WARNING: unable to clean mvnw file: %s\n%s", command, err)
	}

	wrapperProperties := filepath.Join(w.appPath, ".mvn/wrapper/maven-wrapper.properties")
	if _, err := os.Stat(wrapperProperties); err == nil {
		if err := w.cleanMvnWrapper(wrapperProperties); err != nil {
                	w.logger.Bodyf("WARNING: unable to clean wrapper properties file: %s\n%s", wrapperProperties, err)
        	}
	}

	return command, nil, nil, nil
}

func (w WrapperMavenManager) cleanMvnWrapper(fileName string) error {
	fileContents, err := os.ReadFile(fileName)
	if err != nil {
		return err
	}
	// the mvnw file can contain Windows CRLF line endings, e.g. from a 'git clone' on windows
	// we replace these so that the unix container can execute the wrapper successfully

	// replace CRLF with LF
	fileContents = bytes.ReplaceAll(fileContents, []byte{13}, []byte{})

	err = os.WriteFile(fileName, fileContents, 0755)
	if err != nil {
		return err
	}

	return nil
}

// NoopMavenManager doesn't provide Maven, but expects it to exist on the path
type NoopMavenManager struct {
	logger bard.Logger
}

func NewNoopMavenManager(logger bard.Logger) NoopMavenManager {
	return NoopMavenManager{
		logger: logger,
	}
}

// ShouldInstall determines if Maven is on the $PATH
func (n NoopMavenManager) ShouldInstall() bool {
	path, err := exec.LookPath("mvn")
	return path != "" && err == nil
}

// Install nothing.
// Slightly misleading as this doesn't install anything, it just makes sure mvn is on the $PATH
func (n NoopMavenManager) Install() (string, libcnb.LayerContributor, *libcnb.BOMEntry, error) {
	command, err := exec.LookPath("mvn")
	if err != nil {
		return "", nil, nil, fmt.Errorf("unable to lookup 'mvn'\n%w", err)
	}

	if command == "" {
		return "", nil, nil, fmt.Errorf("unable to find 'mvn' on $PATH")
	}

	return command, nil, nil, nil
}
