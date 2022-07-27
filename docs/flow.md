# modelx workflow

## 基本概念

|name|description|
|---|---|
| index | 索引 |
| manifest | 描述文件 |
| blob | 数据 |

## 上传

1. 客户端准备本地文件，生成manifest，对每个需要上传的blob文件，计算sha256。
2. 客户端对每个 blob 文件执行：
   1. 检查服务端是否存在对应hash的blob文件，如果存在，则跳过。
   2. 开始上传，从服务器拿到blob上传地址，该地址为s3的put object/multi part upload 的url。
   3. 使用s3 sdk 完成文件上传。
   4. 客户端向服务端报告上传完成。
3. 客户端上传manifest文件
   1. 服务端：解析manifest文件，检查每个blob文件是否存在，如果不存在，则报错。

## 下载

1. 客户端向服务端查询 index 文件，获取 manifest 文件的地址。
2. 客户端向服务端获取 manifest 文件，并解析 manifest 文件，获取每个blob文件的地址。
3. 客户端对每个 blob 文件执行：
   1. 检查本地文件是否存在，如果存在，判断hash是否相等，若相等则认为本地文件于远端相同。
   2. 若不存在或者hash不同，则下载该文件覆盖本地文件。

## 搜索

1. 客户端向服务端查询 index 。
2. 在index中进行搜索。

## 删除

1. 客户端向服务端删除 manifest 。
2. 服务端： 增加垃圾回收机制，每次删除 manifest 时，删除未使用的 blob 文件。
