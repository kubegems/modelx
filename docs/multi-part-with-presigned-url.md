# multi part upload with presigned url

现在使用的是 http 上传，也就是单线程上传，如果上传的文件比较大，那么上传的时间就会比较长，这个时候就需要使用分片上传了。
分片上传是将一个大文件分割成多个小文件，然后再将这些小文件上传到服务器，这样就可以实现多线程上传，从而提高上传速度，并且可以对每个分片进行重传。

在使用 s3 为存储时，我们使用到了 presigned url，客户端 post 上传到 presigned url 。
而在使用 multi part upload 时，无法使用 presigned url，因为 multi part upload 不是一个操作，而是多个操作的组合。

然而根据 s3 的文档，我们依旧可以为每个分片都生成一个 presigned url，然后客户端使用这些 url 上传分片，最后服务端再将这些分片合并成一个文件。

> presigned url 只能 sign upload part 操作，不能 sign complete 操作，因此客户端需要在上传完所有分片后，再上传 manifest 。

那么流程就变为：

1. PUT blobs/{digest}, 服务端 [CreateMultipartUpload](https://docs.aws.amazon.com/AmazonS3/latest/API/API_CreateMultipartUpload.html),
   并计算(PresignUploadPart)出每个分片的 presigned url，返回给客户端。
2. 客户端根据分片大小，将文件分割成多个分片，然后使用 presigned url 上传分片。
3. 客户端上传完所有 blob 后，PUT manifests/{tag} 时服务端对每个 blob 执行
   [CompleteMultipartUpload](https://docs.aws.amazon.com/AmazonS3/latest/API/API_CompleteMultipartUpload.html),
   服务端需要每个 part 的 etag 才能完成 complete ，因此需要使用
   [ListParts](https://docs.aws.amazon.com/AmazonS3/latest/API/API_ListParts.html)
   获取到以及上传的 part 信息。

由于约定客户端总是在上传完所有 blobs 后才上传 manifest 所以服务端可以在收到 manifest 时直接执行 complete 操作。
