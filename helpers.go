// Copyright (c) 2015 SUSE LLC. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"strings"

	"github.com/codegangsta/cli"
)

// It appends the set flags with the given command.
// `boolFlags` is a list of strings containing the names of the boolean
// command line options. These have to be handled in a slightly different
// way because zypper expects `--boolflag` instead of `--boolflag true`. Also
// boolean flags with a false value are ignored because zypper set all the
// undefined bool flags to false by default.
func cmdWithFlags(cmd string, ctx *cli.Context, boolFlags []string) string {
	arrayInclude := func(s string, arr []string) bool {
		for _, i := range arr {
			if i == s {
				return true
			}
		}
		return false
	}

	for _, name := range ctx.FlagNames() {
		if value := ctx.String(name); value != "" {
			var dash string
			if len(name) == 1 {
				dash = "-"
			} else {
				dash = "--"
			}

			if arrayInclude(name, boolFlags) {
				if ctx.Bool(name) {
					// ignore bool flags with value set to false
					cmd += fmt.Sprintf(" %v%s", dash, name)
				}
			} else {
				cmd += fmt.Sprintf(" %v%s %s", dash, name, value)
			}
		}
	}
	return cmd
}

// Given a Docker image name it returns the repository and the tag composing it
// Returns the repository and the tag strings.
// Examples:
//   * suse/sles11sp3:1.0.0 -> repo is suse/sles11sp3, tag is 1.0.0
//   * suse/sles11sp3 -> repo is suse/sles11sp3, tag is latest
func parseImageName(name string) (string, string) {
	var repo, tag string
	target := strings.SplitN(name, ":", 2)
	repo = target[0]
	if len(target) != 2 {
		tag = "latest"
	} else {
		tag = target[1]
	}

	return repo, tag
}

// Exists with error if the image identified by repo and tag already exists
// Returns an error when the image already exists or something went wrong.
func preventImageOverwrite(repo, tag string) error {
	imageExists, err := checkImageExists(repo, tag)
	if err != nil {
		return err
	}
	if imageExists {
		return fmt.Errorf("Cannot overwrite an existing image. Please use a different repository/tag.")
	}
	return nil
}
