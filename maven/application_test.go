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
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/buildpacks/libcnb"
	. "github.com/onsi/gomega"
	"github.com/paketo-buildpacks/libjvm"
	"github.com/paketo-buildpacks/libpak/bard"
	"github.com/paketo-buildpacks/libpak/effect"
	"github.com/paketo-buildpacks/libpak/effect/mocks"
	"github.com/paketo-buildpacks/maven/maven"
	"github.com/sclevine/spec"
	"github.com/stretchr/testify/mock"
)

func testApplication(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		cachePath   string
		ctx         libcnb.BuildContext
		application maven.Application
		executor    *mocks.Executor
		plan        *libcnb.BuildpackPlan
	)

	it.Before(func() {
		var err error

		ctx.Application.Path, err = ioutil.TempDir("", "application-application")
		Expect(err).NotTo(HaveOccurred())

		ctx.Layers.Path, err = ioutil.TempDir("", "application-layers")
		Expect(err).NotTo(HaveOccurred())

		cachePath, err = ioutil.TempDir("", "application-cache")
		Expect(err).NotTo(HaveOccurred())

		plan = &libcnb.BuildpackPlan{}

		application, err = maven.NewApplication(ctx.Application.Path, cachePath, "test-command", plan)
		Expect(err).NotTo(HaveOccurred())

		executor = &mocks.Executor{}
		application.Executor = executor
	})

	it.After(func() {
		Expect(os.RemoveAll(ctx.Application.Path)).To(Succeed())
		Expect(os.RemoveAll(ctx.Layers.Path)).To(Succeed())
		Expect(os.RemoveAll(cachePath)).To(Succeed())
	})

	it("contributes layer", func() {
		in, err := os.Open(filepath.Join("testdata", "stub-application.jar"))
		Expect(err).NotTo(HaveOccurred())
		Expect(os.MkdirAll(filepath.Join(ctx.Application.Path, "target"), 0755)).To(Succeed())
		out, err := os.OpenFile(filepath.Join(ctx.Application.Path, "target", "stub-application.jar"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		Expect(err).NotTo(HaveOccurred())
		_, err = io.Copy(out, in)
		Expect(err).NotTo(HaveOccurred())
		Expect(in.Close()).To(Succeed())
		Expect(out.Close()).To(Succeed())
		Expect(ioutil.WriteFile(filepath.Join(cachePath, "test-file-1.1.1.jar"), []byte{}, 0644)).To(Succeed())

		application.Logger = bard.NewLogger(ioutil.Discard)
		executor.On("Execute", mock.Anything).Return(nil)

		layer, err := ctx.Layers.Layer("test-layer")
		Expect(err).NotTo(HaveOccurred())

		layer, err = application.Contribute(layer)
		Expect(err).NotTo(HaveOccurred())

		Expect(layer.Cache).To(BeTrue())

		e := executor.Calls[0].Arguments[0].(effect.Execution)
		Expect(e.Command).To(Equal("test-command"))
		Expect(e.Args).To(Equal([]string{"-Dmaven.test.skip=true", "package"}))
		Expect(e.Dir).To(Equal(ctx.Application.Path))
		Expect(e.Stdout).NotTo(BeNil())
		Expect(e.Stderr).NotTo(BeNil())

		Expect(filepath.Join(layer.Path, "application.zip")).To(BeARegularFile())
		Expect(filepath.Join(ctx.Application.Path, "stub-application.jar")).NotTo(BeAnExistingFile())
		Expect(filepath.Join(ctx.Application.Path, "fixture-marker")).To(BeARegularFile())

		Expect(plan).To(Equal(&libcnb.BuildpackPlan{
			Entries: []libcnb.BuildpackPlanEntry{
				{
					Name: "maven",
					Metadata: map[string]interface{}{
						"dependencies": []libjvm.MavenJAR{
							{
								Name:    "test-file",
								Version: "1.1.1",
								SHA256:  "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
							},
						},
					},
				},
			},
		}))

	})

	context("ResolveArguments", func() {
		it("uses default arguments", func() {
			Expect(application.ResolveArguments()).To(Equal([]string{"-Dmaven.test.skip=true", "package"}))
		})

		context("$BP_MAVEN_BUILD_ARGUMENTS", func() {

			it.Before(func() {
				Expect(os.Setenv("BP_MAVEN_BUILD_ARGUMENTS", "test configured arguments")).To(Succeed())
			})

			it.After(func() {
				Expect(os.Unsetenv("BP_MAVEN_BUILD_ARGUMENTS")).To(Succeed())
			})

			it("parses value from $BP_MAVEN_BUILD_ARGUMENTS", func() {
				Expect(application.ResolveArguments()).To(Equal([]string{"test", "configured", "arguments"}))
			})
		})
	})

	context("ResolveArtifact", func() {
		it("fails with no files", func() {
			_, err := application.ResolveArtifact()
			Expect(err).To(MatchError("unable to find built artifact (executable JAR or WAR) in target/*.[jw]ar, candidates: []"))
		})

		it("fails with multiple candidates", func() {
			Expect(os.MkdirAll(filepath.Join(ctx.Application.Path, "target"), 0755)).To(Succeed())

			for _, f := range []string{"stub-application.jar", "stub-application.war", "stub-executable.jar"} {
				in, err := os.Open(filepath.Join("testdata", f))
				Expect(err).NotTo(HaveOccurred())

				out, err := os.OpenFile(filepath.Join(ctx.Application.Path, "target", f), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
				Expect(err).NotTo(HaveOccurred())

				_, err = io.Copy(out, in)
				Expect(err).NotTo(HaveOccurred())

				Expect(in.Close()).To(Succeed())
				Expect(out.Close()).To(Succeed())
			}

			_, err := application.ResolveArtifact()
			Expect(err).To(MatchError(
				fmt.Sprintf("unable to find built artifact (executable JAR or WAR) in target/*.[jw]ar, candidates: [%s %s %s]",
					filepath.Join(ctx.Application.Path, "target", "stub-application.jar"),
					filepath.Join(ctx.Application.Path, "target", "stub-application.war"),
					filepath.Join(ctx.Application.Path, "target", "stub-executable.jar"))))

		})

		it("passes with a single candidate", func() {
			Expect(os.MkdirAll(filepath.Join(ctx.Application.Path, "target"), 0755)).To(Succeed())

			in, err := os.Open(filepath.Join("testdata", "stub-application.jar"))
			Expect(err).NotTo(HaveOccurred())

			out, err := os.OpenFile(filepath.Join(ctx.Application.Path, "target", "stub-application.jar"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
			Expect(err).NotTo(HaveOccurred())

			_, err = io.Copy(out, in)
			Expect(err).NotTo(HaveOccurred())

			Expect(in.Close()).To(Succeed())
			Expect(out.Close()).To(Succeed())

			Expect(application.ResolveArtifact()).To(Equal(filepath.Join(ctx.Application.Path, "target", "stub-application.jar")))
		})

		it("passes with a single executable JAR", func() {
			Expect(os.MkdirAll(filepath.Join(ctx.Application.Path, "target"), 0755)).To(Succeed())

			for _, f := range []string{"stub-application.jar", "stub-executable.jar"} {
				in, err := os.Open(filepath.Join("testdata", f))
				Expect(err).NotTo(HaveOccurred())

				out, err := os.OpenFile(filepath.Join(ctx.Application.Path, "target", f), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
				Expect(err).NotTo(HaveOccurred())

				_, err = io.Copy(out, in)
				Expect(err).NotTo(HaveOccurred())

				Expect(in.Close()).To(Succeed())
				Expect(out.Close()).To(Succeed())
			}

			Expect(application.ResolveArtifact()).To(Equal(filepath.Join(ctx.Application.Path, "target", "stub-executable.jar")))
		})

		it("passes with a single WAR", func() {
			Expect(os.MkdirAll(filepath.Join(ctx.Application.Path, "target"), 0755)).To(Succeed())

			for _, f := range []string{"stub-application.jar", "stub-application.war"} {
				in, err := os.Open(filepath.Join("testdata", f))
				Expect(err).NotTo(HaveOccurred())

				out, err := os.OpenFile(filepath.Join(ctx.Application.Path, "target", f), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
				Expect(err).NotTo(HaveOccurred())

				_, err = io.Copy(out, in)
				Expect(err).NotTo(HaveOccurred())

				Expect(in.Close()).To(Succeed())
				Expect(out.Close()).To(Succeed())
			}

			Expect(application.ResolveArtifact()).To(Equal(filepath.Join(ctx.Application.Path, "target", "stub-application.war")))
		})

		context("$BP_MAVEN_BUILT_MODULE", func() {

			it.Before(func() {
				Expect(os.Setenv("BP_MAVEN_BUILT_MODULE", "test-directory")).To(Succeed())
			})

			it.After(func() {
				Expect(os.Unsetenv("BP_MAVEN_BUILT_MODULE")).To(Succeed())
			})

			it("passes with $BP_MAVEN_BUILT_MODULE", func() {
				Expect(os.MkdirAll(filepath.Join(ctx.Application.Path, "test-directory", "target"), 0755)).To(Succeed())

				in, err := os.Open(filepath.Join("testdata", "stub-application.jar"))
				Expect(err).NotTo(HaveOccurred())

				out, err := os.OpenFile(filepath.Join(ctx.Application.Path, "test-directory", "target", "stub-application.jar"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
				Expect(err).NotTo(HaveOccurred())

				_, err = io.Copy(out, in)
				Expect(err).NotTo(HaveOccurred())

				Expect(in.Close()).To(Succeed())
				Expect(out.Close()).To(Succeed())

				Expect(application.ResolveArtifact()).To(Equal(filepath.Join(ctx.Application.Path, "test-directory", "target", "stub-application.jar")))
			})

		})

		context("$BP_MAVEN_BUILT_ARTIFACT", func() {
			it.Before(func() {
				Expect(os.Setenv("BP_MAVEN_BUILT_ARTIFACT", "test-directory/stub-application.jar")).To(Succeed())
			})

			it.After(func() {
				Expect(os.Unsetenv("BP_MAVEN_BUILT_ARTIFACT")).To(Succeed())
			})

			it("passes with BP_MAVEN_BUILT_ARTIFACT", func() {
				Expect(os.MkdirAll(filepath.Join(ctx.Application.Path, "test-directory"), 0755)).To(Succeed())

				in, err := os.Open(filepath.Join("testdata", "stub-application.jar"))
				Expect(err).NotTo(HaveOccurred())

				out, err := os.OpenFile(filepath.Join(ctx.Application.Path, "test-directory", "stub-application.jar"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
				Expect(err).NotTo(HaveOccurred())

				_, err = io.Copy(out, in)
				Expect(err).NotTo(HaveOccurred())

				Expect(in.Close()).To(Succeed())
				Expect(out.Close()).To(Succeed())

				Expect(application.ResolveArtifact()).To(Equal(filepath.Join(ctx.Application.Path, "test-directory", "stub-application.jar")))
			})

		})
	})
}
