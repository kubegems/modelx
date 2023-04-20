# modelx idoe

## 概念对应关系

| idoe    | modelx       | -                                                                  |
| ------- | ------------ | ------------------------------------------------------------------ |
| project | whole modelx | 一个 modelx 就是一个 idoe project                                  |
| asset   | repository   | 一个 idoe asset 就是 一个 modelx repository 及其多个版本 manifets  |
| rawdata | blob         | 一个 rawdata 就是一个 blob 文件或者一个 manifest 文件 （可以多级） |

目前 idoe 中最小单元为一个 asset，对应 modelx 中一个 repository(全部版本)

## modelx 改动

- login 时依旧使用 kubegems token，modelx 验证成功后查询本地数据库该用户是否已有 idoe 私钥，若没有则创建。
- push manifest 时验证 idoe asset 是否存在，否则创建。
  - 这里涉及到 asset 所有者问题：
    1. 可以暂时只接用 modelx 作为所有者，后续上传下载均由这个所有者授权。
    1. 需要用户手动操作，必须先创建仓库，创建者为所有者。需要增加 modelx 命令，`modelx repo create`
- 一个模型存储于一个 asset，assets 中维护多个版本的 manifest 文件
- list versions 实质上变更为 get project attrs。
- 下载 blob 使用的 url 变更为服务端使用仓库对应所有者的私钥生成 access grant，并在客户端使用临时账户和 access grant 下载

部署时下载：部署时下载依旧填写 token ，token 为仓库所有者的 grant

现有存储目录结构：

```txt
/index
/{repository}/index
/{repository}/manifests/v1
/{repository}/manifests/v2
/{repository}/blobs/sha256:{32}
/{repository}/blobs/sha256:{32}
/{repository}/blobs/sha256:{32}
```

## Q&A

Q：维护用户名和私钥的对应关系是交给 kubegems 做还是交给 modelx 做？

- 交给 modelx 做，kubegems 可以保持现状不变，modelx 在验证 token 成功后，查询或立即为用户生成私钥并存储至数据库。

Q: 上传/下载 时生成的 access grant 要设置白名单吗？

- 否，则所有获得 access grant 的人都可以使用；

Q: 关于 vault project asset 所有者,所有者是 modelx 还是具体的用户？

- 若所有者是 modelx，则所有资产都认为是 modelx 的，上传下载都使用 modelx 授权。如果 modelx 服务账户私钥遗失则所有数据遗失。
- 若所有者是 具体用户，则需要增加 “创建空仓库” 动作。如果平台遗失用户私钥则数据遗失。
