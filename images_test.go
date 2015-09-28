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
	"bytes"
	"log"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/mssola/capture"
)

func TestMain(m *testing.M) {
	status := 0

	home, umask := os.Getenv("HOME"), syscall.Umask(0)
	abs, _ := filepath.Abs(".")
	test := filepath.Join(abs, "test")

	defer func() {
		_ = os.Setenv("HOME", home)
		syscall.Umask(umask)
		_ = os.Remove(filepath.Join(test, ".cache", cacheName))
		os.Exit(status)
	}()

	_ = os.Setenv("HOME", test)

	status = m.Run()
}

// IMAGES

func TestImagesCommand(t *testing.T) {
	cases := testCases{
		{"List fail", &mockClient{listFail: true}, 1, []string{}, false, "Cannot proceed safely: List Failed", ""},
		{"Empty list of images", &mockClient{listEmpty: true}, 0, []string{}, false, "", "REPOSITORY"},
	}
	cases.run(t, imagesCmd, "", "")
}

func TestImagesCommandList(t *testing.T) {
	dockerClient = &mockClient{waitSleep: 100 * time.Millisecond}
	setupTestExitStatus()

	buffer := bytes.NewBuffer([]byte{})
	log.SetOutput(buffer)

	res := capture.All(func() { imagesCmd(testContext([]string{}, false)) })

	lines := strings.Split(string(res.Stdout), "\n")
	if len(lines) != 7 {
		t.Fatal("Wrong number of lines")
	}
	if !strings.HasPrefix(lines[1], "REPOSITORY") {
		t.Fatal("Wrong contents")
	}
	str := "opensuse            latest              1                   Less than a second ago   254.5 MB"
	if lines[2] != str {
		t.Fatal("Wrong contents")
	}
	str = "opensuse            tag                 1                   Less than a second ago   254.5 MB"
	if lines[3] != str {
		t.Fatal("Wrong contents")
	}
	str = "opensuse            13.2                2                   Less than a second ago   254.5 MB"
	if lines[4] != str {
		t.Fatal("Wrong contents")
	}
	if exitInvocations != 1 && lastCode != 0 {
		t.Fatal("Wrong exit code")
	}
}

// Special tests for the IMAGES command.

func TestImagesListUsingCache(t *testing.T) {
	dockerClient = &mockClient{waitSleep: 100 * time.Millisecond}
	setupTestExitStatus()

	// Dump some dummy value.
	cd := getCacheFile()
	cd.Suse = []string{"1"}
	cd.Other = []string{"3"}
	cd.flush()

	buffer := bytes.NewBuffer([]byte{})
	log.SetOutput(buffer)

	res := capture.All(func() { imagesCmd(testContext([]string{}, false)) })

	lines := strings.Split(string(res.Stdout), "\n")
	if len(lines) != 7 {
		t.Fatal("Wrong number of lines")
	}
	if !strings.HasPrefix(lines[1], "REPOSITORY") {
		t.Fatal("Wrong contents")
	}
	str := "opensuse            latest              1                   Less than a second ago   254.5 MB"
	if lines[2] != str {
		t.Fatal("Wrong contents")
	}
	str = "opensuse            tag                 1                   Less than a second ago   254.5 MB"
	if lines[3] != str {
		t.Fatal("Wrong contents")
	}
	str = "opensuse            13.2                2                   Less than a second ago   254.5 MB"
	if lines[4] != str {
		t.Fatal("Wrong contents")
	}
	if exitInvocations != 1 && lastCode != 0 {
		t.Fatal("Wrong exit code")
	}
}

func TestImagesForce(t *testing.T) {
	dockerClient = &mockClient{waitSleep: 100 * time.Millisecond}
	setupTestExitStatus()

	// Dump some dummy value.
	cd := getCacheFile()
	cd.Suse = []string{"1234"}
	cd.flush()

	// Check that they are really written there.
	cd = getCacheFile()
	if len(cd.Suse) != 1 || cd.Suse[0] != "1234" {
		t.Fatal("Unexpected value")
	}

	// Luke, use the force!
	capture.All(func() { imagesCmd(testContext([]string{}, true)) })
	cd = getCacheFile()

	if !cd.Valid {
		t.Fatal("It should be valid")
	}
	for i, v := range []string{"1", "2", "4"} {
		if cd.Suse[i] != v {
			t.Fatal("Unexpected value")
		}
	}
	if len(cd.Other) != 1 || cd.Other[0] != "3" {
		t.Fatal("Unexpected value")
	}
	if exitInvocations != 1 && lastCode != 0 {
		t.Fatal("Wrong exit code")
	}
}

// Helper functions in the images.go file.

func TestCheckImageListFail(t *testing.T) {
	dockerClient = &mockClient{listFail: true}

	var err error

	capture.All(func() {
		_, err = checkImageExists("opensuse", "bar")
	})

	if err == nil {
		t.Fatal("Error did not occur")
	}
}

func TestCheckImageExistsEmptyList(t *testing.T) {
	var found bool
	var err error

	dockerClient = &mockClient{listEmpty: true}

	capture.All(func() {
		found, err = checkImageExists("suse/sles11sp3", "latest")
	})

	if err != nil {
		t.Fatal("Unexpected error")
	}
	if found == true {
		t.Fatal("The image should not have been found")
	}
}

func TestCheckImageExists(t *testing.T) {
	var found bool
	var err error

	dockerClient = &mockClient{waitSleep: 100 * time.Millisecond}

	expected := []string{"latest", "13.2"}
	for _, e := range expected {
		capture.All(func() {
			found, err = checkImageExists("opensuse", e)
		})

		if err != nil {
			t.Fatal("Unexpected error")
		}
		if found != true {
			t.Fatal("The image should have been found")
		}
	}

	not_expected := []string{"unexpected_tag"}
	for _, unexpected := range not_expected {
		capture.All(func() {
			found, err = checkImageExists("opensuse", unexpected)
		})

		if err != nil {
			t.Fatal("Unexpected error")
		}
		if found != false {
			t.Fatal("The image should not have been found")
		}
	}
}
