// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package version

var version = "v0.0.0-dev"

// Version returs the codebase version. It's for detecting
// what code a binary was built from.
func Version() string {
	return version
}
