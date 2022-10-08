# Setup local modelx

## Prepare S3

Setup a temp S3 server using minio:

```sh
helm install --namespace minio --create-namespace  --repo https://charts.min.io \
--set rootUser=root,rootPassword=password \
--set 'mode=standalone,replicas=1,persistence.enabled=false,buckets[0].name=modelx,buckets[0].policy=none' \
--set service.type=NodePort \
minio minio
```

**Make sure we can access S3 url out of cluster, modelx client pull/push from the address directly.**

## Setup modelx server

Setup modelx from helm:

```sh
export S3_URL="http://$(kubectl get node -o jsonpath='{.items[0].status.addresses[0].address}'):$(kubectl -n minio get svc minio -o jsonpath='{.spec.ports[0].nodePort}')"
echo ${S3_URL} # minio service node port address
helm install --namespace modelx --create-namespace --repo https://charts.kubegems.io/kubegems \
--set "storage.s3.url=${S3_URL},storage.s3.accessKey=root,storage.s3.secretKey=password,storage.s3.bucket=modelx" \
--set service.type=NodePort \
modelx modelx
```

Access modelx server fom node port:

```sh
export MODELX_URL="http://$(kubectl get node -o jsonpath='{.items[0].status.addresses[0].address}'):$(kubectl -n modelx get svc modelx -ojsonpath='{.spec.ports[0].nodePort}')"
echo ${MODELX_URL}  # modelx service node port address
curl ${MODELX_URL}
# {"schemaVersion":0,"manifests":null} # OK, if see this output
```

## Install modelx client

```sh
wget https://github.com/kubegems/modelx/releases/download/v0.1.2/modelx-linux-amd64 -O ~/.local/bin/modelx
chmod +x ~/.local/bin/modelx
```

> follow `modelx completion -h` add completions for your shell.

## Usage

Init a local model repo:

```sh
$ modelx init my-model
Modelx model initialized in my-model

$ cd my-model
# add some file into my-model, empty 
$ ls
data.bin  modelx.yaml
```

Push to modelx repo:

```sh
# add a repo before we can move.
$ modelx repo add local ${MODELX_URL}

$ modelx repo list 
+-------+-----------------------+
| NAME  | URL                   |
+-------+-----------------------+
| local | http://<ip>:<port>    |
+-------+-----------------------+

$ modelx push local/my-model@v1
Pushing to http://<ip>:<port>/library/my-model@v1 
0d9b4fc5 [++++++++++++++++++++++++++++++++++++++++] done
67de5de8 [++++++++++++++++++++++++++++++++++++++++] done
```

List models exists in remote:

```sh
$ modelx list local
+---------+----------+-------------------------------------------+
| PROJECT | NAME     | URL                                       |
+---------+----------+-------------------------------------------+
| library | my-model | http://<ip>:<port>/library/my-model       |
+---------+----------+-------------------------------------------+

$ modelx list local:library/my-model
+---------+----------------------------------------------+------+
| VERSION | URL                                          | SIZE |
+---------+----------------------------------------------+------+
| v1      | http://<ip>:<port>/library/my-model@v1       | 182B |
+---------+----------------------------------------------+------+

$ modelx list local:library/my-model@v1
+-------------+--------+------+------------------+---------------------------+
| FILE        | TYPE   | SIZE | DIGEST           | MODIFIED                  |
+-------------+--------+------+------------------+---------------------------+
| modelx.yaml | config | 174B | 67de5de8ca26a84b | 2022-09-28T15:02:48+08:00 |
| data.bin    | file   | 8B   | 0d9b4fc5ed307b31 | 2022-09-28T15:38:51+08:00 |
+-------------+--------+------+------------------+---------------------------+

$ modelx info local/library/my-model@v1
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

Pull from remote:

```sh
$ modelx pull local/library/my-model@v1 other-dir
Pulling http://<ip>:<port>/library/my-model@v1 into other-dir 
0d9b4fc5 [++++++++++++++++++++++++++++++++++++++++] done
67de5de8 [++++++++++++++++++++++++++++++++++++++++] done

$ ls other-dir 
data.bin  modelx.yaml
```
