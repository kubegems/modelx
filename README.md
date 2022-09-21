# modelx

A model registry to host marchine leaning models.

## Install modex cli

Download binary from [latest release](https://github.com/kubegems/modelx/releases/latest) to your PATH.

Completions provided via `modelx completions zsh|bash|fish|powershell`.

## Quick Start

Init a model and push to modelx server:

```sh
$ modelx init my-model
Modelx model initialized in my-model
$ tree my-model
my-model
├── modelx.yaml
└── README.md
$ cd my-model
# add model files
$ echo "some script" > scripy.sh
$ echo -n "some binary" > binary.dat
# update modelx.yaml

# add modelx registry
$ modelx repo add kubegems http://modelx.kubegems.io
# "login" to modelx registry (Obtain token from your modelx registry admin)
$ modelx login kubegems --token <token>
# push my-model v1 to modelx registry
$ modelx push kubegems:my-model@v1
# list model you just uploaded
$ modelx list kubegems:mymodel
```

## About modelx.yaml

`modelx.yaml` contains model's metadata, a full example is:

```yaml
config:
  inputs: {}
  outputs: {}
description: This is a modelx model
framework: <some framework>
maintainers:
  - maintainer
modelFiles: []
tags:
  - modelx
  - <other>
task: ""
```

## Setup Modelx Server

See helm [charts](charts/modelx)
