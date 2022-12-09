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
	"encoding/xml"
	"io/ioutil"
	"os"
)

type pomProject struct {
	XMLName    xml.Name      `xml:"project"`
	Properties pomProperties `xml:"properties"`
}

type pomProperties struct {
	JavaVersion string `xml:"java.version"`
}

func ParseJDKVersionFromPOM(filename string) (string, error) {
	pomFile, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer pomFile.Close()
	byteValue, err := ioutil.ReadAll(pomFile)

	project := pomProject{}
	xml.Unmarshal(byteValue, &project)
	if err != nil {
		return "", err
	}

	if project.Properties.JavaVersion != "" {
		return project.Properties.JavaVersion, nil
	}

	return "", nil
}
