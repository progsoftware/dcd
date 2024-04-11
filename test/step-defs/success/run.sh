#!/bin/sh

echo "component: $COMPONENT"
echo "git sha: $GIT_SHA"
echo "build id: $BUILD_ID"
echo "global env var: $GLOBAL_ENV_VAR"
echo "output to stdout 1"
echo "output to stderr 1" >&2
echo "output to stdout 2"
echo "output to stderr 2" >&2
