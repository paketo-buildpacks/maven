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
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"

	"github.com/buildpacks/libcnb"
	"github.com/paketo-buildpacks/libbs"
	"github.com/paketo-buildpacks/libpak"
	"github.com/paketo-buildpacks/libpak/bard"
)

type Build struct {
	Logger             bard.Logger
	ApplicationFactory ApplicationFactory
}

type ApplicationFactory interface {
	NewApplication(additionalMetadata map[string]interface{}, arguments []string, artifactResolver libbs.ArtifactResolver,
		cache libbs.Cache, command string, plan *libcnb.BuildpackPlan, applicationPath string) (libbs.Application, error)
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

		d := NewDistribution(dep, dc, result.Plan)
		d.Logger = b.Logger
		result.Layers = append(result.Layers, d)

		command = filepath.Join(context.Layers.Path, d.Name(), "bin", "mvn")
	} else if err != nil {
		return libcnb.BuildResult{}, fmt.Errorf("unable to stat %s\n%w", command, err)
	} else {
		if err := os.Chmod(command, 0755); err != nil {
			return libcnb.BuildResult{}, fmt.Errorf("unable to chmod %s\n%w", command, err)
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
	md := map[string]interface{}{}
	br := libpak.BindingResolver{Bindings: context.Platform.Bindings}
	if binding, ok, err := br.Resolve("maven"); err != nil {
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
		result.Plan,
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
	path, ok := binding.SecretFilePath("settings.xml")
	if !ok {
		return args, nil
	}
	args = append([]string{fmt.Sprintf("--settings=%s", path)}, args...)
	hasher := sha256.New()
	settingsFile, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("unable to open settings.xml\n%w", err)
	}
	if _, err := io.Copy(hasher, settingsFile); err != nil {
		return nil, fmt.Errorf("error hashing settings.xml\n%w", err)
	}
	md["settings-sha256"] = hex.EncodeToString(hasher.Sum(nil))
	return args, nil
}
