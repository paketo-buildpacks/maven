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
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/buildpacks/libcnb"
	. "github.com/onsi/gomega"
	"github.com/paketo-buildpacks/libbs"
	"github.com/paketo-buildpacks/maven/maven"
	"github.com/sclevine/spec"
)

func testBuild(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		ctx libcnb.BuildContext
	)

	it.Before(func() {
		var err error

		ctx.Application.Path, err = ioutil.TempDir("", "build-application")
		Expect(err).NotTo(HaveOccurred())

		ctx.Buildpack.Metadata = map[string]interface{}{
			"configurations": []map[string]interface{}{
				{"name": "BP_MAVEN_BUILD_ARGUMENTS", "default": "test-argument"},
			},
		}

		ctx.Layers.Path, err = ioutil.TempDir("", "build-layers")
		Expect(err).NotTo(HaveOccurred())
	})

	it.After(func() {
		Expect(os.RemoveAll(ctx.Application.Path)).To(Succeed())
		Expect(os.RemoveAll(ctx.Layers.Path)).To(Succeed())
	})

	it("does not contribute distribution if wrapper exists", func() {
		Expect(ioutil.WriteFile(filepath.Join(ctx.Application.Path, "mvnw"), []byte{}, 0644)).To(Succeed())
		ctx.StackID = "test-stack-id"

		result, err := maven.Build{}.Build(ctx)
		Expect(err).NotTo(HaveOccurred())

		Expect(result.Layers).To(HaveLen(2))
		Expect(result.Layers[0].Name()).To(Equal("cache"))
		Expect(result.Layers[1].Name()).To(Equal("application"))
		Expect(result.Layers[1].(libbs.Application).Command).To(Equal(filepath.Join(ctx.Application.Path, "mvnw")))
		Expect(result.Layers[1].(libbs.Application).Arguments).To(Equal([]string{"test-argument"}))
	})

	it("contributes distribution", func() {
		ctx.Buildpack.Metadata["dependencies"] = []map[string]interface{}{
			{
				"id":      "maven",
				"version": "1.1.1",
				"stacks":  []interface{}{"test-stack-id"},
			},
		}
		ctx.StackID = "test-stack-id"

		result, err := maven.Build{}.Build(ctx)
		Expect(err).NotTo(HaveOccurred())

		Expect(result.Layers).To(HaveLen(3))
		Expect(result.Layers[0].Name()).To(Equal("maven"))
		Expect(result.Layers[1].Name()).To(Equal("cache"))
		Expect(result.Layers[2].Name()).To(Equal("application"))
		Expect(result.Layers[2].(libbs.Application).Command).To(Equal(filepath.Join(ctx.Layers.Path, "maven", "bin", "mvn")))
		Expect(result.Layers[2].(libbs.Application).Arguments).To(Equal([]string{"test-argument"}))
	})

	context("BP_MAVEN_SETTINGS", func() {
		it.Before(func() {
			Expect(os.Setenv("BP_MAVEN_SETTINGS", "test-value")).To(Succeed())
		})

		it.After(func() {
			Expect(os.Unsetenv("BP_MAVEN_SETTINGS")).To(Succeed())
		})

		it("contributes settings.xml", func() {
			Expect(ioutil.WriteFile(filepath.Join(ctx.Application.Path, "mvnw"), []byte{}, 0644)).To(Succeed())
			ctx.StackID = "test-stack-id"

			result, err := maven.Build{}.Build(ctx)
			Expect(err).NotTo(HaveOccurred())

			Expect(result.Layers).To(HaveLen(3))
			Expect(result.Layers[1].Name()).To(Equal("settings"))
			Expect(result.Layers[2].(libbs.Application).Arguments).To(Equal([]string{
				fmt.Sprintf("--settings=%s", filepath.Join(ctx.Layers.Path, "settings", "settings.xml")),
				"test-argument",
			}))

		})
	})

}
