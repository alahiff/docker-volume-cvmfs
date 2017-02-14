%global gopath %{_tmppath}/gopath
%define debug_package %{nil}
%define _unitdir /lib/systemd/system
%global cvmfs %{gopath}/src/gitlab.cern.ch/cloud-infrastructure/docker-volume-cvmfs

Name:       docker-volume-cvmfs
Version:    0.3.1
Release:    3
Summary:    Docker volume plugin for CVMFS
Source0:    https://gitlab.cern.ch/cloud-infrastructure/docker-volume-cvmfs/%{name}-%{version}.tar.gz

Group:      Development/Languages
License:    ASL 2.0
URL:        http://gitlab.cern.ch/cloud-infrastructure/docker-volume-cvmfs

BuildRequires: golang
BuildRequires: systemd

Requires: cvmfs >= 2.2

%description
Docker volume plugin for the CVMFS filesystem.
https://cernvm.cern.ch/portal/filesystem

%prep
%setup -q -n %{name}-%{version}
rm -rf %{gopath}
mkdir -p %{gopath} %{gopath}/pkg %{gopath}/bin %{gopath}/src %{cvmfs}

%build
cp -R * %{cvmfs}
cd %{cvmfs}
ln -sf `pwd`/vendor/* %{gopath}/src # we need to support go 1.4 (no vendor)
GOPATH=%{gopath} go build .

%install
cd %{cvmfs}

# binary
GOPATH=%{gopath} go install .
install -d %{buildroot}%{_bindir}
install -m 755 %{_tmppath}/gopath/bin/%{name} %{buildroot}%{_bindir}

# default cvmfs config
install -d %{buildroot}%{_sysconfdir}
install -d %{buildroot}%{_sysconfdir}/cvmfs
install default.local %{buildroot}%{_sysconfdir}/cvmfs

# systemd
install -d %{buildroot}%{_unitdir}
install -p -m 644 %{name}.service %{buildroot}%{_unitdir}
mkdir -p %{buildroot}%{_sysconfdir}/systemd/system/multi-user.target.wants
ln -sf %{_unitdir}/%{name}.service %{buildroot}%{_sysconfdir}/systemd/system/multi-user.target.wants/%{name}.service

# logrotate
install -p -D -m 644 docker-volume-cvmfs.logrotate %{buildroot}%{_sysconfdir}/logrotate.d/%{name}

# kubernetes plugin (symlink)
install -d %{buildroot}%{_libexecdir}/kubernetes/kubelet-plugins/volume/exec/cern~cvmfs
ln -sf %{_bindir}/docker-volume-cvmfs %{buildroot}%{_libexecdir}/kubernetes/kubelet-plugins/volume/exec/cern~cvmfs/cvmfs

%files
%doc LICENSE
%doc README.md
%{_bindir}/%{name}
%config(noreplace) %{_sysconfdir}/cvmfs/default.local
%{_unitdir}/%{name}.service
%{_sysconfdir}/systemd/system/multi-user.target.wants/%{name}.service
%{_sysconfdir}/logrotate.d/%{name}
%{_libexecdir}/kubernetes/kubelet-plugins/volume/exec/cern~cvmfs/cvmfs

%changelog
* Fri Nov 11 2016 Ricardo Rocha <ricardo.rocha@cern.ch> 0.3.1-3
- Set ulimit to 16384 (upstream recommended), in systemd service

* Thu Jul 14 2016 Ricardo Rocha <ricardo.rocha@cern.ch> 0.3.1-2
- Add symlink for systemd multi-user target

* Wed Jul 13 2016 Ricardo Rocha <ricardo.rocha@cern.ch> 0.3.1-1
- Update for new release (umount renamed to unmount)

* Thu Jul 7 2016 Ricardo Rocha <ricardo.rocha@cern.ch> 0.3.0-3
- Fix path of kubernetes cern cvmfs plugin symlink

* Tue Jul 5 2016 Ricardo Rocha <ricardo.rocha@cern.ch> 0.3.0-2
- Rebuild for koji force

* Tue Jul 5 2016 Ricardo Rocha <ricardo.rocha@cern.ch> 0.3.0-1
- Add support for kubernetes

* Tue Jun 14 2016 Ricardo Rocha <ricardo.rocha@cern.ch> 0.2.0-1
- Add config option for mount point

* Fri May 27 2016 Ricardo Rocha <ricardo.rocha@cern.ch> 0.1.1-5
- Redirect stdout to /var/log file in systemd script

* Fri May 27 2016 Ricardo Rocha <ricardo.rocha@cern.ch> 0.1.1-2
- Add logrotate script

* Fri May 27 2016 Ricardo Rocha <ricardo.rocha@cern.ch> 0.1.1-1
- New release for 0.1.1

* Thu May 26 2016 Ricardo Rocha <ricardo.rocha@cern.ch> 0.1.0-2
- Added systemd scripts

* Wed May 25 2016 Ricardo Rocha <ricardo.rocha@cern.ch> 0.1.0-1
- First release
