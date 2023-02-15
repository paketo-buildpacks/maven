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
	"fmt"
	"github.com/buildpacks/libcnb"
	"github.com/paketo-buildpacks/libpak"
	"github.com/paketo-buildpacks/libpak/bard"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const (
	PlanEntryMaven                 = "maven"
	PlanEntryJVMApplicationPackage = "jvm-application-package"
	PlanEntryJDK                   = "jdk"
	PlanEntrySyft                  = "syft"
	PlanEntryYarn				   = "yarn"
	PlanEntryNode				   = "node"
)

type Detect struct{}

func (Detect) Detect(context libcnb.DetectContext) (libcnb.DetectResult, error) {
	// if MANIFEST.MF exists, we have a WAR/JAR so we don't build
	_, err := os.Stat(filepath.Join(context.Application.Path, "META-INF", "MANIFEST.MF"))
	if err == nil {
		return libcnb.DetectResult{}, nil
	}

	l := bard.NewLogger(io.Discard)
	cr, err := libpak.NewConfigurationResolver(context.Buildpack, &l)
	if err != nil {
		return libcnb.DetectResult{}, err
	}

	result := libcnb.DetectResult{
		Pass: true,
		Plans: []libcnb.BuildPlan{
			// just offer to provide Maven
			{
				Provides: []libcnb.BuildPlanProvide{
					{Name: PlanEntryMaven},
				},
				Requires: []libcnb.BuildPlanRequire{
					{Name: PlanEntryJDK},
				},
			},
			// offer to install & build with Maven
			{
				Provides: []libcnb.BuildPlanProvide{
					{Name: PlanEntryJVMApplicationPackage},
					{Name: PlanEntryMaven},
				},
			},
		},
	}

	pomFile, _ := cr.Resolve("BP_MAVEN_POM_FILE")
	var performBuild bool
	file := filepath.Join(context.Application.Path, pomFile)
	if _, err = os.Stat(file); err != nil && !os.IsNotExist(err) {
		return libcnb.DetectResult{}, fmt.Errorf("unable to determine if %s exists\n%w", file, err)
	} else if err == nil {
		performBuild = true

		// buildplan entry to support build-only
		result.Plans = append(result.Plans, libcnb.BuildPlan{
			Provides: []libcnb.BuildPlanProvide{
				{Name: PlanEntryJVMApplicationPackage},
			},
		})

		// add requires for install & build
		for i := 1; i < len(result.Plans); i++ {
			result.Plans[i].Requires = []libcnb.BuildPlanRequire{
				{Name: PlanEntrySyft},
				{Name: PlanEntryJDK},
				{Name: PlanEntryMaven},
			}
		}
	}
	// Only require Node if we will perform the build
	if performBuild {
		if cr.ResolveBool("BP_JAVA_INSTALL_NODE") {
			var fileFound bool
			files := []string{filepath.Join(context.Application.Path, "yarn.lock"), filepath.Join(context.Application.Path,"package.json")}
			if customNodePath, _ := cr.Resolve("BP_NODE_PROJECT_PATH"); customNodePath != "" {
				files = []string{filepath.Join(context.Application.Path, customNodePath, "yarn.lock"), filepath.Join(context.Application.Path, customNodePath, "package.json")}
			}
			if err := findFile(files, func (file string) bool{
				if strings.Contains(file,"yarn.lock") {
					for i := 1; i < len(result.Plans); i++ {
						result.Plans[i].Requires = []libcnb.BuildPlanRequire{
							{Name: PlanEntrySyft},
							{Name: PlanEntryJDK},
							{Name: PlanEntryMaven},
							{Name: PlanEntryYarn, Metadata: map[string]interface{}{"build": true}},
							{Name: PlanEntryNode, Metadata: map[string]interface{}{"build": true}},
						}
					}
					fileFound = true
					return true
				} else if strings.Contains(file,"package.json") {
					for i := 1; i < len(result.Plans); i++ {
						result.Plans[i].Requires = []libcnb.BuildPlanRequire{
							{Name: PlanEntrySyft},
							{Name: PlanEntryJDK},
							{Name: PlanEntryMaven},
							{Name: PlanEntryNode, Metadata: map[string]interface{}{"build": true}},
						}
					}
					fileFound = true
				}
				return false
			}); err != nil{
				return libcnb.DetectResult{}, err
			}
			if !fileFound{
				l.Infof("unable to find a yarn.lock or package.json file, you may need to set BP_NODE_PROJECT_PATH")
			}
		}
		return result, nil
	}

	return result, nil
}

func findFile (files []string, runWhenFound func(fileFound string) bool) error {
	for _, file := range files {
		_, err := os.Stat(file)
		if os.IsNotExist(err) {
			continue
		} else if err != nil {
			return fmt.Errorf("unable to determine if file %s exists \n%w", file, err)
		}
		if runWhenFound(file) {
			break
		}
	}
	return nil
}