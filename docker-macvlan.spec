Name: docker-macvlan
Version: %{_version}
Release: %{_release}%{?dist}
Summary: The open-source application container engine
Group: Tools/Docker

License: ASL 2.0
Source: %{name}.tar.gz

URL: https://dockerproject.org
Vendor: suning
Packager: suning <16030138@cnsuning.com>

# is_systemd conditional
%if 0%{?fedora} >= 21 || 0%{?centos} >= 7 || 0%{?rhel} >= 7 || 0%{?suse_version} >= 1210
%global is_systemd 1
%endif

# required packages for build
# only require systemd on those systems
%if 0%{?is_systemd}
%if 0%{?suse_version} >= 1210
BuildRequires: systemd-rpm-macros
%{?systemd_requires}
%else
%if 0%{?fedora} >= 25
# Systemd 230 and up no longer have libsystemd-journal (see https://bugzilla.redhat.com/show_bug.cgi?id=1350301)
BuildRequires: pkgconfig(systemd)
Requires: systemd-units
%else
BuildRequires: pkgconfig(systemd)
Requires: systemd-units
BuildRequires: pkgconfig(libsystemd-journal)
%endif
%endif
%else
Requires(post): chkconfig
Requires(preun): chkconfig
# This is for /sbin/service
Requires(preun): initscripts
%endif

# required packages on install
Requires: /bin/sh
Requires: iptables
%if !0%{?suse_version}
Requires: libcgroup
%else
Requires: libcgroup1
%endif
Requires: tar
Requires: xz
%if 0%{?fedora} >= 21 || 0%{?centos} >= 7 || 0%{?rhel} >= 7 || 0%{?oraclelinux} >= 7
# Resolves: rhbz#1165615
Requires: device-mapper-libs >= 1.02.90-1
%endif
%if 0%{?oraclelinux} >= 6
# Require Oracle Unbreakable Enterprise Kernel R4 and newer device-mapper
Requires: kernel-uek >= 4.1
Requires: device-mapper >= 1.02.90-2
%endif

# DWZ problem with multiple golang binary, see bug
# https://bugzilla.redhat.com/show_bug.cgi?id=995136#c12
%if 0%{?fedora} >= 20 || 0%{?rhel} >= 7 || 0%{?oraclelinux} >= 7
%global _dwz_low_mem_die_limit 0
%endif

%description
Docker macvlan driver for swarmkit.

%prep
%if 0%{?centos} <= 6 || 0%{?oraclelinux} <=6
%setup -n %{name}
%else
%autosetup -n %{name}
%endif

%build
PKG_DIR=%{_pkgdir} ./make.sh

%check
./docker-macvlan-%{_origversion} -v

%install
# install binary
install -d $RPM_BUILD_ROOT/%{_bindir}
install -p -m 755 ./docker-macvlan-%{_origversion} $RPM_BUILD_ROOT/%{_bindir}/docker-macvlan

#install service
install -d $RPM_BUILD_ROOT/%{_unitdir}
install -p -m 644 docker-macvlan.service $RPM_BUILD_ROOT/%{_unitdir}/docker-macvlan.service

# list files owned by the package here
%files
/%{_bindir}/docker-macvlan
/%{_unitdir}/docker-macvlan.service

%post
%systemd_post docker-macvlan

%preun
%systemd_preun docker-macvlan

%postun
%systemd_postun_with_restart docker-macvlan

%changelog
