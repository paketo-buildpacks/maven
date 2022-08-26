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

	"github.com/paketo-buildpacks/libpak/sbom"

	"github.com/paketo-buildpacks/libpak/effect"

	"github.com/buildpacks/libcnb"
	"github.com/paketo-buildpacks/libbs"
	"github.com/paketo-buildpacks/libpak"
	"github.com/paketo-buildpacks/libpak/bard"
	"github.com/paketo-buildpacks/libpak/bindings"
)

const (
	Command  = "Command"
	RunBuild = "RunBuild"
)

type Build struct {
	Logger             bard.Logger
	ApplicationFactory ApplicationFactory
	TTY                bool
}

type ApplicationFactory interface {
	NewApplication(additionalMetadata map[string]interface{}, arguments []string, artifactResolver libbs.ArtifactResolver,
		cache libbs.Cache, command string, bom *libcnb.BOM, applicationPath string, bomScanner sbom.SBOMScanner) (libbs.Application, error)
}

func install(b Build, context libcnb.BuildContext, artifact string, securityArgs []string) (string, libcnb.LayerContributor, libcnb.BOMEntry, error) {
	dr, err := libpak.NewDependencyResolver(context)
	if err != nil {
		return "", nil, libcnb.BOMEntry{}, fmt.Errorf("unable to create dependency resolver\n%w", err)
	}

	dc, err := libpak.NewDependencyCache(context)
	if err != nil {
		return "", nil, libcnb.BOMEntry{}, fmt.Errorf("unable to create dependency cache\n%w", err)
	}
	dc.Logger = b.Logger

	dep, err := dr.Resolve(artifact, "")
	if err != nil {
		return "", nil, libcnb.BOMEntry{}, fmt.Errorf("unable to find dependency\n%w", err)
	}

	if artifact == "maven" {
		dist, be := NewDistribution(dep, dc)
		dist.SecurityArgs = securityArgs
		dist.Logger = b.Logger
		command := filepath.Join(context.Layers.Path, dist.Name(), "bin", "mvn")
		return command, dist, be, nil
	}
	dist, be := NewDistribution(dep, dc)
	dist.SecurityArgs = securityArgs
	dist.Logger = b.Logger
	command := filepath.Join(context.Layers.Path, dist.Name(), "bin", "mvnd")
	return command, dist, be, nil
}

func (b Build) Build(context libcnb.BuildContext) (libcnb.BuildResult, error) {
	b.Logger.Title(context.Buildpack)
	result := libcnb.NewBuildResult()

	pr := libpak.PlanEntryResolver{
		Plan: context.Plan,
	}
	runBuild := true
	entry, ok, err := pr.Resolve(PlanEntryMaven)
	if ok && err == nil {
		if runBuildValue, ok := entry.Metadata[RunBuild].(bool); ok {
			runBuild = runBuildValue
		}
	}
	mavenCommand := ""
	if command, ok := entry.Metadata[Command].(string); ok {
		mavenCommand = command
	}

	cr, err := libpak.NewConfigurationResolver(context.Buildpack, &b.Logger)
	if err != nil {
		return libcnb.BuildResult{}, fmt.Errorf("unable to create configuration resolver\n%w", err)
	}

	// no install requested and no build requested
	if mavenCommand == "" && !runBuild {
		return libcnb.BuildResult{}, nil
	}

	md := map[string]interface{}{}
	securityArgs := []string{}
	if binding, ok, err := bindings.ResolveOne(context.Platform.Bindings, bindings.OfType("maven")); err != nil {
		return libcnb.BuildResult{}, fmt.Errorf("unable to resolve binding\n%w", err)
	} else if ok {
		securityArgs, err = handleMavenSettings(binding, securityArgs, md)
		if err != nil {
			return libcnb.BuildResult{}, fmt.Errorf("unable to process maven settings from binding\n%w", err)
		}
	}

	if mavenCommand == "maven" || mavenCommand == "mvnd" {
		cmd, layer, bomEntry, err := install(b, context, mavenCommand, securityArgs)
		if cmd == "" {
			return libcnb.BuildResult{}, fmt.Errorf("unable to install dependency\n%w", err)
		}
		result.Layers = append(result.Layers, layer)
		result.BOM.Entries = append(result.BOM.Entries, bomEntry)
	}

	u, err := user.Current()
	if err != nil {
		return libcnb.BuildResult{}, fmt.Errorf("unable to determine user home directory\n%w", err)
	}

	c := libbs.Cache{Path: filepath.Join(u.HomeDir, ".m2")}
	c.Logger = b.Logger

	if runBuild {
		command := ""
		if cr.ResolveBool("BP_MAVEN_DAEMON_ENABLED") && mavenCommand != "mvnd" {
			cmd, layer, bomEntry, err := install(b, context, "mvnd", securityArgs)
			if err != nil {
				return libcnb.BuildResult{}, err
			}
			result.Layers = append(result.Layers, layer)
			result.BOM.Entries = append(result.BOM.Entries, bomEntry)
			command = cmd
		} else {
			command = filepath.Join(context.Application.Path, "mvnw")
			if _, err := os.Stat(command); os.IsNotExist(err) && mavenCommand != "maven" {
				cmd, layer, bomEntry, err := install(b, context, "maven", securityArgs)
				if err != nil {
					return libcnb.BuildResult{}, err
				}
				result.Layers = append(result.Layers, layer)
				result.BOM.Entries = append(result.BOM.Entries, bomEntry)
				command = cmd
			} else if err != nil {
				return libcnb.BuildResult{}, fmt.Errorf("unable to stat %s\n%w", command, err)
			} else {
				if err := os.Chmod(command, 0755); err != nil {
					b.Logger.Bodyf("WARNING: unable to chmod %s:\n%s", command, err)
				}

				if err = b.CleanMvnWrapper(command); err != nil {
					b.Logger.Bodyf("WARNING: unable to clean mvnw file: %s\n%s", command, err)
				}
			}
		}

		result.Layers = append(result.Layers, c)

		args, err := libbs.ResolveArguments("BP_MAVEN_BUILD_ARGUMENTS", cr)
		if err != nil {
			return libcnb.BuildResult{}, fmt.Errorf("unable to resolve build arguments\n%w", err)
		}

		pomFile, userSet := cr.Resolve("BP_MAVEN_POM_FILE")
		if userSet {
			args = append([]string{"--file", pomFile}, args...)
		}

		if !b.TTY && !contains(args, []string{"-B", "--batch-mode"}) {
			// terminal is not tty, and the user did not set batch mode; let's set it
			args = append([]string{"--batch-mode"}, args...)
		}

		args = append(securityArgs, args...)

		art := libbs.ArtifactResolver{
			ArtifactConfigurationKey: "BP_MAVEN_BUILT_ARTIFACT",
			ConfigurationResolver:    cr,
			ModuleConfigurationKey:   "BP_MAVEN_BUILT_MODULE",
			InterestingFileDetector:  libbs.JARInterestingFileDetector{},
		}

		bomScanner := sbom.NewSyftCLISBOMScanner(context.Layers, effect.NewExecutor(), b.Logger)

		a, err := b.ApplicationFactory.NewApplication(
			md,
			args,
			art,
			c,
			command,
			result.BOM,
			context.Application.Path,
			bomScanner,
		)
		if err != nil {
			return libcnb.BuildResult{}, fmt.Errorf("unable to create application layer\n%w", err)
		}

		a.Logger = b.Logger
		result.Layers = append(result.Layers, a)
	} else {
		result.Layers = append(result.Layers, c)
	}

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
