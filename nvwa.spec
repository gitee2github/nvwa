Name:           nvwa
Version:        0.0.1
Release:        1%{?dist}
Summary:        a tool used for openEuler kernel update

License:        MulanPSL-2.0
URL:            https://gitee.com/openeuler/nvwa
Source:         %{name}-v%{version}.tar.gz

BuildRequires:  golang

%description
A tool used to automate the process of seamless update of the openEuler.

%prep
%autosetup -n %{name}


%build

cd src
go get %{name}
go build
cd -

%install

mkdir -p %{buildroot}/%{_bindir}
mkdir -p %{buildroot}/etc/%{name}

install -m 0644 %{_builddir}/src/%{name} %{buildroot}/%{_bindir}/
install %{_builddir}/config/* %{buildroot}/etc/%{name}

%files
%license LICENSE
%dir /etc/%{name}/
/etc/%{name}/*
%{_bindir}/%{name}


%changelog
* Thu Feb 18 2021 anatasluo <luolongjun@huawei.com>
- Update to 0.0.1