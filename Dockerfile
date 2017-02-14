FROM fedora:23

RUN yum install -y \
	wget

RUN wget -q https://ecsft.cern.ch/dist/cvmfs/cvmfs-2.1.20/cvmfs-2.1.20-1.fc21.x86_64.rpm; \
	wget -q https://ecsft.cern.ch/dist/cvmfs/cvmfs-config/cvmfs-config-default-latest.noarch.rpm; \
	yum -y install cvmfs-2.1.20-1.fc21.x86_64.rpm cvmfs-config-default-latest.noarch.rpm

RUN rmdir /cvmfs

ADD docker-volume-cvmfs /usr/sbin/docker-volume-cvmfs
RUN chmod 755 /usr/sbin/docker-volume-cvmfs

CMD ["/usr/sbin/docker-volume-cvmfs"]
