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
	xglob "github.com/bmatcuk/doublestar/v4"
	"path/filepath"
	"strings"
)

// CommonPath Remove all subdirectories that have a common parent directory in a set of directory arrays,
// leaving only their parent directories
// eg: paths: "/var/log/*.log", "/var/log/*.txt", "/data/**", "/usr/local/tomcat/access.log"
//     results: "/var/log", "/data", "/usr/local/tomcat"
func CommonPath(paths []string) []string {
	if len(paths) == 0 {
		return nil
	}

	var dirs []string
	for _, p := range paths {
		base, _ := xglob.SplitPattern(p)
		dirs = append(dirs, base)
	}

	return removeDuplicateDirs(dirs)
}

func removeDuplicateDirs(dirs []string) []string {
	m := make(map[string]bool)

	for _, dir := range dirs {
		add := true
		for k := range m {
			result := commonParents(dir, k)
			if result != "" {
				delete(m, k)
				m[result] = true
				add = false
				break
			}
		}

		if add {
			// has no common parents
			m[dir] = true
		}
	}

	var res []string
	for k := range m {
		res = append(res, k)
	}

	return res
}

func commonParents(path1, path2 string) string {
	// Split the paths into slices of directories.
	path1Dirs := splitPaths(path1)
	path2Dirs := splitPaths(path2)

	// Find the common parent by iterating through the slices
	// and comparing the corresponding directories.
	var commonParent string
	for i := 0; i < len(path1Dirs) && i < len(path2Dirs); i++ {
		if path1Dirs[i] != path2Dirs[i] {
			break
		}
		commonParent = filepath.Join(commonParent, path1Dirs[i])
	}

	if commonParent != "" && !strings.HasPrefix(commonParent, "/") {
		commonParent = "/" + commonParent
	}

	return commonParent
}

func splitPaths(path string) []string {
	if path == "" {
		return []string{}
	}
	return strings.Split(path, "/")
}
