# From scratch image example

Build:

```bash
$ docker build -t scratch .
```

Push to some registry and upgrade your `cOS` vm to it:

```bash
cos-vm $ elemental upgrade --docker-image --no-verify $IMAGE
```