#!/bin/bash

# This is PoC for building images without requiring admin capabilities (CAP_SYS_ADMIN)

# Warning: these default values must be aligned with the values provided
# in 'packages/cos-config/cos-config', provide an environment file as the
# second argument if different values are needed.
: "${OEM_LABEL:=COS_OEM}"
: "${RECOVERY_LABEL:=COS_RECOVERY}"

MANIFEST=$1
if [ -z "${MANIFEST}" ]; then
  >&2 echo "Manifest required as parameter, cannot continue"
  exit 1
fi

CONFIG_FILE=$2
if [ -n "${CONFIG_FILE}" ] && [ -e "${CONFIG_FILE}" ]; then
  source "${CONFIG_FILE}"
fi

ARCH=${ARCH:-$(uname -p)}

YQ_VERSION=$( yq -V | cut -d " " -f 3 | cut -d "." -f 1)

if [[ "${YQ_VERSION}" == "3" ]]; then
  YQ_REPO_COMMAND=(yq r "${MANIFEST}" "raw_disk.$ARCH.repo")
  YQ_PACKAGES_COMMAND=(yq r -j "${MANIFEST}")
else
  YQ_REPO_COMMAND=(yq e ".raw_disk.$ARCH.repo" "${MANIFEST}")
  YQ_PACKAGES_COMMAND=(yq e -o=json "$MANIFEST")
fi

FINAL_REPO=${FINAL_REPO:-$("${YQ_REPO_COMMAND[@]}")}

rm -rf ./*.part disk.raw grub_efi.cfg recovery root .luet.yaml oem efi

set -e

# Create a luet config for local repositories
cat << EOF > .luet.yaml
repositories:
  - name: local
    enable: true
    urls:
      - build
    type: disk
    priority: 0
  - name: cOS
    enable: true
    urls:
      - $FINAL_REPO
    type: docker
    priority: 90
EOF

# Create root-tree forOVERY
while IFS=$'\t' read -r name target ; do
  luet install --no-spinner --system-target "$target" -y "$name"
done < <("${YQ_PACKAGES_COMMAND[@]}" | jq -r ".raw_disk.$ARCH.packages[] | [.name, .target] | @tsv")

# Create a 2GB filesystem for RECOVERY including the contents for root (grub config and squasfs container)
truncate -s $((2048*1024*1024)) rootfs.part
mkfs.ext2 -L "${RECOVERY_LABEL}" -d root rootfs.part

# Create the EFI partition FAT16 and include the EFI image and a basic grub.cfg
truncate -s $((20*1024*1024)) efi.part

mkfs.fat -F16 -n COS_GRUB efi.part
mcopy -s -i efi.part efi/EFI ::EFI

# Create the grubenv forcing first boot to be on recovery system
mkdir -p oem
cp root/etc/cos/grubenv_firstboot oem/grubenv

# Create a 64MB filesystem for OEM volume
truncate -s $((64*1024*1024)) oem.part
mkfs.ext2 -L "${OEM_LABEL}" -d oem oem.part

# Create disk image, add 3MB of initial free space to disk, 1MB is for proper alignement, 2MB are for the hybrid legacy boot.
truncate -s $((3*1024*1024)) disk.raw
{
    cat efi.part
    cat oem.part
    cat rootfs.part
} >> disk.raw

# Add an extra MB at the end of the disk for the gpt headers, in fact 34 sectors would be enough, but adding some more does not hurt.
truncate -s "+$((1024*1024))" disk.raw

# Create the partition table in disk.raw (assumes sectors of 512 bytes)
sgdisk -n 1:2048:+2M -c 1:legacy -t 1:EF02 disk.raw
sgdisk -n 2:0:+20M -c 2:UEFI -t 2:EF00 disk.raw
sgdisk -n 3:0:+64M -c 3:oem -t 3:8300 disk.raw
sgdisk -n 4:0:+2048M -c 4:root -t 4:8300 disk.raw

rm -rf ./*.part grub_efi.cfg root .luet.yaml oem efi

# TODO hybrid boot. I did not fully figure out how to avoid grub2-install. Rough steps are:
#   1-> add MBR pointing to sector 2048 (first block of legacy partition) (this is the tricky part grub2-install patches the MBR binary)
#   2-> create core.img with the relevant modules and an embedded grub.cfg equivalent to grub_efi.cfg (grub2-mkimage tool should do the trick)
#   3-> dump grub core image for i386-pc into 2048 sector
