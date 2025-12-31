#!/usr/bin/env bash

PKGDIR="dtools2-0.80.00-0_amd64"

mkdir -p ${PKGDIR}/opt/bin ${PKGDIR}/DEBIAN
mkdir -p ${PKGDIR}/opt/bin ${PKGDIR}/DEBIAN
#for i in control preinst prerm postinst postrm;do
  mv control ${PKGDIR}/DEBIAN/
#done

echo "Building binary from source"
cd ../src
CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -buildid=" -o ../__debian/${PKGDIR}/opt/bin/dtools .
sudo chown 0:0 ../__debian/${PKGDIR}/opt/bin/dtools

echo "Binary built. Now packaging..."
cd ../__debian/
dpkg-deb -b ${PKGDIR}
