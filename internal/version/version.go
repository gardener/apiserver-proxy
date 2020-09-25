// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package version

var (
	gitCommit = ""
	gitTag    = "v0.0.0-dev"
)

// Version returs the version in format `{gitTag}-{gitCommit}`
func Version() string {
	v := gitTag
	if gitCommit != "" {
		v += "-" + gitCommit
	}

	return v
}
