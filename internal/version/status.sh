#!/usr/bin/env bash

echo STABLE_BUILD_GIT_COMMIT $(git rev-parse HEAD)
echo STABLE_BUILD_GIT_TAG $(cat VERSION)
