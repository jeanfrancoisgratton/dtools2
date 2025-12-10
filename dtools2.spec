%ifarch aarch64
%global _arch aarch64
%global BuildArchitectures aarch64
%endif

%ifarch x86_64
%global _arch x86_64
%global BuildArchitectures x86_64
%endif

%define debug_package   %{nil}
%define _build_id_links none
%define _name dtools2
%define _prefix /opt
%define _version 0.40.00
%define _rel 0
#%define _arch x86_64
%define _binaryname dtools2

Name:       dtools2
Version:    %{_version}
Release:    %{_rel}
Summary:    docker/podman client

Group:      containers
License:    GPL2.0
URL:        https://git.famillegratton.net:3000/devops/dtools2.git

Source0:    %{name}-%{_version}.tar.gz
#BuildArchitectures: x86_64
BuildRequires: gcc
#Requires: sudo
#Obsoletes: vmman1 > 1.140

%description
docker/podman client

%prep
%autosetup

%build
cd %{_sourcedir}/%{_name}-%{_version}/src
PATH=$PATH:/opt/go/bin CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -buildid=" -o %{_sourcedir}/%{_binaryname} .

%clean
rm -rf $RPM_BUILD_ROOT

%pre
exit 0

%install
install -Dpm 0755 %{_sourcedir}/%{_binaryname} %{buildroot}%{_bindir}/%{_binaryname}

%post

%preun

%postun

%files
%defattr(-,root,root,-)
%{_bindir}/%{_binaryname}


%changelog
* Sun Dec 07 2025 Binary package builder <builder@famillegratton.net> 0.30.00-0
- Fixing APKBUILD (jean-francois@famillegratton.net)
- Completed the container subcommand (jean-francois@famillegratton.net)
- Edited container subcommand name (jean-francois@famillegratton.net)
- Completed container rm (jean-francois@famillegratton.net)
- Fixed comments typo (jean-francois@famillegratton.net)
- Changed TLS flag (jean-francois@famillegratton.net)
- interim sync (jean-francois@famillegratton.net)
- Fixed blacklist pointer issue (jean-francois@famillegratton.net)
- Fully migrated blacklist from error to customError (jean-
  francois@famillegratton.net)
- Fixed go vet issue with non-constant string in Fprint() (jean-
  francois@famillegratton.net)
- Added stub for remove (jean-francois@famillegratton.net)
- added 'containers rm' stub (jean-francois@famillegratton.net)
- GO version bump, version bump (jean-francois@famillegratton.net)

* Tue Dec 02 2025 Binary package builder <builder@famillegratton.net> 0.21.00-0
- Fixed mountpoints display issue, cutting a new release (jean-
  francois@famillegratton.net)
- Completed container info (jean-francois@famillegratton.net)
- Refactored the rest subpackage (jean-francois@famillegratton.net)
- Fully migrated from COBRA func RunE() error -> func Run() (jean-
  francois@famillegratton.net)
- interim commit (jean-francois@famillegratton.net)
- Fixed branches merge mess (jean-francois@famillegratton.net)
- builddeps update (jean-francois@famillegratton.net)
- Completed the BLACKLIST subcommand; error handling to come later (jean-
  francois@famillegratton.net)
- Completed bl ls and bl add (jean-francois@famillegratton.net)
- Sync before branching out to new branch (jean-francois@famillegratton.net)
- sync before branching out (jean-francois@famillegratton.net)
- Completed image push (jean-francois@famillegratton.net)
- Completed container ls (jean-francois@famillegratton.net)
- Version bump (jean-francois@famillegratton.net)
- Fixed APK script (jean-francois@famillegratton.net)


