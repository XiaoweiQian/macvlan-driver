#!/bin/bash

PKGDIR=./
if [ "$PKG_DIR" ]; then
    echo "set gopath"
    rm -rf .gopath
    mkdir -p .gopath
    ln -sf $GOPATH/src .gopath/
    export GOPATH="${PWD}/.gopath"
    echo $GOPATH
    PKGDIR=./.gopath/src/${PKG_DIR##*src}
fi

go build -o ./docker-macvlan $PKGDIR
