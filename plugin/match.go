// Copyright 2018 Drone.IO Inc
// Use of this software is governed by the Business Source License
// that can be found in the LICENSE file.

package plugin

import (
	"path"
	"strings"
)

func match(name string, patterns []string) bool {
	if len(patterns) == 0 {
		return true
	}
	name = strings.ToLower(name)
	for _, pattern := range patterns {
		pattern = strings.ToLower(pattern)
		match, _ := path.Match(pattern, name)
		if match {
			return true
		}
	}
	return false
}
