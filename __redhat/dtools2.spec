%define debug_package   %{nil}
%define _build_id_links none
%define _name dtools
%define _prefix /opt
%define _bash_completionsdir /usr/share/bash-completion/completions
%define _zsh_completionsdir  /usr/share/zsh/site-functions
%define _version 2.7.1
%define _rel 6
%define _arch x86_64
%define _binaryname dtools

Name:       dtools
Version:    %{_version}
Release:    %{_rel}
Summary:    docker/podman client

Group:      containers
License:    GPL2.0
URL:        https://git.famillegratton.net:3000/devops/dtools2.git

Source0:    %{name}-%{_version}.tar.gz
#BuildArchitectures: x86_64
BuildRequires: gcc

%description
docker/podman client

%prep
%autosetup

%build
cd src
CGO_ENABLED=0 /opt/go/bin/go build -trimpath -ldflags="-s -w -buildid=" -o %{_builddir}/%{name}-%{version}/%{_binaryname} .

%clean
rm -rf $RPM_BUILD_ROOT

%pre

%install
rm -rf %{buildroot}
install -Dpm 0755 %{_builddir}/%{name}-%{version}/%{_binaryname} %{buildroot}%{_bindir}/%{_binaryname}

%post
# Bash completion — always install
mkdir -p /etc/usr/share/bash-completion/completions
/opt/bin/%{_binaryname} completion bash > %{_bash_completionsdir}/%{_binaryname}

# Zsh completion — only if zsh is present
if command -v zsh > /dev/null 2>&1; then
    mkdir -p %{_zsh_completionsdir}/zsh/site-functions
    /opt/bin/%{_binaryname} completion zsh > %{_zsh_completionsdir}/%{_binaryname}
fi

%preun

%postun
if [ $1 -eq 0 ]; then
    # $1 == 0 means this is a full uninstall, not an upgrade
    rm -f %{_bash_completionsdir}/%{_binaryname}
    rm -f %{_zsh_completionsdir}/_%{_binaryname}
fi

%files
%defattr(-,root,root,-)
%{_bindir}/%{_binaryname}


%changelog
* Mon Jul 20 2026 Binary package builder <builder@famillegratton.net> 2.7.1-5
- version bump to ensure all packages are consistent
- chore: update changelog for 2.7.1-3
- fixed typo in specfile, rpmbuild refresh
- fixed possible issue in numbering
- fixed postinstall apk script
- chore: update changelog for 2.7.1-2
- Enforce package name to dtools, dtools2 except for Alpine

* Mon Jul 20 2026 Binary package builder <builder@famillegratton.net> 2.7.1-3
- fixed typo in specfile, rpmbuild refresh
- fixed possible issue in numbering
- fixed postinstall apk script
- chore: update changelog for 2.7.1-2
- Enforce package name to dtools, dtools2 except for Alpine

* Sat Jul 18 2026 Binary package builder <builder@famillegratton.net> 2.7.1-2
- Enforce package name to dtools, dtools2 except for Alpine

* Fri Jul 17 2026 Binary package builder <builder@famillegratton.net> 2.7.1-1
- renamed specfile in rpmbuild
- alpine packaging rename
- Changed package name (arch)
- Merge remote-tracking branch 'refs/remotes/origin/develop' into develop
- Added test suites
- version bump and packaging scripts enhancements
- Doc update for the build subcommand
- Merge remote-tracking branch 'refs/remotes/origin/develop' into develop
- Revamped documentation
- Merge branch 'develop' of ssh://git.famillegratton.net:9722/devops/dtools2 into develop
- refreshed rhel support
- cosmetic output fix
- removed tito
- Fixed Makefile, added the Release variable
- moved rpm build script and specfile into its owm context

* Fri May 01 2026 Binary package builder <builder@famillegratton.net> 2.70.00-0
- deps update (jean-francois@famillegratton.net)
- Another batch of Arch fixes (builder@famillegratton.net)
- Fixed the build script's permission denied (jean-francois@famillegratton.net)
- Perm fixes (builder@famillegratton.net)
- blacklist list (lsb) now a root command (jean-francois@famillegratton.net)
- changed lsc behaviour (jean-francois@famillegratton.net)
- Version update, added archlinux packaging support (jean-
  francois@famillegratton.net)
- blacklist commands now ranked as 1st citizen commands (jean-
  francois@famillegratton.net)
- re-implement build (jean-francois@famillegratton.net)

* Tue Mar 31 2026 Binary package builder <builder@famillegratton.net> 2.60.00-0
- updated helperFunctions to v5 (jean-francois@famillegratton.net)
- added flags to dtools run (jean-francois@famillegratton.net)
- Important CAVEAT doc update (jean-francois@famillegratton.net)

* Tue Mar 03 2026 Binary package builder <builder@famillegratton.net> 2.52.00-0
- Closing dev on dtools2 for now (jean-francois@famillegratton.net)
- 'fixed' dtools vol create issue. unfixable right now (jean-
  francois@famillegratton.net)
- another buildkit fix (jean-francois@famillegratton.net)
- interim submit (jean-francois@famillegratton.net)
- builddeps update (webhook test) (jean-francois@famillegratton.net)
- version bump (jean-francois@famillegratton.net)
- more interim stuff (jean-francois@famillegratton.net)
- interim commit (jean-francois@famillegratton.net)

* Wed Feb 18 2026 Binary package builder <builder@famillegratton.net> 2.51.00-0
- Fixed build and run, version bump, builddeps update (jean-
  francois@famillegratton.net)
- added command to volume subcommand (jean-francois@famillegratton.net)
- various version fixes (jean-francois@famillegratton.net)
- jetbrains renaming went overboard, correcting... (jean-
  francois@famillegratton.net)
- refactor : merged build and run subpackages (jean-
  francois@famillegratton.net)
- doc update (jean-francois@famillegratton.net)

* Fri Feb 13 2026 Binary package builder <builder@famillegratton.net> 2.50.00-0
- fixed perms on shell script (builder@famillegratton.net)
- Automatic commit of package [dtools] release [2.50.00-0].
  (builder@famillegratton.net)
- completed image inspect (jean-francois@famillegratton.net)
- completed network inspect and volume inspect (jean-
  francois@famillegratton.net)
- go version bump (jean-francois@famillegratton.net)
- doc update (jean-francois@famillegratton.net)
- completed dtools clean fix (jean-francois@famillegratton.net)
- imagespecs are now sorted (sort -u) (jean-francois@famillegratton.net)
- builddeps updates (jean-francois@famillegratton.net)
- refactored filenames to ensure no issue if running on NTFS (jean-
  francois@famillegratton.net)
- moved inspect-related types in their own file (jean-
  francois@famillegratton.net)
- Changed error type in inspect (jean-francois@famillegratton.net)
- completed container inspect (jean-francois@famillegratton.net)
- doc update (jean-francois@famillegratton.net)
- version bump (jean-francois@famillegratton.net)
- Fixed http timeout issue (jean-francois@famillegratton.net)
- Automatic commit of package [dtools] release [2.40.01-0].
  (builder@famillegratton.net)
- Fixed embarassing version numbering snafu (jean-francois@famillegratton.net)
- Automatic commit of package [dtools] release [2.40.00-0].
  (builder@famillegratton.net)
- Aligned all return codes with ce.CustomError (jean-
  francois@famillegratton.net)
- Completed commit (jean-francois@famillegratton.net)
- Completed load (jean-francois@famillegratton.net)
- Completed save (jean-francois@famillegratton.net)
- Code-reuse : container attach (jean-francois@famillegratton.net)
- Preparing 2.31.00 : version bump and doc update (jean-
  francois@famillegratton.net)
- README.md clarifications (jean-francois@famillegratton.net)
- Doc update (jean-francois@famillegratton.net)
- yet another forgotten file update (jean-francois@famillegratton.net)
- updated CHANGELOG (jean-francois@famillegratton.net)
- Automatic commit of package [dtools] release [2.30.00-0].
  (builder@famillegratton.net)
- reverted tag as specfile had the wrong release number
  (builder@famillegratton.net)
- Automatic commit of package [dtools] release [2.30.00-1].
  (builder@famillegratton.net)
- closing 2.30.00 with current feature set (jean-francois@famillegratton.net)
- added GO version in version name (jean-francois@famillegratton.net)
- moved system catalog and system tags to the new 'get' subcommand (jean-
  francois@famillegratton.net)
- build deps update (jean-francois@famillegratton.net)
- updated to go v1.25.6 (jean-francois@famillegratton.net)
- interim commit (jean-francois@famillegratton.net)
- Reverted colour on dtools lsi (jean-francois@famillegratton.net)
- Completed cp (jean-francois@famillegratton.net)
- preparing v2.30.00 : version bump (jean-francois@famillegratton.net)
- Updated README.md (jean-francois@famillegratton.net)
- Completed the documentation (jean-francois@famillegratton.net)
- Updated CHANGELOG (jean-francois@famillegratton.net)
- Automatic commit of package [dtools] release [2.21.01-1].
  (builder@famillegratton.net)
- Bumped package version number (builder@famillegratton.net)
- Automatic commit of package [dtools] release [2.21.01-0].
  (builder@famillegratton.net)
- Added the --format flag, used when the --json flag is set (jean-
  francois@famillegratton.net)
- Automatic commit of package [dtools] release [2.21.00-0].
  (builder@famillegratton.net)
- completed json formatting (jean-francois@famillegratton.net)
- version bump (jean-francois@famillegratton.net)

* Fri Feb 13 2026 Binary package builder <builder@famillegratton.net> 2.50.00-0
- completed image inspect (jean-francois@famillegratton.net)
- completed network inspect and volume inspect (jean-
  francois@famillegratton.net)
- go version bump (jean-francois@famillegratton.net)
- doc update (jean-francois@famillegratton.net)
- completed dtools clean fix (jean-francois@famillegratton.net)
- imagespecs are now sorted (sort -u) (jean-francois@famillegratton.net)
- builddeps updates (jean-francois@famillegratton.net)
- refactored filenames to ensure no issue if running on NTFS (jean-
  francois@famillegratton.net)
- moved inspect-related types in their own file (jean-
  francois@famillegratton.net)
- Changed error type in inspect (jean-francois@famillegratton.net)
- completed container inspect (jean-francois@famillegratton.net)
- doc update (jean-francois@famillegratton.net)
- version bump (jean-francois@famillegratton.net)
- Fixed http timeout issue (jean-francois@famillegratton.net)

* Fri Jan 30 2026 Binary package builder <builder@famillegratton.net> 2.40.01-0
- Fixed embarassing version numbering snafu (jean-francois@famillegratton.net)

* Fri Jan 30 2026 Binary package builder <builder@famillegratton.net> 2.40.00-0
- Aligned all return codes with ce.CustomError (jean-
  francois@famillegratton.net)
- Completed commit (jean-francois@famillegratton.net)
- Completed load (jean-francois@famillegratton.net)
- Completed save (jean-francois@famillegratton.net)
- Code-reuse : container attach (jean-francois@famillegratton.net)
- Preparing 2.31.00 : version bump and doc update (jean-
  francois@famillegratton.net)
- README.md clarifications (jean-francois@famillegratton.net)
- Doc update (jean-francois@famillegratton.net)
- yet another forgotten file update (jean-francois@famillegratton.net)
- updated CHANGELOG (jean-francois@famillegratton.net)
- closing 2.30.00 with current feature set (jean-francois@famillegratton.net)

* Thu Jan 29 2026 Binary package builder <builder@famillegratton.net> 2.30.00-0
- reverted tag as specfile had the wrong release number
  (builder@famillegratton.net)
- Automatic commit of package [dtools] release [2.30.00-1].
  (builder@famillegratton.net)
- added GO version in version name (jean-francois@famillegratton.net)
- moved system catalog and system tags to the new 'get' subcommand (jean-
  francois@famillegratton.net)
- build deps update (jean-francois@famillegratton.net)
- updated to go v1.25.6 (jean-francois@famillegratton.net)
- interim commit (jean-francois@famillegratton.net)
- Reverted colour on dtools lsi (jean-francois@famillegratton.net)
- Completed cp (jean-francois@famillegratton.net)
- preparing v2.30.00 : version bump (jean-francois@famillegratton.net)
- Updated README.md (jean-francois@famillegratton.net)
- Completed the documentation (jean-francois@famillegratton.net)
- Updated CHANGELOG (jean-francois@famillegratton.net)

* Thu Jan 29 2026 Binary package builder <builder@famillegratton.net> 2.30.00-1
- added GO version in version name (jean-francois@famillegratton.net)
- moved system catalog and system tags to the new 'get' subcommand (jean-
  francois@famillegratton.net)
- build deps update (jean-francois@famillegratton.net)
- updated to go v1.25.6 (jean-francois@famillegratton.net)
- interim commit (jean-francois@famillegratton.net)
- Reverted colour on dtools lsi (jean-francois@famillegratton.net)
- Completed cp (jean-francois@famillegratton.net)
- preparing v2.30.00 : version bump (jean-francois@famillegratton.net)
- Updated README.md (jean-francois@famillegratton.net)
- Completed the documentation (jean-francois@famillegratton.net)
- Updated CHANGELOG (jean-francois@famillegratton.net)

* Thu Jan 08 2026 Binary package builder <builder@famillegratton.net> 2.21.01-1
- Bumped package version number (builder@famillegratton.net)

* Thu Jan 08 2026 Binary package builder <builder@famillegratton.net> 2.21.01-0
- Added the --format flag, used when the --json flag is set (jean-
  francois@famillegratton.net)

* Wed Jan 07 2026 Binary package builder <builder@famillegratton.net> 2.21.00-0
- completed json formatting (jean-francois@famillegratton.net)
- version bump (jean-francois@famillegratton.net)

* Tue Jan 06 2026 Binary package builder <builder@famillegratton.net> 2.20.00-0
- fixed tito tag numbering (jean-francois@famillegratton.net)
- Fixed version/tag number for RPMs (jean-francois@famillegratton.net)
- Automatic commit of package [dtools] release [2.13.00-0].
  (builder@famillegratton.net)
- Fixed clean() that was using the wrong variables (jean-
  francois@famillegratton.net)
- Fixed wrong field in candidates (jean-francois@famillegratton.net)
- Completed systems commands: info, clean, rms (jean-
  francois@famillegratton.net)

* Tue Jan 06 2026 Binary package builder <builder@famillegratton.net> 2.13.00-0
- Fixed clean() that was using the wrong variables (jean-
  francois@famillegratton.net)
- Fixed wrong field in candidates (jean-francois@famillegratton.net)
- Completed systems commands: info, clean, rms (jean-
  francois@famillegratton.net)

* Mon Jan 05 2026 Binary package builder <builder@famillegratton.net> 2.12.01-0
- Fixed volume list and prune (jean-francois@famillegratton.net)
- interim sync (jean-francois@famillegratton.net)
- Update README.md (jean-francois@famillegratton.net)
- Updated doc to test image embedding 4 (jean-francois@famillegratton.net)
- Updated doc to test image embedding 3 (jean-francois@famillegratton.net)
- Updated doc to test image embedding 2 (jean-francois@famillegratton.net)
- Updated doc to test image embedding (jean-francois@famillegratton.net)
- doc update (jean-francois@famillegratton.net)

* Sun Jan 04 2026 Binary package builder <builder@famillegratton.net> 2.12.00-0
- bumped version number (builder@famillegratton.net)
- Fixed tito tag issue (builder@famillegratton.net)
- Automatic commit of package [dtools] release [2.12.00-0].
  (builder@famillegratton.net)
- reverted partial tito build (jean-francois@famillegratton.net)
- Fixed wrong function name (jean-francois@famillegratton.net)
- Automatic commit of package [dtools] release [2.12.00-0].
  (builder@famillegratton.net)
- Added volume create (jean-francois@famillegratton.net)

* Sun Jan 04 2026 Binary package builder <builder@famillegratton.net>
- Fixed tito tag issue (builder@famillegratton.net)
- Automatic commit of package [dtools] release [2.12.00-0].
  (builder@famillegratton.net)
- reverted partial tito build (jean-francois@famillegratton.net)
- Fixed wrong function name (jean-francois@famillegratton.net)
- Automatic commit of package [dtools] release [2.12.00-0].
  (builder@famillegratton.net)
- Added volume create (jean-francois@famillegratton.net)

* Sun Jan 04 2026 Binary package builder <builder@famillegratton.net> 2.12.00-0
- reverted partial tito build (jean-francois@famillegratton.net)
- Fixed wrong function name (jean-francois@famillegratton.net)
- Automatic commit of package [dtools] release [2.12.00-0].
  (builder@famillegratton.net)
- Added volume create (jean-francois@famillegratton.net)

* Sun Jan 04 2026 Binary package builder <builder@famillegratton.net> 2.11.00-0
- Added the progress flag to the build command (jean-
  francois@famillegratton.net)
- updated the restore repo script (builder@famillegratton.net)

* Sun Jan 04 2026 Binary package builder <builder@famillegratton.net> 2.10.00-0
- Version bump (jean-francois@famillegratton.net)
- Completed the build subcommand (forgotten) (jean-francois@famillegratton.net)

* Sat Jan 03 2026 Binary package builder <builder@famillegratton.net> 2.00.01-0
- Fixed wrong path for config files (jean-francois@famillegratton.net)

* Sat Jan 03 2026 Binary package builder <builder@famillegratton.net> 2.00.00-0
- new package built with tito

* Sat Jan 03 2026 Binary package builder <builder@famillegratton.net> 0.80.00-0
- Completed volume prune (jean-francois@famillegratton.net)
- made ls/rm subcomments more consistent across the board (jean-
  francois@famillegratton.net)
- Completed volume rm (jean-francois@famillegratton.net)
- Completed volume ls (jean-francois@famillegratton.net)
- version bump (jean-francois@famillegratton.net)

* Wed Dec 31 2025 Binary package builder <builder@famillegratton.net> 0.70.00-0
- Completed sys catalog and sys tags (jean-francois@famillegratton.net)
- Fixed behaviour of sys tags and sys catalog, needs fixing output now (jean-
  francois@famillegratton.net)
- completed sys catalog (jean-francois@famillegratton.net)
- Completed get catalog and get tags (jean-francois@famillegratton.net)
- interim sync as refactoring is getting messy (jean-
  francois@famillegratton.net)
- Fixed gitignore (jean-francois@famillegratton.net)
- added temp gitignore (jean-francois@famillegratton.net)
- Completed the registry subcommands (jean-francois@famillegratton.net)
- Version bump (jean-francois@famillegratton.net)
- Updated packaging scripts (builder@famillegratton.net)

* Mon Dec 29 2025 Binary package builder <builder@famillegratton.net> 0.60.00-0
- Completed the network subcommand (jean-francois@famillegratton.net)
- Added network add (jean-francois@famillegratton.net)
- Fixed conditional in computeNetworkUse to add a simple true/false value for
  simpler spot checks (jean-francois@famillegratton.net)
- Prettified net ls (jean-francois@famillegratton.net)
- net ls completed (jean-francois@famillegratton.net)
- added restart / restartall, with issue on restartall -k (jean-
  francois@famillegratton.net)
- added a forgotten subcommand, restart (jean-francois@famillegratton.net)
- version bump (jean-francois@famillegratton.net)
- Automatic commit of package [dtools2] release [0.51.00-0].
  (builder@famillegratton.net)
- interim sync (jean-francois@famillegratton.net)
- Moved the Debug variable in another package to avoid circular imports (jean-
  francois@famillegratton.net)
- Removed un-needed DEB build files (jean-francois@famillegratton.net)
- Network subcommand stub (jean-francois@famillegratton.net)
- Various minor fixes (jean-francois@famillegratton.net)
- version bump (jean-francois@famillegratton.net)
- Coloured output for image ls (jean-francois@famillegratton.net)

* Sun Dec 14 2025 Binary package builder <builder@famillegratton.net> 0.51.00-0
- Version bump and completed/fixed containers and images (jean-
  francois@famillegratton.net)
- Removed forgotten Code var from customError (jean-
  francois@famillegratton.net)
- Fixed container rm -f, removed all exit codes from ce.CustomError (jean-
  francois@famillegratton.net)
- Completed killall/kill (jean-francois@famillegratton.net)
- Interim commit (jean-francois@famillegratton.net)
- Changed binary name from dtools2 to dtools (jean-francois@famillegratton.net)
- fixed removal (jean-francois@famillegratton.net)
- Fixed regression in the blacklist subpackage (jean-
  francois@famillegratton.net)
- Fixed image tag (jean-francois@famillegratton.net)
- fixed container rename (jean-francois@famillegratton.net)
- Version bump (jean-francois@famillegratton.net)

* Wed Dec 10 2025 Binary package builder <builder@famillegratton.net> 0.40.00-0
- Fixed ENV vars issue in APK build scripts (jean-francois@famillegratton.net)
- Re-instated variable removed by mistake (jean-francois@famillegratton.net)
- Added image tag (jean-francois@famillegratton.net)
- Simplified the http queries (jean-francois@famillegratton.net)
- Version bump (jean-francois@famillegratton.net)
- Completed container rename command (jean-francois@famillegratton.net)
- Completed image ls (jean-francois@famillegratton.net)
- updated DEB package buildscripts (jean-francois@famillegratton.net)

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


