#
# cOS-toolkit Makefile
#
#

#----------------------- global variables -----------------------
#
# Path to luet binary
#
export LUET?=$(shell which luet 2> /dev/null)
ifeq ("$(LUET)","")
LUET="/usr/bin/luet"
endif

#
# Path to jq binary
#
export JQ?=$(shell which jq 2> /dev/null)
ifeq ("$(JQ)","")
JQ="/usr/bin/jq"
endif

#
# Path to yq binary
#
export YQ?=$(shell which yq 2> /dev/null)
ifeq ("$(YQ)","")
YQ="/usr/bin/yq"
endif

#
# Path to luet-mtree binary
#
export MTREE?=$(shell which luet-mtree 2> /dev/null)
ifeq ("$(MTREE)","")
MTREE="/usr/bin/luet-mtree"
endif

#
# Path to luet-cosign binary
#
export COSIGN?=$(shell which luet-cosign 2> /dev/null)
ifeq ("$(COSIGN)","")
COSIGN="/usr/bin/luet-cosign"
endif

export ELEMENTAL?=$(shell which elemental 2> /dev/null)
ifeq ("$(ELEMENTAL)","")
ELEMENTAL="/usr/bin/elemental"
endif

#
# Location of package tree
#
TREE?=$(ROOT_DIR)/packages

#
# Directory of Makefile
#
export ROOT_DIR:=$(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))

#
# OS flavor to build
#
FLAVOR?=green

#
# Arch to build for
#

ARCH?=x86_64

#
# Output for "make publish-repo" and base for "make iso"
#
ifneq ($(strip $(ARCH)), x86_64)
	FINAL_REPO?=quay.io/costoolkit/releases-$(FLAVOR)-$(ARCH)
else
	FINAL_REPO?=quay.io/costoolkit/releases-$(FLAVOR)
endif

#
# folder for build artefacts
#
DESTINATION?=$(ROOT_DIR)/build

#
# yaml specification of build targets
#
export MANIFEST?=$(ROOT_DIR)/manifest.yaml

#
# cos config environment file
#
export COS_CONFIG?=$(ROOT_DIR)/packages/cos-config/cos-config

#
# Packer target to build
#
PACKER_TARGET?=virtualbox-iso.cos

#
# Used by .iso, .test, and .run
#
ISO?=$(shell ls $(ROOT_DIR)/*.iso 2> /dev/null)


#----------------------- end global variables -----------------------


#----------------------- default target -----------------------

all: deps build

#----------------------- includes -----------------------

include make/Makefile.build
include make/Makefile.iso
include make/Makefile.run
include make/Makefile.test
include make/Makefile.images

#----------------------- targets -----------------------

deps: $(LUET) $(YQ) $(JQ) $(MAKEISO) $(MTREE) $(COSIGN) $(ELEMENTAL)

deps_ci: $(LUET) ci_deps

as_root:
ifneq ($(shell id -u), 0)
	@echo "Please run 'make $@' as root"
	@exit 1
endif

luet: as_root $(LUET)

add_local_repo: luet
	$(LUET) repo add local -y --url $(DESTINATION) --type disk --priority 1 --description local-repo

$(LUET):
ifneq ($(shell id -u), 0)
	@echo "'$@' is missing and you must be root to install it."
	@exit 1
else
	$(ROOT_DIR)/scripts/get_luet.sh
endif

$(YQ):
ifneq ($(shell id -u), 0)
	@echo "'$@' is missing and you must be root to install it."
	@exit 1
else
	$(LUET) install -y toolchain/yq
endif

$(JQ):
ifneq ($(shell id -u), 0)
	@echo "'$@' is missing and you must be root to install it."
	@exit 1
else
	$(LUET) install -y utils/jq
endif

$(MTREE):
ifneq ($(shell id -u), 0)
	@echo "'$@' is missing and you must be root to install it."
	@exit 1
else
	$(LUET) install -y toolchain/luet-mtree
endif

$(COSIGN):
ifneq ($(shell id -u), 0)
	@echo "'$@' is missing and you must be root to install it."
	@exit 1
else
	$(LUET) install -y meta/cos-verify
endif

$(ELEMENTAL):
ifneq ($(shell id -u), 0)
	@echo "'$@' is missing and you must be root to install it."
	@exit 1
else
	$(LUET) install -y toolchain/elemental-cli
endif

ci_deps: as_root
	$(LUET) install -y toolchain/elemental-cli meta/cos-verify toolchain/luet-mtree utils/jq toolchain/yq

clean: clean_build clean_iso clean_run clean_test
	rm -rf $(ROOT_DIR)/*.sha256
