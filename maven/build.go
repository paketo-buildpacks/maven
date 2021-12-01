/*
 * Copyright 2018-2020 the original author or authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package maven

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"

	"github.com/buildpacks/libcnb"
	"github.com/paketo-buildpacks/libbs"
	"github.com/paketo-buildpacks/libpak"
	"github.com/paketo-buildpacks/libpak/bard"
	"github.com/paketo-buildpacks/libpak/bindings"
)

type Build struct {
	Logger             bard.Logger
	ApplicationFactory ApplicationFactory
	TTY                bool
}

type ApplicationFactory interface {
	NewApplication(additionalMetadata map[string]interface{}, arguments []string, artifactResolver libbs.ArtifactResolver,
		cache libbs.Cache, command string, bom *libcnb.BOM, applicationPath string) (libbs.Application, error)
}

func (b Build) Build(context libcnb.BuildContext) (libcnb.BuildResult, error) {
	b.Logger.Title(context.Buildpack)
	result := libcnb.NewBuildResult()

	cr, err := libpak.NewConfigurationResolver(context.Buildpack, &b.Logger)
	if err != nil {
		return libcnb.BuildResult{}, fmt.Errorf("unable to create configuration resolver\n%w", err)
	}

	dr, err := libpak.NewDependencyResolver(context)
	if err != nil {
		return libcnb.BuildResult{}, fmt.Errorf("unable to create dependency resolver\n%w", err)
	}

	dc, err := libpak.NewDependencyCache(context)
	if err != nil {
		return libcnb.BuildResult{}, fmt.Errorf("unable to create dependency cache\n%w", err)
	}
	dc.Logger = b.Logger

	command := filepath.Join(context.Application.Path, "mvnw")
	if _, err := os.Stat(command); os.IsNotExist(err) {
		dep, err := dr.Resolve("maven", "")
		if err != nil {
			return libcnb.BuildResult{}, fmt.Errorf("unable to find dependency\n%w", err)
		}

		d, be := NewDistribution(dep, dc)
		d.Logger = b.Logger
		result.Layers = append(result.Layers, d)
		result.BOM.Entries = append(result.BOM.Entries, be)

		command = filepath.Join(context.Layers.Path, d.Name(), "bin", "mvn")
	} else if err != nil {
		return libcnb.BuildResult{}, fmt.Errorf("unable to stat %s\n%w", command, err)
	} else {
		if err := os.Chmod(command, 0755); err != nil {
			return libcnb.BuildResult{}, fmt.Errorf("unable to chmod %s\n%w", command, err)
		}

		if err = b.CleanMvnWrapper(command); err != nil {
			return libcnb.BuildResult{}, fmt.Errorf("unable to clean mvnw file: %s\n%w", command, err)
		}

	}

	u, err := user.Current()
	if err != nil {
		return libcnb.BuildResult{}, fmt.Errorf("unable to determine user home directory\n%w", err)
	}

	c := libbs.Cache{Path: filepath.Join(u.HomeDir, ".m2")}
	c.Logger = b.Logger
	result.Layers = append(result.Layers, c)

	args, err := libbs.ResolveArguments("BP_MAVEN_BUILD_ARGUMENTS", cr)
	if err != nil {
		return libcnb.BuildResult{}, fmt.Errorf("unable to resolve build arguments\n%w", err)
	}

	_, supportMultipleArtifacts := cr.Resolve("BP_MAVEN_SUPPORT_MULTIPLE_ARTIFACTS")
	if supportMultipleArtifacts {
		err = copyAssemblyDistribution(filepath.Join(context.Buildpack.Path, "maven/assembly"), "zip.xml",
			context.Application.Path)
		if err != nil {
			return  libcnb.BuildResult{}, fmt.Errorf("unable to configure maven assembly plugin to " +
				"create one distributable archive\n%w", err)
		}
		nativeSource, _ := cr.Resolve("BP_MAVEN_BUILT_ARTIFACT_NATIVE_SOURCE")
		args = append(args, []string{"assembly:single","-Ddescriptor=zip.xml",fmt.Sprintf("-Doutput.folder=%s",
			nativeSource)}...)
	}

	pomFile, userSet := cr.Resolve("BP_MAVEN_POM_FILE")
	if userSet {
		args = append([]string{"--file", pomFile}, args...)
	}

	if !b.TTY && !contains(args, []string{"-B", "--batch-mode"}) {
		// terminal is not tty, and the user did not set batch mode; let's set it
		args = append([]string{"--batch-mode"}, args...)
	}

	md := map[string]interface{}{}
	if binding, ok, err := bindings.ResolveOne(context.Platform.Bindings, bindings.OfType("maven")); err != nil {
		return libcnb.BuildResult{}, fmt.Errorf("unable to resolve binding\n%w", err)
	} else if ok {
		args, err = handleMavenSettings(binding, args, md)
		if err != nil {
			return libcnb.BuildResult{}, fmt.Errorf("unable to process maven settings from binding\n%w", err)
		}
	}

	art := libbs.ArtifactResolver{
		ArtifactConfigurationKey: "BP_MAVEN_BUILT_ARTIFACT",
		ConfigurationResolver:    cr,
		ModuleConfigurationKey:   "BP_MAVEN_BUILT_MODULE",
		InterestingFileDetector:  libbs.JARInterestingFileDetector{},
	}

	a, err := b.ApplicationFactory.NewApplication(
		md,
		args,
		art,
		c,
		command,
		result.BOM,
		context.Application.Path,
	)
	if err != nil {
		return libcnb.BuildResult{}, fmt.Errorf("unable to create application layer\n%w", err)
	}
	a.Logger = b.Logger
	result.Layers = append(result.Layers, a)

	return result, nil
}

func handleMavenSettings(binding libcnb.Binding, args []string, md map[string]interface{}) ([]string, error) {
	settingsPath, ok := binding.SecretFilePath("settings.xml")
	if !ok {
		return args, nil
	}
	args = append([]string{fmt.Sprintf("--settings=%s", settingsPath)}, args...)

	hasher := sha256.New()
	settingsFile, err := os.Open(settingsPath)
	if err != nil {
		return nil, fmt.Errorf("unable to open settings.xml\n%w", err)
	}
	if _, err := io.Copy(hasher, settingsFile); err != nil {
		return nil, fmt.Errorf("error hashing settings.xml\n%w", err)
	}
	md["settings-sha256"] = hex.EncodeToString(hasher.Sum(nil))

	settingsSecurityPath, ok := binding.SecretFilePath("settings-security.xml")
	if !ok {
		return args, nil
	}
	args = append([]string{fmt.Sprintf("-Dsettings.security=%s", settingsSecurityPath)}, args...)

	hasher.Reset()
	settingsSecurityFile, err := os.Open(settingsSecurityPath)
	if err != nil {
		return nil, fmt.Errorf("unable to open settings-security.xml\n%w", err)
	}
	if _, err := io.Copy(hasher, settingsSecurityFile); err != nil {
		return nil, fmt.Errorf("error hashing settings-security.xml\n%w", err)
	}
	md["settings-security-sha256"] = hex.EncodeToString(hasher.Sum(nil))

	return args, nil
}

func contains(strings []string, stringsSearchedAfter []string) bool {
	for _, v := range strings {
		for _, stringSearchedAfter := range stringsSearchedAfter {
			if v == stringSearchedAfter {
				return true
			}
		}
	}
	return false
}

func (b Build) CleanMvnWrapper(fileName string) error {

	fileContents, err := ioutil.ReadFile(fileName)
	if err != nil {
		return err
	}

	// the mvnw file can contain Windows CRLF line endings, e.g. from a 'git clone' on windows
	// we replace these so that the unix container can execute the wrapper successfully

	// replace CRLF with LF
	fileContents = bytes.ReplaceAll(fileContents, []byte{13}, []byte{})

	err = ioutil.WriteFile(fileName, fileContents, 0755)
	if err != nil {
		return err
	}

	return nil
}

func copyAssemblyDistribution(srcPath string, file string, destPath string) error {
	dist, err := os.Open(filepath.Join(srcPath, file))
	if err != nil {
		return fmt.Errorf("unable to open original distribution file\n%w", err)
	}
	defer dist.Close()
	destination, err := os.Create(filepath.Join(destPath, file))
	if err != nil {
		return fmt.Errorf("unable to open the destination distribution file\n%w", err)
	}
	defer destination.Close()
	_, err = io.Copy(destination, dist)
	if err != nil {
		return  fmt.Errorf("unable to copy file\n%w", err)
	}
	return nil
}
