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
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/buildpacks/libcnb"
	"github.com/paketo-buildpacks/libpak"
	"github.com/paketo-buildpacks/libpak/bard"
)

const (
	PlanEntryMaven                 = "maven"
	PlanEntryJVMApplicationPackage = "jvm-application-package"
	PlanEntryJDK                   = "jdk"
	PlanEntrySyft                  = "syft"
)

type Detect struct{}

func (Detect) Detect(context libcnb.DetectContext) (libcnb.DetectResult, error) {
	l := bard.NewLogger(ioutil.Discard)
	cr, err := libpak.NewConfigurationResolver(context.Buildpack, &l)
	if err != nil {
		return libcnb.DetectResult{}, err
	}

	provide_maven := libcnb.DetectResult{
		Pass: true,
		Plans: []libcnb.BuildPlan{
			{
				Provides: []libcnb.BuildPlanProvide{
					{Name: PlanEntryMaven},
				},
				Requires: []libcnb.BuildPlanRequire{
					{Name: PlanEntryMaven},
				},
			},
		},
	}

	pomFile, _ := cr.Resolve("BP_MAVEN_POM_FILE")
	file := filepath.Join(context.Application.Path, pomFile)
	_, err = os.Stat(file)
	if os.IsNotExist(err) {
		return provide_maven, nil
	} else if err != nil {
		return libcnb.DetectResult{}, fmt.Errorf("unable to determine if %s exists\n%w", file, err)
	}

	result := libcnb.DetectResult{
		Pass: true,
		Plans: []libcnb.BuildPlan{
			{
				Provides: []libcnb.BuildPlanProvide{
					{Name: PlanEntryJVMApplicationPackage},
					{Name: PlanEntryMaven},
				},
				Requires: []libcnb.BuildPlanRequire{
					{Name: PlanEntrySyft},
					{Name: PlanEntryJDK},
					{Name: PlanEntryMaven},
				},
			},
		},
	}
	return result, nil
}
