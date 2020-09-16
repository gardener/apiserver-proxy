#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors
# SPDX-License-Identifier: Apache-2.0
echo STABLE_BUILD_GIT_COMMIT $(git rev-parse HEAD)
echo STABLE_BUILD_GIT_TAG $(cat VERSION)
