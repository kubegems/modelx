# modelx API

## 基本概念

| name     | description                  |
| -------- | ---------------------------- |
| index    | 索引，用于寻找所有 manifest  |
| manifest | 描述文件，用于寻找 blob 文件 |
| blob     | 数据文件，实际存储数据的类型 |

## endpoints

| method | path                                 | description              |
| ------ | ------------------------------------ | ------------------------ |
| GET    | /                                    | 获取全局索引             |
| GET    | /{repository}/{name}/index           | 获取索引                 |
| DELETE | /{repository}/{name}/index           | 删除索引以及所有版本数据 |
| GET    | /{repository}/{name}/manifests/{tag} | 获取特定版本描述文件     |
| DELETE | /{repository}/{name}/manifests/{tag} | 删除特定版本描述文件     |
| HEAD   | /{repository}/{name}/blobs/{digest}  | 判断数据文件是否存在     |
| GET    | /{repository}/{name}/blobs/{digest}  | 获取特定版本数据文件     |
| PUT    | /{repository}/{name}/blobs/{digest}  | 上传特定版本数据文件     |
| POST   | /{repository}/{name}/garbage-collect | 触发垃圾收集             |

## endpoints (redirect)

| method | path                                                   | description  |
| ------ | ------------------------------------------------------ | ------------ |
| GET    | /{repository}/{name}/blobs/{digest}/locations/upload   | 获取上传位置 |
| GET    | /{repository}/{name}/blobs/{digest}/locations/download | 获取下载位置 |

## 负载转移

服务端的主要功能仅有两个，一是数据存储，二是索引更新。
对于数据存储，一般来说，会选择将数据实际存入对象存储。用户需要将数据从 客户端->服务端->对象存储。
对于大文件来说，这无疑增加了服务端的压力，服务端本可以不用承载这些流量。
于是考虑通过服务端进行协调，让客户端可以直接将数据发送到对象存储服务器，但同时也保留客户端直接推送数据到服务端并由服务端转存的能力。

客户端携带全部数据向 blob 上传接口执行上传，服务端接受的请求后，判断服务端是否支持负载转移，若不支持，则接收客户端上传数据并由服务端写入后端存储。
若支持，则响应 302 Found，并在响应 header 中携带 Location: url , 其中 url 是实际需要上传的地址。

将大流量请求分离到独立的服务中有助于保持核心服务的稳定性。

## 约定

大致有几个约定，这些约定有利于使用简洁的 API 即可完成功能：

- 客户端在上传 manifest 之前，确保已经上传了所有 blob。上传 manifest 意味着客户端承诺上传了所有的部分。其他客户端可以发现并能够下载完整的 pack。

## 上传

1. 客户端准备本地文件，对每个需要上传的 blob 文件，计算 sha256。生成 manifest。
2. 客户端对每个 blob 文件执行：
   1. 检查服务端是否存在对应 hash 的 blob 文件，如果存在，则跳过。
   2. 否则开始上传，服务端可能存在重定向时遵循重定向。
3. 客户端上传 manifest 文件
   1. 服务端：解析 manifest 文件，检查每个 blob 文件是否存在，如果不存在，则报错。

## 下载

1. 客户端向服务端查询 index 文件，获取 manifest 文件的地址。
2. 客户端向服务端获取 manifest 文件，并解析 manifest 文件，获取每个 blob 文件的地址。
3. 客户端对每个 blob 文件执行：
   1. 检查本地文件是否存在，如果存在，判断 hash 是否相等，若相等则认为本地文件于远端相同。
   2. 若不存在或者 hash 不同，则下载该文件覆盖本地文件。

## 搜索

1. 客户端向服务端查询 index 。
2. 在 index 中进行搜索。

## 删除

1. 客户端向服务端删除 manifest 。
2. 服务端： 增加垃圾回收机制，每次删除 manifest 时，删除未使用的 blob 文件。
