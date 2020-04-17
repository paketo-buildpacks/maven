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
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/buildpacks/libcnb"
	. "github.com/onsi/gomega"
	"github.com/paketo-buildpacks/maven/maven"
	"github.com/sclevine/spec"
)

func testCache(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		ctx  libcnb.BuildContext
		path string
	)

	it.Before(func() {
		var err error

		path, err = ioutil.TempDir("", "cache")
		Expect(err).NotTo(HaveOccurred())

		ctx.Layers.Path, err = ioutil.TempDir("", "cache-layers")
		Expect(err).NotTo(HaveOccurred())
	})

	it.After(func() {
		Expect(os.RemoveAll(path)).To(Succeed())
		Expect(os.RemoveAll(ctx.Layers.Path)).To(Succeed())
	})

	it("symlinks to destination if it does not exist", func() {
		file := filepath.Join(path, "test")

		layer, err := ctx.Layers.Layer("test-layer")
		Expect(err).NotTo(HaveOccurred())

		layer, err = maven.Cache{Path: file}.Contribute(layer)
		Expect(err).NotTo(HaveOccurred())

		Expect(layer.Cache).To(BeTrue())

		fi, err := os.Lstat(file)
		Expect(err).NotTo(HaveOccurred())
		Expect(fi.Mode() & os.ModeSymlink).To(Equal(os.ModeSymlink))

		Expect(os.Readlink(file)).To(Equal(layer.Path))
	})
}
