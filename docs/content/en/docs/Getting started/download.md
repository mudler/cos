
---
title: "Download"
linkTitle: "Download"
weight: 1
date: 2017-01-05
description: >
  How to get Elemental vanilla assets: ISOs, Cloud Images, Vagrant boxes, ....
---

Elemental-toolkit releases consist on container images that can be used to build derived against and the cos source tree itself.

Elemental supports different release channels, all the final and cache images used are tagged and pushed regularly [to Quay Container Registry](https://quay.io/repository/costoolkit/releases-teal) and can be pulled for inspection from the registry as well.

Those are exactly the same images used during upgrades, and can also be used to build Linux derivatives from Elemental.

For example, if you want to see locally what's in a openSUSE Elemental version, you can:

```bash
$ docker run -ti --rm quay.io/costoolkit/releases-teal:cos-system-$VERSION /bin/bash
```
 
## Download Elemental

You can also try out Elemental from the vanilla images and use it to experiment locally or either bootstrap a derivative: those are minimal system with a small package set in order to boot and deploy a container. 

Latest Elemental-toolkit releases assets (ISOs, Raw disks, Cloud images) can be found on [Github](https://github.com/rancher/elemental-toolkit/releases/), check [Booting](../booting) for an explanation of each asset type and how to use it.

Elemental can run in: VMs, baremetals and Cloud - the default login username/password is `root/cos`.

### Install

To install run `elemental install <device>` to start the installation process. Remove the ISO/medium and reboot.

_Note_: `elemental install` supports other options as well. Run `elemental install --help` to see a complete help.

## Releases

Elemental has 4 variants:

- [teal](https://quay.io/repository/costoolkit/releases-teal): sle-micro-rancher based one, shipping packages from Sle Micro 5.2.
- [green](https://quay.io/repository/costoolkit/releases-green): openSUSE based one, shipping packages from OpenSUSE Leap 15.3 repositories.
- [blue](https://quay.io/repository/costoolkit/releases-blue): Fedora based one, shipping packages from Fedora 33 repositories
- [orange](https://quay.io/repository/costoolkit/releases-orange): Ubuntu based one, shipping packages form Ubuntu 20.10 repositories

We currently support and test only the **teal** variant.

## Published AMI images

We publish AMI images for each release, you can find them into ec2 for example with:

```bash
aws ec2 describe-images --filters 'Name=description,Values=cOS*'
```

The list of all the published AMI is released as part of the [releases](https://github.com/rancher/elemental-toolkit/releases) assets with the `ami_id.txt.tar.xz` file, e.g. [v0.6.7](https://github.com/rancher/elemental-toolkit/releases/download/v0.6.7/ami_id.txt.tar.xz)

The AMI Owner ID is `053594193760`.

## What to do next?

Check out [the customization section](../../customizing) to customize `Elemental` or [the example section](../../examples) for some already prepared recipe examples.
