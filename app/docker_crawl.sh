#!/usr/bin/env bash

docker run --rm --read-only --security-opt="no-new-privileges" --security-opt="seccomp=.\crawl.seccomp" bytemare/crawl https://bytema.re