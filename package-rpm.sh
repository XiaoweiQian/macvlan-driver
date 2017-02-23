#!/bin/bash

VERSION="0.1.1"
rpmName=docker-macvlan
rpmVersion=$VERSION
rpmRelease="1.sn"
pkgDir=${PWD}

echo "Package start."
echo "rpmVersion="$rpmVersion
echo "rpmRelease="$rpmRelease
cp ./docker-macvlan.spec /root/rpmbuild/SPECS/
cp -r ../macvlan-driver ../${rpmName}
tar --exclude .git -zcf /root/rpmbuild/SOURCES/${rpmName}.tar.gz -P ../${rpmName}
rm -rf ../${rpmName}
rpmbuild -ba \
            --define "_release $rpmRelease" \
            --define "_version $rpmVersion" \
            --define "_origversion $VERSION" \
            --define "_pkgdir $pkgDir" \
            /root/rpmbuild/SPECS/${rpmName}.spec

echo "Package end."
