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

	. "github.com/onsi/gomega"
	"github.com/paketo-buildpacks/maven/v6/maven"
	"github.com/sclevine/spec"
)

func testParsePOM(t *testing.T, context spec.G, it spec.S) {
	var (
		path   string
		Expect = NewWithT(t).Expect
		err    error
	)

	it.Before(func() {
		path, err = os.MkdirTemp("", "pom")
		Expect(err).NotTo(HaveOccurred())
	})

	it.After(func() {
		Expect(os.RemoveAll(path)).To(Succeed())
	})

	context("parsing JDK version from POM", func() {
		it("fails if file does not exist", func() {
			_, err = maven.ParseJDKVersionFromPOM("doesNotExist")

			Expect(os.IsNotExist(err)).To(BeTrue())
		})

		it("returns empty string if no version found", func() {
			filename := filepath.Join(path, "pom.xml")
			os.WriteFile(filename, []byte(""), os.ModePerm)
			version, err := maven.ParseJDKVersionFromPOM(filename)

			Expect(err).NotTo(HaveOccurred())
			Expect(version).To(Equal(""))
		})

		it("returns java.version", func() {
			content :=
				`<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0">
	<properties>
		<java.version>17</java.version>
	</properties>
</project>
`
			filename := filepath.Join(path, "pom.xml")
			os.WriteFile(filename, []byte(content), os.ModePerm)
			version, err := maven.ParseJDKVersionFromPOM(filename)

			Expect(err).NotTo(HaveOccurred())
			Expect(version).To(Equal("17"))
		})

	})
}
