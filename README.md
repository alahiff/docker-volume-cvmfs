# docker-volume-cvmfs

This package provides management of cvmfs repositories in docker and kubernetes.

It provides a nicer interface to handle cvmfs volume definitions.

This is based on https://gitlab.cern.ch/cloud-infrastructure/docker-volume-cvmfs/ but with changes (in progress!) to make it useful to people outside of CERN:
* ability to handle repositories from multiple domains, not just cern.ch
* will read in /etc/cvmfs/domain.d/\<your_domain\>.local config files (previously these were ignored)
* uses CVMFS_CACHE_BASE as defined in the usual config files (previously used per-repo caches in /var/cache & ignored the config files)
* uses existing cvmfs uid rather than having a particular uid hard-wired (makes it easier to use existing CVMFS RPMs installed on a node)

## Requirements

Docker 1.9.x or above, Kubernetes 1.2.x or above.

## Installation

### Docker

The recommended way to run the daemon is to use the docker image.

You'll need to make sure the /cvmfs directory exists on the host.
```
docker run -d --name docker-volume-cvmfs --privileged --restart always \
	-v /cvmfs:/cvmfs:shared \
	-v /run/docker/plugins:/run/docker/plugins \
	-v /var/cache/cvmfs:/var/cache/cvmfs:shared \
	gitlab-registry.cern.ch/cloud-infrastructure/docker-volume-cvmfs
```

### CentOS 7

Package installation is available for CentOS7, just add the following repos:
```
yum install https://ecsft.cern.ch/dist/cvmfs/cvmfs-release/cvmfs-release-latest.noarch.rpm

tee /etc/yum.repos.d/cci7-utils.repo <<-'EOF'
[cci7-utils]
name=CERN Cloud Infrastructure Utils
baseurl=http://linuxsoft.cern.ch/internal/repos/cci7-utils-stable/x86_64/os/
enabled=1
gpgcheck=0
EOF
```

Install the packages and launch the service (assuming you have docker already running):
```
yum install -y docker-volume-cvmfs
systemctl restart docker-volume-cvmfs
```

Logging will be at /var/log/docker-volume-cvmfs.log.

### Other

Otherwise you'll need to compile it manually and do the install. Example for Ubuntu:
```
wget https://ecsft.cern.ch/dist/cvmfs/cvmfs-2.1.20/cvmfs_2.1.20_amd64.deb
wget https://ecsft.cern.ch/dist/cvmfs/cvmfs-config/cvmfs-config-default_latest_all.deb
sudo dpkg --install cvmfs_2.1.20_amd64.deb cvmfs-config-default_latest_all.deb
cat >/etc/cvmfs/default.local << EOF
CVMFS_HTTP_PROXY="http://ca-proxy.cern.ch:3128"
CVMFS_CACHE_BASE=/var/cache/cvmfs
CVMFS_QUOTA_LIMIT=20000
EOF
```

Build the binary (go build from this repo source) and launch it:
```
sudo docker-volume-cvmfs
docker-volume-cvmfs: registering with docker
docker-volume-cvmfs: listening on /run/docker/plugins/cvmfs.sock
```

## Usage

### Docker

As with all docker volumes, you can explicitly create them or on container creation.

To create an independent volume (the plugin should output some useful info):
```
sudo docker volume ls
sudo docker volume create -d cvmfs --name=cms.cern.ch
```
The plugin should have given the following:
```
docker-volume-cvmfs: create cms.cern.ch map[]
docker-volume-cvmfs: mounting cms.cern.ch in /cvmfs/cms.cern.ch
docker-volume-cvmfs: path {cms.cern.ch map[]}
```

Similarly, you can create the volume when launching the container:
```
sudo docker run -it --rm --volume-driver cvmfs -v cms.cern.ch:/cvmfs/cms.cern.ch centos:7 /bin/bash
[root@874cbf8199d0 /]# ls /cvmfs/cms.cern.ch
CMS@Home bootstrap_slc5_amd64_gcc462.log cmssw.git
...
```

Deleting a volume explicitly:
```
sudo docker volume rm cms.cern.ch
```
Plugin should give some info:
```
docker-volume-cvmfs: path {cms.cern.ch map[]}
docker-volume-cvmfs: remove {cms.cern.ch map[]}
docker-volume-cvmfs: unmounting cms.cern.ch in /cvmfs/cms.cern.ch
```

### Kubernetes

Here's a sample manifest file to use a cvmfs volume:
```
apiVersion: v1
kind: Pod
metadata:
  name: sample
spec:
  containers:
    - name: nginx
      image: nginx
      volumeMounts:
        - name: atlas
          mountPath: /cvmfs/atlas.cern.ch
        - name: atlas-condb
          mountPath: /cvmfs/atlas-condb.cern.ch
  volumes:
    - name: atlas
      flexVolume:
        driver: "cern/cvmfs"
        options:
          repository: "atlas.cern.ch"
    - name: atlas-condb
      flexVolume:
        driver: "cern/cvmfs"
        options:
          repository: "atlas-condb.cern.ch"
```

The name of the volume goes in the *repository* field. To create this pod:
```
kubectl create -f nginx-cvmfs.yaml
pod "sample" created

kubectl exec -it -p sample -c nginx /bin/bash
root@sample:/# ls /cvmfs/atlas.cern.ch/repo/
ATLASLocalRootBase  benchmarks	conditions  dev  sw  tools
```

### Using tags and hashes

CVMFS supports mounting a repository using a tag or a hashes, and so does this plugin.

You can use the *@* separator for tags, and *#* for hashes in the cvmfs repository name.

Example to mount *trunk-previous* of atlas.cern.ch in docker:
```
sudo docker volume create -d cvmfs atlas.cern.ch@trunk-previous
```

If nothing is specified in the name, the *trunk* tag is used.

## Troubleshooting
