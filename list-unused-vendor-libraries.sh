#!/usr/bin/env bash

# from within a status-go git checkout
# lists each orphaned vendor library on stdout
# lists references to each vendor library on stderr

for vendorPath in `git rev-parse --show-toplevel`/vendor/* ; do
  pushd $vendorPath >&2
  for library in * ; do
    # list library references to stderr
    echo library $(basename ${vendorPath})/$library >&2
    pushd ../../.. >&2
    pwd >&2
    grep -nHre "$(basename ${vendorPath})/$library" ./status-{go,react} \
      | grep -ve "^./status-go/vendor/$(basename ${vendorPath})/$library" \
      | grep -ve "git/index" \
      | grep -ve ".sw[a-z] " >&2

    if [ $? -ne 0 ]; then # prior grep failed
      # list orphan to stdout
      echo "$(basename ${vendorPath})/$library ";
    fi
    popd >&2
  done
  popd >&2
done
