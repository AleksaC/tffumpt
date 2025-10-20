// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package diff

import (
	"os"
	"os/exec"
)

// Taken from terraform fmt code. Calling a subprocess for diffing sucks, but given
// the state of diffing libraries in Go it's reasonable for now.
//
//nolint:errcheck,nakedret // Copied from terraform, more likely to be completely replaced than changed
func BytesDiff(b1, b2 []byte, path string) (data []byte, err error) {
	f1, err := os.CreateTemp("", "")
	if err != nil {
		return data, err
	}

	defer os.Remove(f1.Name())
	defer f1.Close()

	f2, err := os.CreateTemp("", "")
	if err != nil {
		return
	}
	defer os.Remove(f2.Name())
	defer f2.Close()

	f1.Write(b1)
	f2.Write(b2)

	args := []string{
		"--label=old/" + path,
		"--label=new/" + path,
		"-u", f1.Name(), f2.Name(),
	}

	data, err = exec.Command("diff", args...).CombinedOutput()
	if len(data) > 0 {
		// diff exits with a non-zero status when the files don't match.
		// Ignore that failure as long as we get output.
		err = nil
	}

	return
}
