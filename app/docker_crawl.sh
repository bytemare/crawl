#!/usr/bin/env bash

docker run --read-only --security-opt="no-new-privileges" --security-opt="seccomp=.\crawl.seccomp" bytemare/crawl:crawler.latest