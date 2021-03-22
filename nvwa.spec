Name:           nvwa
Version:        0.1
Release:        2
Summary:        a tool used for openEuler kernel update

License:        MulanPSL-2.0 and Apache-2.0 and MIT and MPL-2.0
URL:            https://gitee.com/openeuler/nvwa
Source:         %{name}-v%{version}.tar.gz

BuildRequires:  golang >= 1.13
Requires:       kexec-tools criu
Requires:       systemd-units iptables-services iproute
Requires:       gcc

%description
A tool used to automate the process of seamless update of the openEuler.

%global debug_package %{nil}

%prep
%autosetup -n %{name}-v%{version}


%build

cd src
go build -mod=vendor
cd -

cd tools
gcc %{name}-pin.c -o %{name}-pin
cd -

%install

mkdir -p %{buildroot}/%{_bindir}
mkdir -p %{buildroot}/etc/%{name}
mkdir -p %{buildroot}/etc/%{name}/log
mkdir -p %{buildroot}/usr/lib/systemd/system
mkdir -p %{buildroot}/var/%{name}
mkdir -p %{buildroot}/var/%{name}/running

install -m 0750 %{_builddir}/%{name}-v%{version}/src/%{name} %{buildroot}/%{_bindir}/
install -m 0750 %{_builddir}/%{name}-v%{version}/tools/%{name}-pin %{buildroot}/%{_bindir}/
install -m 0640 %{_builddir}/%{name}-v%{version}/src/config/%{name}-restore.yaml %{buildroot}/etc/%{name}/
install -m 0640 %{_builddir}/%{name}-v%{version}/src/config/%{name}-server.yaml %{buildroot}/etc/%{name}/

install -m 0750 %{_builddir}/%{name}-v%{version}/misc/%{name}-pre.sh %{buildroot}/%{_bindir}/
install -m 0644 %{_builddir}/%{name}-v%{version}/misc/%{name}.service %{buildroot}/usr/lib/systemd/system
install -m 0644 %{_builddir}/%{name}-v%{version}/misc/%{name}-pre.service %{buildroot}/usr/lib/systemd/system

%post
%systemd_post %{name}.service
%systemd_post %{name}-pre.service

%preun
%systemd_preun %{name}.service

%postun
%systemd_postun_with_restart %{name}.service

%files
%license LICENSE
%dir /etc/%{name}/
%dir /etc/%{name}/log
%dir /var/%{name}
%dir /var/%{name}/running
/etc/%{name}/%{name}-restore.yaml
/etc/%{name}/%{name}-server.yaml
/usr/lib/systemd/system/%{name}.service
/usr/lib/systemd/system/%{name}-pre.service
%{_bindir}/%{name}
%{_bindir}/%{name}-pin
%{_bindir}/%{name}-pre.sh

%changelog
* Wed 17 Mar 2021 anatasluo <luolongjun@huawei.com>
- Update to 0.1-r2
* Thu Feb 18 2021 anatasluo <luolongjun@huawei.com>
- Update to 0.0.1
