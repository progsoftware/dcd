#!/bin/sh

echo "output to stdout 1"
echo "output to stderr 1" >&2
echo "output to stdout 2"
echo "output to stderr 2" >&2

exit 1
