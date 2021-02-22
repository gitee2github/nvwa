Name:           nvwa
Version:        0.0.1
Release:        1
Summary:        a tool used for openEuler kernel update

License:        MulanPSL-2.0 and Apache-2.0 and MIT
URL:            https://gitee.com/openeuler/nvwa
Source:         %{name}-v%{version}.tar.gz

BuildRequires:  golang >= 1.13
Requires:       kexec-tools criu
Requires:       systemd-units iptables-services iproute

%description
A tool used to automate the process of seamless update of the openEuler.

%global debug_package %{nil}

%prep
%autosetup -n %{name}-v%{version}


%build

cd src
go build -mod=vendor
cd -

%install

mkdir -p %{buildroot}/%{_bindir}
mkdir -p %{buildroot}/etc/%{name}
mkdir -p %{buildroot}/etc/%{name}/running/
mkdir -p %{buildroot}/etc/systemd/system/

install -m 0750 %{_builddir}/%{name}-v%{version}/src/%{name} %{buildroot}/%{_bindir}/
install -m 0640 %{_builddir}/%{name}-v%{version}/config/%{name}-restore.yaml %{buildroot}/etc/%{name}/
install -m 0640 %{_builddir}/%{name}-v%{version}/config/%{name}-server.yaml %{buildroot}/etc/%{name}/

install -m 0644 %{_builddir}/%{name}-v%{version}/%{name}.service %{buildroot}/etc/systemd/system/

%post
%systemd_post %{name}.service

%preun
%systemd_preun %{name}.service

%postun
%systemd_postun_with_restart %{name}.service

%files
%license LICENSE
%dir /etc/%{name}/
/etc/%{name}/%{name}-restore.yaml
/etc/%{name}/%{name}-server.yaml
/etc/systemd/system/%{name}.service
%{_bindir}/%{name}


%changelog
* Thu Feb 18 2021 anatasluo <luolongjun@huawei.com>
- Update to 0.0.1
