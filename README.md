# [![Packages](https://cos-toolkit.herokuapp.com/badge/cos-toolkit-green "List of packages")](https://cos-toolkit.herokuapp.com/cos-toolkit-green) [![Build cOS master](https://github.com/rancher-sandbox/cOS-toolkit/actions/workflows/build-master-x86_64.yaml/badge.svg)](https://github.com/rancher-sandbox/cOS-toolkit/actions/workflows/build-master-x86_64.yaml) containerOS toolkit

containerOS (**cOS**) is a toolkit to build, ship and maintain cloud-init driven Linux derivatives based on container images with a common featureset - allows container images to be bootable in VMs, baremetals, embedded devices, and much more.

It is designed to reduce the maintenance surface, with a flexible approach to provide upgrades from container registries. It is cloud-init driven and also designed to be adaptive-first, allowing easily to build changes on top.

Documentation is available at [https://rancher-sandbox.github.io/cos-toolkit-docs/docs](https://rancher-sandbox.github.io/cos-toolkit-docs/docs)

## Design goals

- A Manifest for container-based OS. It contains just the common bits to make a container image bootable and to be upgraded from, with few customization on top
- Immutable-first, but with a flexible layout
- Cloud-init driven
- Based on systemd
- Built and upgraded from containers - It is a [single image OS](https://quay.io/repository/costoolkit/releases-green)!
- OTA updates
- Easy to customize
- Cryptographically verified

### Quick start

Check out our [getting-started](https://rancher-sandbox.github.io/cos-toolkit-docs/docs/getting-started/) section in the documentation.

## License

Copyright (c) 2020-2021 [SUSE, LLC](http://suse.com)

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

[http://www.apache.org/licenses/LICENSE-2.0](http://www.apache.org/licenses/LICENSE-2.0)

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
