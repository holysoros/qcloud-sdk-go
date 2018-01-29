腾讯云对象存储(COS)
===

[![GoDoc](https://godoc.org/github.com/zhengchun/qcloud-sdk-go/cos?status.svg)](https://godoc.org/github.com/zhengchun/qcloud-sdk-go/cos)

API Docs : [https://cloud.tencent.com/document/product/436](https://cloud.tencent.com/document/product/436)

支持的API接口
===

### Service API

   - [Get Service](https://cloud.tencent.com/document/product/436/8291)

### Bucket API

   - [Put Bucket](https://cloud.tencent.com/document/product/436/7738)

   - [Delete Bucket](https://cloud.tencent.com/document/product/436/7732)

### Object API

   - [Get Object](https://cloud.tencent.com/document/product/436/7753)

   - [Put Object](https://cloud.tencent.com/document/product/436/7749)

   - [Delete Object](https://cloud.tencent.com/document/product/436/7743)

快速入门
===

```go
// 安全密钥配置
conf := cos.Config{
    AppId:     "",
    SecretId:  "",
    SecretKey: "",
}
client := cos.NewClient(conf)
// 创建新桶
if _, err := client.PutBucket("test", "ap-shanghai", nil); err != nil {
    panic(err)
}
// 上传文件到桶。
// 文件访问的URL地址。
// https://<BucketName-APPID>.cos.<Region>.myqcloud.com/ObjectName
urlStr := "https://test-123333.cos.ap-shanghai.myqcloud.com/hello.txt"
if err := client.PutObject(urlStr, strings.NewReader("Hello,world!"), nil); err != nil {
    panic(err)
}
```

#### 如何上传文件到存储桶

   - 先将路径格式为URL的绝对格式：`https://<BucketName-APPID>.cos.<Region>.myqcloud.com/ObjectName`。
   - 调用`PutObject()`函数。

