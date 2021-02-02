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
	"github.com/paketo-buildpacks/libpak"
	"github.com/sclevine/spec"

	"github.com/paketo-buildpacks/maven/maven"
)

func testBuild(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		ctx        libcnb.BuildContext
		mavenBuild maven.Build
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
		mavenBuild = maven.Build{ApplicationFactory: &FakeApplicationFactory{}}
	})

	it.After(func() {
		Expect(os.RemoveAll(ctx.Application.Path)).To(Succeed())
		Expect(os.RemoveAll(ctx.Layers.Path)).To(Succeed())
	})

	it("does not contribute distribution if wrapper exists", func() {
		Expect(ioutil.WriteFile(filepath.Join(ctx.Application.Path, "mvnw"), []byte{}, 0644)).To(Succeed())
		ctx.StackID = "test-stack-id"

		result, err := mavenBuild.Build(ctx)
		Expect(err).NotTo(HaveOccurred())

		fi, err := os.Stat(filepath.Join(ctx.Application.Path, "mvnw"))
		Expect(err).NotTo(HaveOccurred())
		Expect(fi.Mode()).To(BeEquivalentTo(0755))

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

		result, err := mavenBuild.Build(ctx)
		Expect(err).NotTo(HaveOccurred())

		Expect(result.Layers).To(HaveLen(3))
		Expect(result.Layers[0].Name()).To(Equal("maven"))
		Expect(result.Layers[1].Name()).To(Equal("cache"))
		Expect(result.Layers[2].Name()).To(Equal("application"))
		Expect(result.Layers[2].(libbs.Application).Command).To(Equal(filepath.Join(ctx.Layers.Path, "maven", "bin", "mvn")))
		Expect(result.Layers[2].(libbs.Application).Arguments).To(Equal([]string{"test-argument"}))

		Expect(result.BOM.Entries).To(HaveLen(1))
		Expect(result.BOM.Entries[0].Name).To(Equal("maven"))
		Expect(result.BOM.Entries[0].Build).To(BeTrue())
		Expect(result.BOM.Entries[0].Launch).To(BeFalse())
	})

	context("maven settings bindings exists", func() {
		var result libcnb.BuildResult

		it.Before(func() {
			var err error
			ctx.StackID = "test-stack-id"
			ctx.Platform.Path, err = ioutil.TempDir("", "maven-test-platform")
			Expect(ioutil.WriteFile(filepath.Join(ctx.Application.Path, "mvnw"), []byte{}, 0644)).To(Succeed())
			ctx.Platform.Bindings = libcnb.Bindings{
				{
					Name:   "some-maven",
					Type:   "maven",
					Secret: map[string]string{"settings.xml": "maven-settings-content"},
					Path:   filepath.Join(ctx.Platform.Path, "bindings", "some-maven"),
				},
			}
			mavenSettingsPath, ok := ctx.Platform.Bindings[0].SecretFilePath("settings.xml")
			Expect(os.MkdirAll(filepath.Dir(mavenSettingsPath), 0777)).To(Succeed())
			Expect(ok).To(BeTrue())
			Expect(ioutil.WriteFile(
				mavenSettingsPath,
				[]byte("maven-settings-content"),
				0644,
			)).To(Succeed())

			result, err = mavenBuild.Build(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Layers).To(HaveLen(2))
		})

		it.After(func() {
			Expect(os.RemoveAll(ctx.Platform.Path)).To(Succeed())
		})

		it("provides --settings argument to maven", func() {
			Expect(result.Layers[1].(libbs.Application).Arguments).To(Equal([]string{
				fmt.Sprintf("--settings=%s", filepath.Join(ctx.Platform.Path, "bindings", "some-maven", "settings.xml")),
				"test-argument",
			}))
		})

		it("adds the hash of settings.xml to the layer metadata", func() {
			md := result.Layers[1].(libbs.Application).LayerContributor.ExpectedMetadata
			mdMap, ok := md.(map[string]interface{})
			Expect(ok).To(BeTrue())
			// expected: sha256 of the string "maven-settings-content"
			expected := "cc784f356a8efb8e138b99aabe8b1c813a3e921b059c48a0b39b2497a2c478c5"
			Expect(mdMap["settings-sha256"]).To(Equal(expected))
		})
	})
}

type FakeApplicationFactory struct{}

func (f *FakeApplicationFactory) NewApplication(
	additionalMetdata map[string]interface{},
	argugments []string,
	_ libbs.ArtifactResolver,
	_ libbs.Cache,
	command string,
	_ *libcnb.BOM,
	_ string,
) (libbs.Application, error) {
	contributor := libpak.NewLayerContributor(
		"Compiled Application",
		additionalMetdata,
		libcnb.LayerTypes{Cache: true},
	)
	return libbs.Application{
		LayerContributor: contributor,
		Arguments:        argugments,
		Command:          command,
	}, nil
}
