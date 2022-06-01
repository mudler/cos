
---
title: "Documentation"
linkTitle: "Documentation"
weight: 20
menu:
  main:
    weight: 20
---

## What is cOS?

cOS is a toolkit which allows container images to be bootable in VMs, baremetals, embedded devices, and much more.

cOS allows to create meta-Linux derivatives which are configured throughout cloud-init configuration files and are immutable by default.

cOS and derivatives shares a common feature set, can be upgraded with a A/B mechanism, and upgrades are delivered with standard container registries. 

cOS comes also with vanilla images that can be used to boot directly container images built with the toolkit.

## Why cOS? 

cOS allows to create custom OS versions in your cluster with standard container images with a high degree of customization. It can also be used in its vanilla form - cOS enables then everyone to build their own derivative and access it in various formats. 

To build a bootable image is as simple as running `docker build`.

* **What is it good for?**: Embedded, Cloud, Containers, VM, Baremetals, Servers, IoT, Edge

## Design goals

- A Manifest for container-based OS. It contains just the common bits to make a container image bootable and to be upgraded from, with few customization on top
- Everything is an OCI artifact from the ground-up
- Immutable-first, but with a flexible layout
- Cloud-init driven
- Based on systemd
- Built and upgraded from containers - It is a [single image OS](https://quay.io/repository/costoolkit/releases-teal)!
- A/B updates
- Easy to customize
- Cryptographically verified
- instant switch from different versions
- recovery mechanism with `cOS` vanilla images (or bring your own)

## Mission

The elemental-toolkit project is under the Elemental umbrella.

Elemental-toolkit provides a unique container based approach to define the system lifecycle of an immutable Linux derivative, without any string attached to a specific Linux distribution.

At its heart, Elemental-toolkit is the abstraction layer between Linux distro management and the specific purpose of the OS.

Elemental-toolkit empowers anyone to create derivatives from standard OCI images. Frees whoever wants to create a Linux derivative from handling the heavy bits of packaging and managing entire repositories to propagate upgrades, simplifying the entire process by using container images as base for OS.
At the same time, Elemental-toolkit provides an highly integrated ecosystem which is designed to be container-first, cloud native, and immutable.
Anyone can tweak Elemental-toolkit derivatives from the bottom-up to enable and disable its featureset.

As the Elemental team, the [elemental](https://github.com/rancher/elemental) project is our point of reference.

`Elemental` is a complete derivative built with elemental-toolkit tied with the rancher ecosystem and full cycle node management solution with Kubernetes. 

We are supporting directly and indirectly `elemental` within changes also in the Elemental ecosystem.

`Elemental` is our main show-case, and as the Elemental team we are committed to it. It encompasses several technologies to create a Kubernetes-focused Linux derivative which lifecycle is managed entirely from Kubernetes itself, [Secure Device Onboarding](https://www.intel.it/content/www/it/it/internet-of-things/secure-device-onboard.html) included, and automatic provisioning via cloud-init.
