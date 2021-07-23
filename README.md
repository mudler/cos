# containerOS toolkit

containerOS (**cOS**) is a toolkit to build, ship and maintain cloud-init driven Linux derivatives based on container images with a common featureset. 

It is designed to reduce the maintenance surface, with a flexible approach to provide upgrades from container registries. It is cloud-init driven and also designed to be adaptive-first, allowing easily to build changes on top.

[Documentation is available at https://rancher-sandbox.github.io/cos-toolkit-docs/docs](https://rancher-sandbox.github.io/cos-toolkit-docs/docs)

<!-- TOC -->

- [containerOS toolkit](#containeros-toolkit)
    - [In a nutshell](#in-a-nutshell)
    - [Releases](#releases)
    - [Design goals](#design-goals)
        - [Build cOS Locally](#build-cos-locally)
    - [First steps](#first-steps)
        - [Samples](#samples)
        - [cOS development](#cos-development)
    - [License](#license)

<!-- /TOC -->

## In a nutshell

cOS derivatives are built from containers, and completely hosted on image registries. The build process results in a single container image used to deliver regular upgrades in OTA approach. Each derivative built with `cos-toolkit` inherits a default featureset.

cOS supports different release channels, all the final and cache images used are tagged and pushed regularly [to Quay Container Registry](https://quay.io/repository/costoolkit/releases-green) and can be pulled for inspection from the registry as well.

Those are exactly the same images used during upgrades, and can also be used to build Linux derivatives from cOS.

For example, if you want to see locally what's in a openSUSE cOS version , you can:

```bash
$ docker run -ti --rm quay.io/costoolkit/releases-green:cos-system-$VERSION /bin/bash
```

## Releases

cOS-toolkit releases consist on container images that can be used to build derived against and the cos source tree itself.
 
cOS is a manifest which assembles an OS from containers, so if you want to make substantial changes to the layout you can also fork directly cOS.

Currently, the toolkit supports creating derivatives from [OpenSUSE (green), Fedora (blue) and Ubuntu (orange)](https://github.com/rancher-sandbox/cOS-toolkit/tree/master/values), although it's rather simple to add support for other OS families and architecures.

The cOS CI generates ISO and images artifacts used for testing, so you can also try out cOS by downloading the 
ISO [from the Github Actions page](https://github.com/rancher-sandbox/cOS-toolkit/actions/workflows/build.yaml), to the commit you are interested into.

## Design goals

- A Manifest for container-based OS. It contains just the common bits to make a container image bootable and to be upgraded from, with few customization on top
- Immutable-first, but with a flexible layout
- Cloud-init driven
- Based on systemd
- Built and upgraded from containers - It is a [single image OS](https://quay.io/repository/costoolkit/releases-green)!
- OTA updates
- Easy to customize
- Cryptographically verified

### Build cOS Locally

The starting point to use cos-toolkit is to see it in action with our [sample repository](https://github.com/rancher-sandbox/cos-toolkit-sample-repo) or check out our `examples` folder, see also [creating bootable images](https://rancher-sandbox.github.io/cos-toolkit-docs/docs/creating-derivatives/creating_bootable_images/).

The only requirement to build derivatives with `cos-toolkit` is docker installed, see [Development notes](https://rancher-sandbox.github.io/cos-toolkit-docs/docs/development/) for more details on how to build `cos` instead.

## First steps

The [sample repository](https://github.com/rancher-sandbox/cos-toolkit-sample-repo) contains the definitions of a [SampleOS](https://github.com/rancher-sandbox/cos-toolkit-sample-repo/tree/master/packages/sampleOS) boilerplate, which results in an immutable single-image distro and a [simple HTTP service on top](https://github.com/rancher-sandbox/cos-toolkit-sample-repo/tree/master/packages/sampleOSService) that gets started on boot.

To give it a quick shot, it's as simple as cloning the [Github repository](https://github.com/rancher-sandbox/cos-toolkit-sample-repo), and running cos-build:

```bash
$ git clone https://github.com/rancher-sandbox/cos-toolkit-sample-repo
$ cd cos-toolkit-sample-repo
$ source .envrc
$ cos-build
```

This command will build a container image which contains the required dependencies to build the custom OS, and will later be used to build the OS itself. The result will be a set of container images and an ISO which you can boot with your environment of choice.  See [Creating derivatives](https://rancher-sandbox.github.io/cos-toolkit-docs/docs/creating-derivatives/creating_derivatives/) for more details about the process.

If you are looking after only generating a container image that can be used for upgrades from the cOS vanilla images, see [creating bootable images](https://rancher-sandbox.github.io/cos-toolkit-docs/docs/creating-derivatives/creating_bootable_images/) and see also [how to drive upgrades with Fleet](https://rancher-sandbox.github.io/cos-toolkit-docs/docs/tutorials/trigger_upgrades_with_fleet/).


### Samples
- [Sample repository](https://github.com/rancher-sandbox/cos-toolkit-sample-repo)
- [EpinioOS sample repository](https://github.com/rancher-sandbox/epinio-appliance-demo-sample)
- [Use Fleet to upgrade a cOS derivative](https://github.com/rancher-sandbox/cos-fleet-upgrades-sample)
- [Deploy Fleet on a cOS vanilla image](/docs/k3s_and_fleet_on_vanilla_image_example.md)

### cOS development
- [Github project](https://github.com/mudler/cOS/projects/1) for a short-term Roadmap

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
