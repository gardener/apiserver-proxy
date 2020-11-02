// +build !linux

/*
SPDX-FileCopyrightText: 2017 The Kubernetes Authors.
SPDX-License-Identifier: Apache-2.0
*/

package iptables

import (
	"fmt"
	"os"
)

func grabIptablesLocks(lockfilePath string) (iptablesLocker, error) {
	return nil, fmt.Errorf("iptables unsupported on this platform")
}

func grabIptablesFileLock(f *os.File) error {
	return fmt.Errorf("iptables unsupported on this platform")
}
