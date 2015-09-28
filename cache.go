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
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"
)

const cacheName = "docker-zypper.json"

// The representation of cached data for this application.
type cachedData struct {
	// The path to the original cache file.
	Path string `json:"-"`

	// Contains all the IDs that are known to be valid openSUSE/SLE images.
	Suse []string `json:"suse"`

	// Contains all the IDs that are known to be non-openSUSE/SLE images.
	Other []string `json:"other"`

	// Contains all the IDs of the images that have been either patched or
	// upgraded or patched using zypper-docker
	Outdated []string `json:"outdated"`

	// Whether this data comes from a valid file or not.
	Valid bool `json:"-"`
}

// Checks whether the given Id exists or not. It returns two booleans:
//  - Whether it exists or not.
//  - If it exists, whether it is a SUSE image or not.
func (cd *cachedData) idExists(id string) (bool, bool) {
	suse := arrayIncludeString(cd.Suse, id)
	other := arrayIncludeString(cd.Other, id)

	return suse || other, suse
}

// Returns whether the given ID matches an image that has been
// updated via zypper-docker patch|update
func (cd *cachedData) isImageOutdated(id string) bool {
	return arrayIncludeString(cd.Outdated, id)
}

// Returns whether the given ID matches an image that is based on SUSE.
func (cd *cachedData) isSUSE(id string) bool {
	if cd.Valid {
		if exists, suse := cd.idExists(id); exists {
			return suse
		}
	}

	suse := checkCommandInImage(id, "zypper")
	if cd.Valid {
		if suse {
			cd.Suse = append(cd.Suse, id)
		} else {
			cd.Other = append(cd.Other, id)
		}
	}
	return suse
}

// Writes all the cached data back to the cache file. This is needed because
// functions like `inSUSE` only write to memory. Therefore, once you're done
// with this instance, you should call this function to keep everything synced.
func (cd *cachedData) flush() {
	if !cd.Valid {
		// Silently fail, the user has probably already been notified about it.
		return
	}

	file, err := os.OpenFile(cd.Path, os.O_RDWR|os.O_TRUNC, 0666)
	if err != nil {
		log.Printf("Cannot write to the cache file: %v", err)
		return
	}

	enc := json.NewEncoder(file)
	_ = enc.Encode(cd)
	_ = file.Close()
}

// Empty the contents of the cache file.
func (cd *cachedData) reset() {
	cd.Suse, cd.Other = []string{}, []string{}
	cd.flush()
}

// Update the Cachefile after an update.
// The ID of the outdated image will be added to outdated Images and
// the ID of the new image will be added to the SUSE Images.
func (cd *cachedData) updateCacheAfterUpdate(outdatedImg, updatedImgId string) error {
	outdatedImgId, err := getImageId(outdatedImg)
	if err != nil {
		return err
	}
	if !arrayIncludeString(cd.Outdated, outdatedImgId) {
		cd.Outdated = append(cd.Outdated, outdatedImgId)
		cd.flush()
	}
	if !arrayIncludeString(cd.Suse, updatedImgId) {
		cd.Suse = append(cd.Suse, updatedImgId)
		cd.flush()
	}

	return nil
}

// Retrieves the path for the cache file. It checks the following directories
// in this specific order:
//  1. $HOME/.cache
//  2. /tmp
// It will try to open (or create if it doesn't exist) the cache file in each
// directory until it finds a directory that is accessible.
func cachePath() *os.File {
	candidates := []string{filepath.Join(os.Getenv("HOME"), ".cache"), "/tmp"}

	for _, dir := range candidates {
		dirs := strings.Split(dir, ":")
		for _, d := range dirs {
			name := filepath.Join(d, cacheName)
			file, err := os.OpenFile(name, os.O_RDWR|os.O_CREATE, 0666)
			if err == nil {
				return file
			}
		}
	}
	return nil
}

// Create a cache file or get the current one if it already exists. If that is
// not possible, then the returned struct will be marked as invalid (meaning
// that `isSUSE` will work without caching).
func getCacheFile() *cachedData {
	file := cachePath()
	if file == nil {
		log.Println("Could not find path for the cache!")
		return &cachedData{Valid: false}
	}

	cd := &cachedData{Valid: true, Path: file.Name()}
	dec := json.NewDecoder(file)
	err := dec.Decode(&cd)
	_ = file.Close()
	if err != nil {
		log.Printf("Decoding of cache file failed: %v\n", err)
		return &cachedData{Valid: true, Path: file.Name()}
	}
	return cd
}
