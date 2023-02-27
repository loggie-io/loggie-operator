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

package files

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_removeSubDirs(t *testing.T) {
	type args struct {
		dirs []string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "common",
			args: args{
				dirs: []string{
					"/a/b/c", "/a/b/d", "/a/b", "/a/b/c/d", "/d", "/a/b/d",
				},
			},
			want: []string{
				"/a/b", "/d",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := removeDuplicateDirs(tt.args.dirs)
			assert.EqualValues(t, tt.want, got)
		})
	}
}

func TestCommonPath(t *testing.T) {
	type args struct {
		paths []string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "common",
			args: args{
				paths: []string{
					"/var/log/*.log", "/var/log/*.txt", "/data/**", "/usr/local/tomcat/access.log",
				},
			},
			want: []string{
				"/var/log", "/data", "/usr/local/tomcat",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CommonPath(tt.args.paths)
			assert.EqualValues(t, tt.want, got)
		})
	}
}
