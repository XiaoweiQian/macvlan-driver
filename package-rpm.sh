#!/bin/bash

rpmName=docker-macvlan
rpmVersion="0.2.0"
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
            --define "_pkgdir $pkgDir" \
            /root/rpmbuild/SPECS/${rpmName}.spec

echo "Package end."
