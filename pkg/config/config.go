/*
Copyright 2023 Loggie.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package config

type Config struct {
	Sidecar *Sidecar `yaml:"sidecar,omitempty" validate:"dive"`
}

type Sidecar struct {
	Enabled              bool     `yaml:"enabled,omitempty"`
	Image                string   `yaml:"image,omitempty" validate:"required"`
	IgnoreNamespaces     []string `yaml:"ignoreNamespaces,omitempty"`
	IgnoreContainerNames []string `yaml:"ignoreContainerNames,omitempty"`
	SystemConfig         string   `yaml:"systemConfig,omitempty" validate:"required"`
}
