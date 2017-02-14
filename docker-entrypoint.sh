#!/bin/sh

if [ -z "$CVMFS_HTTP_PROXY" ]; then
    export CVMFS_HTTP_PROXY=http://ca-proxy.cern.ch:3128
fi

echo CVMFS_HTTP_PROXY="$CVMFS_HTTP_PROXY" >> /etc/cvmfs/default.local
echo CVMFS_CACHE_BASE=/var/cache/cvmfs >> /etc/cvmfs/default.local
echo CVMFS_QUOTA_LIMIT=20000 >> /etc/cvmfs/default.local

/usr/sbin/docker-volume-cvmfs
