SPEC    := dtools2.spec
NAME    := $(shell rpmspec -q --qf '%{name}\n' $(SPEC) | head -1)
VERSION := $(shell rpmspec -q --qf '%{version}\n' $(SPEC) | head -1)
RELEASE := $(shell rpmspec -q --qf '%{release}\n' ${SPEC} | head -1)
SOURCES := $(HOME)/rpmbuild/SOURCES
RPMS    := $(HOME)/rpmbuild/RPMS
DEST    := /data
REPOROOT := $(shell git rev-parse --show-toplevel)
NEXUSREPO := dnfLocal

.PHONY: tarball rpm clean

tarball:
	git -C $(REPOROOT) archive --prefix=$(NAME)-$(VERSION)/ HEAD \
	  | gzip > $(SOURCES)/$(NAME)-$(VERSION).tar.gz

rpm: tarball
	rpmbuild -bb $(SPEC)
	find $(RPMS) -name '*.rpm' -exec cp {} $(DEST)/ \;

clean:
	rm -f $(SOURCES)/$(NAME)-$(VERSION).tar.gz
	rm -rf $(HOME)/rpmbuild/BUILD/$(NAME)-$(VERSION)

upload:
	nxtools upload -r ${NEXUSREPO} ${DEST}/${NAME}-${VERSION}-${RELEASE}.x86_64.rpm
