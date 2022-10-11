<h1 align="center">ModelX</h1> 

[![.github/workflows/build.yml](https://github.com/kubegems/modelx/actions/workflows/build.yml/badge.svg)](https://github.com/kubegems/modelx/actions/workflows/build.yml)
[![Docker Pulls](https://img.shields.io/docker/pulls/kubegems/modelx.svg?maxAge=604800)](https://hub.docker.com/r/kubegems/modelx)
[![Go Report Card](https://goreportcard.com/badge/github.com/kubegems/modelx)](https://goreportcard.com/report/github.com/kubegems/modelx)
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/kubegems/modelx?logo=go)
[![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/kubegems/modelx?logo=github&sort=semver)](https://github.com/kubegems/kubegems/releases/latest)
[![Artifact HUB](https://img.shields.io/endpoint?url=https://artifacthub.io/badge/repository/kubegems)](https://artifacthub.io/packages/search?repo=kubegems)
![license](https://img.shields.io/github/license/kubegems/kubegems)

A simple, high-performance, scalable model storage service.

<td width="40%" align="center"><img src="https://github.com/kubegems/.github/blob/master/static/image/modelx.jpg?raw=true"></td>

Modelx contains three components

- modelx

  Cli tools for the user side. You can use it to `initialized`、`push`、`pull` models，or management model repository locally.
- modelxd

  modelx repository server. It follows the OCI protocol and provides an http server to receive the modelx cli push models.
- modelxdl

  model deployment tools. It is integrated into the kubegems model deployment flows.

More infomation see: [How Modelx born](docs/how-modelx-born.md)

## Deploy Modelxd Server

### with CLI

#### installation

Download modelxd binary from [latest release](https://github.com/kubegems/modelx/releases/latest) to your PATH.

Determine your version with modelxd --version.

#### Configuration

Show all CLI options with modelxd --help. Common configurations can be seen below.

```
Usage:
  modelxd [flags]

Flags:
      --enable-redirect              enable blob storage redirect
  -h, --help                         help for modelxd
      --listen string                listen address (default ":8080")
      --oidc-issuer string           oidc issuer
      --s3-access-key string         s3 access key
      --s3-bucket string             s3 bucket (default "registry")
      --s3-presign-expire duration   s3 presign expire (default 1h0m0s)
      --s3-region string             s3 region
      --s3-secret-key string         s3 secret key
      --s3-url string                s3 url (default "https://s3.amazonaws.com")
      --tls-ca string                tls ca file
      --tls-cert string              tls cert file
      --tls-key string               tls key file
  -v, --version                      version for modelxd
```

**Using with Amazon S3 or Compatible services like Minio or DigitalOcean.**

Make sure your environment is properly setup to access my-s3-bucket

```
modelxd --listen=:8080 \
  --s3-url=http://<Minio_URL>:<Port> \
  --s3-access-key=<AccessKey> \
  --s3-secret-key=<SecretKey> \
  --s3-bucket=<Bucket> \
  --enable-redirect=true
```

**Using HTTPS**

If both of the following options are provided, the server will listen and serve HTTPS:

- --tls-cert=<crt> - path to tls certificate chain file
- --tls-key=<key> - path to tls key file

HTTPS with Client Certificate Authentication

If the above HTTPS values are provided in addition to below, the server will listen and serve HTTPS and authenticate client requests against the CA certificate:

- --tls-ca-cert=<cacert> - path to tls certificate file

**Using OIDC with KubeGems**

Make sure you KubeGems API is properly access

```
modelxd --listen=:8080 \
  --s3-url=http(s)://<Minio_URL>:<Port> \
  --s3-access-key=<AccessKey> \
  --s3-secret-key=<SecretKey> \
  --s3-bucket=<Bucket> \
  --enable-redirect=true \
  --oidc-issuer=http(s)://<Kubegems_URL>:<Port>
```

### with docker compose

Clone this repository, and run this command below.

```
export ADVERTISED_IP=<Host_IP> //set your host ip
sed -i "s/__ADVERTISED_IP__/${ADVERTISED_IP}/g" docker-compose.yaml
docker compose up -d
```

### with Helm v3

Setup a temp S3 server using minio:

```bash
helm install --namespace minio --create-namespace  --repo https://charts.min.io \
--set rootUser=root,rootPassword=password \
--set 'mode=standalone,replicas=1,persistence.enabled=false,buckets[0].name=modelx,buckets[0].policy=none' \
--set service.type=NodePort \
minio minio
```

> Make sure we can access S3 url out of cluster, modelx client pull/push from the address directly.

#### Setup modelx server

Setup modelx from helm:

```bash
export S3_URL="http://$(kubectl get node -o jsonpath='{.items[0].status.addresses[0].address}'):$(kubectl -n minio get svc minio -o jsonpath='{.spec.ports[0].nodePort}')"
echo ${S3_URL} # minio service node port address
helm install --namespace modelx --create-namespace --repo https://charts.kubegems.io/kubegems \
--set "storage.s3.url=${S3_URL},storage.s3.accessKey=root,storage.s3.secretKey=password,storage.s3.bucket=modelx" \
--set service.type=NodePort \
modelx modelx
```

Access modelx server fom node port:

```bash
export MODELX_URL="http://$(kubectl get node -o jsonpath='{.items[0].status.addresses[0].address}'):$(kubectl -n modelx get svc modelx -ojsonpath='{.spec.ports[0].nodePort}')"
echo ${MODELX_URL}  # modelx service node port address
curl ${MODELX_URL}
# {"schemaVersion":0,"manifests":null} # OK, if see this output
```

For more infomations see [setup](docs/setup.md).

## Modelx CLI

### installation

Download binary from [latest release](https://github.com/kubegems/modelx/releases/latest) to your PATH.

Completions provided via `modelx completions zsh|bash|fish|powershell`.

## Quick Start

First, add and login a model repository

```bash
# Add model repository
$ modelx repo add modelx http://<your_modelxd_url>

# Login repository, if you don't set oidc iusername, press "enter" to skip token authentication.
$ modelx login modelx
Token:
Login successful for modelx
```

Second, Init a model locally 

```bash
$ modelx init class

Modelx model initialized in class

$ tree class
class
├── modelx.yaml
└── README.md

$ cd class
# add model files

$ echo "some script" > scripy.sh
$ echo -n "some binary" > binary.dat
```

Finally, push your models ! 💪🏻

```bash
# add modelx registry

$ modelx push modelx/library/class@v1

Pushing to http://modelx.kubegems.io/library/class@v1
17e682f0 [++++++++++++++++++++++++++++++++++++++++] done
17e682f0 [++++++++++++++++++++++++++++++++++++++++] done
17e682f0 [++++++++++++++++++++++++++++++++++++++++] done
b6f9dd31 [++++++++++++++++++++++++++++++++++++++++] done
test.img [++++++++++++++++++++++++++++++++++++++++] done
4c513e54 [++++++++++++++++++++++++++++++++++++++++] done
```

### Other Commands

**list repository models**

```bash
$ modelx list modelx

+---------+-------+------------------------------------------+
| PROJECT | NAME  | URL                                      |
+---------+-------+------------------------------------------+
| library | class | http://modelx.kubegems.io/library/class  |
+---------+-------+------------------------------------------+
```


**list model versions**

```bash
$ modelx list test/class

+---------+--------------------------------------------+--------+
| VERSION | URL                                        | SIZE   |
+---------+--------------------------------------------+--------+
| v1      | http://modelx.kubegems.io/library/class@v1 | 4.29GB |
| v2      | http://modelx.kubegems.io/library/class@v2 | 4.29GB |
| v3      | http://modelx.kubegems.io/library/class@v3 | 4.29GB |
+---------+--------------------------------------------+--------+
```

**get model infomation**

```
$ modelx info modelx/library/class@v1

config:
  inputs: {}
  outputs: {}
description: This is a modelx model
framework: pytorch
maintainers:
- support@kubegems.io
modelFiles: []
tags:
- modelx
task: ""
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
