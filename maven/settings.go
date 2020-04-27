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
	"path/filepath"

	"github.com/buildpacks/libcnb"
	"github.com/paketo-buildpacks/libpak"
	"github.com/paketo-buildpacks/libpak/bard"
)

type Settings struct {
	Content          string
	LayerContributor libpak.LayerContributor
	Logger           bard.Logger
	Path             string
}

func NewSettings(content string, layersPath string) Settings {
	return Settings{
		Content:          content,
		LayerContributor: libpak.NewLayerContributor("Maven Settings", map[string]interface{}{"content": content}),
		Path:             filepath.Join(layersPath, "settings", "settings.xml"),
	}
}

func (s Settings) Contribute(layer libcnb.Layer) (libcnb.Layer, error) {
	s.LayerContributor.Logger = s.Logger

	return s.LayerContributor.Contribute(layer, func() (libcnb.Layer, error) {
		s.Logger.Bodyf("Writing %s", s.Path)

		if err := ioutil.WriteFile(s.Path, []byte(s.Content), 0644); err != nil {
			return libcnb.Layer{}, fmt.Errorf("unable to write %s\n%w", s.Path, err)
		}

		layer.Cache = true
		return layer, nil
	})
}

func (Settings) Name() string {
	return "settings"
}
