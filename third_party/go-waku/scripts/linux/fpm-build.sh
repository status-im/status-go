#!/bin/bash

parent_path=$( cd "$(dirname "${BASH_SOURCE[0]}")" ; pwd -P )/../../

VERSION=`cat ${parent_path}/VERSION`

if [ ! -f ${parent_path}/build/waku ]
then
    echo "waku binary does not exist. Execute make first"
    exit
fi

tmpdir=`mktemp -d`

chmod 777 ${tmpdir}

cp ${parent_path}/build/waku ${tmpdir}

strip --strip-unneeded ${tmpdir}/waku

pushd ${tmpdir}

fpm_build () {
    fpm \
    -s dir -t $1 \
    -p gowaku-${VERSION}-x86_64.$1 \
    --name go-waku \
    --license "MIT, Apache 2.0" \
    --version ${VERSION} \
    --architecture x86_64 \
    --depends libc6 \
    --description "Go implementation of Waku v2 protocol" \
    --url "https://github.com/waku-org/go-waku" \
    --maintainer "Richard Ramos <richard@status.im>" \
    waku=/usr/bin/waku
}

fpm_build "deb"
fpm_build "rpm"

ls

mv *.deb *.rpm ${parent_path}/build/.

popd
