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

package maven_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/buildpacks/libcnb"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"

	"github.com/paketo-buildpacks/maven/v6/maven"
)

func testDetect(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		ctx    libcnb.DetectContext
		detect maven.Detect
	)

	it.Before(func() {
		var err error

		ctx.Buildpack.Metadata = map[string]interface{}{
			"configurations": []map[string]interface{}{
				{
					"name":    "BP_MAVEN_POM_FILE",
					"default": "pom.xml",
					"build":   true,
					"detect":  true,
				},
			},
		}

		ctx.Application.Path, err = os.MkdirTemp("", "maven")
		Expect(err).NotTo(HaveOccurred())
	})

	it.After(func() {
		Expect(os.RemoveAll(ctx.Application.Path)).To(Succeed())
	})

	context("there is a META-INF/MANIFEST.MF", func() {
		it("fails", func() {
			Expect(os.MkdirAll(filepath.Join(ctx.Application.Path, "META-INF"), 0755)).Should(Succeed())
			Expect(os.WriteFile(filepath.Join(ctx.Application.Path, "META-INF", "MANIFEST.MF"), []byte{}, 0644))

			Expect(detect.Detect(ctx)).To(Equal(libcnb.DetectResult{}))
		})
	})

	context("there is no pom.xml", func() {
		it("only provides looking at default location", func() {
			Expect(detect.Detect(ctx)).To(Equal(libcnb.DetectResult{
				Pass: true,
				Plans: []libcnb.BuildPlan{
					{
						Provides: []libcnb.BuildPlanProvide{
							{Name: "maven"},
						},
						Requires: []libcnb.BuildPlanRequire{
							{Name: "jdk"},
						},
					},
					{
						Provides: []libcnb.BuildPlanProvide{
							{Name: "jvm-application-package"},
							{Name: "maven"},
						},
					},
				},
			}))
		})

		it("only provides when BP_MAVEN_POM_FILE is set to a place that does not exist", func() {
			t.Setenv("BP_MAVEN_POM_FILE", "pom-does-not-exist.xml")

			Expect(detect.Detect(ctx)).To(Equal(libcnb.DetectResult{
				Pass: true,
				Plans: []libcnb.BuildPlan{
					{
						Provides: []libcnb.BuildPlanProvide{
							{Name: "maven"},
						},
						Requires: []libcnb.BuildPlanRequire{
							{Name: "jdk"},
						},
					},
					{
						Provides: []libcnb.BuildPlanProvide{
							{Name: "jvm-application-package"},
							{Name: "maven"},
						},
					},
				},
			}))
		})
	})

	it("passes with pom.xml at the default location", func() {
		Expect(os.WriteFile(filepath.Join(ctx.Application.Path, "pom.xml"), []byte{}, 0644))

		Expect(detect.Detect(ctx)).To(Equal(libcnb.DetectResult{
			Pass: true,
			Plans: []libcnb.BuildPlan{
				{
					Provides: []libcnb.BuildPlanProvide{
						{Name: "maven"},
					},
					Requires: []libcnb.BuildPlanRequire{
						{Name: "jdk"},
					},
				},
				{
					Provides: []libcnb.BuildPlanProvide{
						{Name: "jvm-application-package"},
						{Name: "maven"},
					},
					Requires: []libcnb.BuildPlanRequire{
						{Name: "syft"},
						{Name: "jdk"},
						{Name: "maven"},
					},
				},
				{
					Provides: []libcnb.BuildPlanProvide{
						{Name: "jvm-application-package"},
					},
					Requires: []libcnb.BuildPlanRequire{
						{Name: "syft"},
						{Name: "jdk"},
						{Name: "maven"},
					},
				},
			},
		}))
	})

	it("passes with a pom.xml at a custom location", func() {
		Expect(os.WriteFile(filepath.Join(ctx.Application.Path, "pom2.xml"), []byte{}, 0644))

		t.Setenv("BP_MAVEN_POM_FILE", "pom2.xml")

		Expect(detect.Detect(ctx)).To(Equal(libcnb.DetectResult{
			Pass: true,
			Plans: []libcnb.BuildPlan{
				{
					Provides: []libcnb.BuildPlanProvide{
						{Name: "maven"},
					},
					Requires: []libcnb.BuildPlanRequire{
						{Name: "jdk"},
					},
				},
				{
					Provides: []libcnb.BuildPlanProvide{
						{Name: "jvm-application-package"},
						{Name: "maven"},
					},
					Requires: []libcnb.BuildPlanRequire{
						{Name: "syft"},
						{Name: "jdk"},
						{Name: "maven"},
					},
				},
				{
					Provides: []libcnb.BuildPlanProvide{
						{Name: "jvm-application-package"},
					},
					Requires: []libcnb.BuildPlanRequire{
						{Name: "syft"},
						{Name: "jdk"},
						{Name: "maven"},
					},
				},
			},
		}))
	})

	it("passes with pom.xml and yarn.lock", func() {
		Expect(os.WriteFile(filepath.Join(ctx.Application.Path, "pom.xml"), []byte{}, 0644))
		Expect(os.WriteFile(filepath.Join(ctx.Application.Path, "yarn.lock"), []byte{}, 0644))
		os.Setenv("BP_JAVA_INSTALL_NODE",  "true")

		Expect(detect.Detect(ctx)).To(Equal(libcnb.DetectResult{
			Pass: true,
			Plans: []libcnb.BuildPlan{
				{
					Provides: []libcnb.BuildPlanProvide{
						{Name: "maven"},
					},
					Requires: []libcnb.BuildPlanRequire{
						{Name: "jdk"},
					},
				},
				{
					Provides: []libcnb.BuildPlanProvide{
						{Name: "jvm-application-package"},
						{Name: "maven"},
					},
					Requires: []libcnb.BuildPlanRequire{
						{Name: "syft"},
						{Name: "jdk"},
						{Name: "maven"},
						{Name: "yarn", Metadata: map[string]interface{}{"build": true}},
						{Name: "node", Metadata: map[string]interface{}{"build": true}},
					},
				},
				{
					Provides: []libcnb.BuildPlanProvide{
						{Name: "jvm-application-package"},
					},
					Requires: []libcnb.BuildPlanRequire{
						{Name: "syft"},
						{Name: "jdk"},
						{Name: "maven"},
						{Name: "yarn", Metadata: map[string]interface{}{"build": true}},
						{Name: "node", Metadata: map[string]interface{}{"build": true}},
					},
				},
			},
		}))
	})

	it("passes with pom.xml and package.json", func() {
		Expect(os.WriteFile(filepath.Join(ctx.Application.Path, "pom.xml"), []byte{}, 0644))
		Expect(os.WriteFile(filepath.Join(ctx.Application.Path, "package.json"), []byte{}, 0644))
		os.Setenv("BP_JAVA_INSTALL_NODE",  "true")

		Expect(detect.Detect(ctx)).To(Equal(libcnb.DetectResult{
			Pass: true,
			Plans: []libcnb.BuildPlan{
				{
					Provides: []libcnb.BuildPlanProvide{
						{Name: "maven"},
					},
					Requires: []libcnb.BuildPlanRequire{
						{Name: "jdk"},
					},
				},
				{
					Provides: []libcnb.BuildPlanProvide{
						{Name: "jvm-application-package"},
						{Name: "maven"},
					},
					Requires: []libcnb.BuildPlanRequire{
						{Name: "syft"},
						{Name: "jdk"},
						{Name: "maven"},
						{Name: "node", Metadata: map[string]interface{}{"build": true}},
					},
				},
				{
					Provides: []libcnb.BuildPlanProvide{
						{Name: "jvm-application-package"},
					},
					Requires: []libcnb.BuildPlanRequire{
						{Name: "syft"},
						{Name: "jdk"},
						{Name: "maven"},
						{Name: "node", Metadata: map[string]interface{}{"build": true}},
					},
				},
			},
		}))
	})

	it("passes without duplication with both yarn.lock & package.json", func() {
		Expect(os.WriteFile(filepath.Join(ctx.Application.Path, "pom.xml"), []byte{}, 0644))
		Expect(os.WriteFile(filepath.Join(ctx.Application.Path, "package.json"), []byte{}, 0644))
		Expect(os.WriteFile(filepath.Join(ctx.Application.Path, "yarn.lock"), []byte{}, 0644))
		os.Setenv("BP_JAVA_INSTALL_NODE",  "true")

		Expect(detect.Detect(ctx)).To(Equal(libcnb.DetectResult{
			Pass: true,
			Plans: []libcnb.BuildPlan{
				{
					Provides: []libcnb.BuildPlanProvide{
						{Name: "maven"},
					},
					Requires: []libcnb.BuildPlanRequire{
						{Name: "jdk"},
					},
				},
				{
					Provides: []libcnb.BuildPlanProvide{
						{Name: "jvm-application-package"},
						{Name: "maven"},
					},
					Requires: []libcnb.BuildPlanRequire{
						{Name: "syft"},
						{Name: "jdk"},
						{Name: "maven"},
						{Name: "yarn", Metadata: map[string]interface{}{"build": true}},
						{Name: "node", Metadata: map[string]interface{}{"build": true}},
					},
				},
				{
					Provides: []libcnb.BuildPlanProvide{
						{Name: "jvm-application-package"},
					},
					Requires: []libcnb.BuildPlanRequire{
						{Name: "syft"},
						{Name: "jdk"},
						{Name: "maven"},
						{Name: "yarn", Metadata: map[string]interface{}{"build": true}},
						{Name: "node", Metadata: map[string]interface{}{"build": true}},
					},
				},
			},
		}))
	})

	it("passes with custom path set via BP_NODE_PROJECT_PATH", func() {
		os.Setenv("BP_NODE_PROJECT_PATH",  "frontend")
		os.Setenv("BP_JAVA_INSTALL_NODE",  "true")
		os.Mkdir(filepath.Join(ctx.Application.Path, "frontend"), 0755)
		Expect(os.WriteFile(filepath.Join(ctx.Application.Path, "pom.xml"), []byte{}, 0644))
		Expect(os.WriteFile(filepath.Join(ctx.Application.Path, "frontend/yarn.lock"), []byte{}, 0644))

		Expect(detect.Detect(ctx)).To(Equal(libcnb.DetectResult{
			Pass: true,
			Plans: []libcnb.BuildPlan{
				{
					Provides: []libcnb.BuildPlanProvide{
						{Name: "maven"},
					},
					Requires: []libcnb.BuildPlanRequire{
						{Name: "jdk"},
					},
				},
				{
					Provides: []libcnb.BuildPlanProvide{
						{Name: "jvm-application-package"},
						{Name: "maven"},
					},
					Requires: []libcnb.BuildPlanRequire{
						{Name: "syft"},
						{Name: "jdk"},
						{Name: "maven"},
						{Name: "yarn", Metadata: map[string]interface{}{"build": true}},
						{Name: "node", Metadata: map[string]interface{}{"build": true}},
					},
				},
				{
					Provides: []libcnb.BuildPlanProvide{
						{Name: "jvm-application-package"},
					},
					Requires: []libcnb.BuildPlanRequire{
						{Name: "syft"},
						{Name: "jdk"},
						{Name: "maven"},
						{Name: "yarn", Metadata: map[string]interface{}{"build": true}},
						{Name: "node", Metadata: map[string]interface{}{"build": true}},
					},
				},
			},
		}))
	})

	it("does not detect false positive without env-var", func() {
		Expect(os.WriteFile(filepath.Join(ctx.Application.Path, "pom.xml"), []byte{}, 0644))
		Expect(os.WriteFile(filepath.Join(ctx.Application.Path, "yarn.lock"), []byte{}, 0644))

		Expect(detect.Detect(ctx)).To(Equal(libcnb.DetectResult{
			Pass: true,
			Plans: []libcnb.BuildPlan{
				{
					Provides: []libcnb.BuildPlanProvide{
						{Name: "maven"},
					},
					Requires: []libcnb.BuildPlanRequire{
						{Name: "jdk"},
					},
				},
				{
					Provides: []libcnb.BuildPlanProvide{
						{Name: "jvm-application-package"},
						{Name: "maven"},
					},
					Requires: []libcnb.BuildPlanRequire{
						{Name: "syft"},
						{Name: "jdk"},
						{Name: "maven"},
					},
				},
				{
					Provides: []libcnb.BuildPlanProvide{
						{Name: "jvm-application-package"},
					},
					Requires: []libcnb.BuildPlanRequire{
						{Name: "syft"},
						{Name: "jdk"},
						{Name: "maven"},
					},
				},
			},
		}))
	})
}
