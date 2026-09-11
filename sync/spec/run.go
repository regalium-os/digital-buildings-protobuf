// Copyright 2026 RegaliumOS™.
// SPDX-License-Identifier: Apache-2.0

package spec

import (
	"bytes"
	"fmt"
	"os/exec"
)

// run executes a git command in dir. Split from spec.go so that the pin logic
// stays free of process handling.
func run(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	var out, errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb
	if err := cmd.Run(); err != nil {
		if errb.Len() > 0 {
			return "", fmt.Errorf("%w: %s", err, bytes.TrimSpace(errb.Bytes()))
		}
		return "", err
	}
	return out.String(), nil
}
