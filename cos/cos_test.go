package cos

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"time"
)

// 测试环境的安全密钥配置
func readConf() Config {
	return Config{
		AppId:     os.Getenv("qcloud_app_id"),
		SecretId:  os.Getenv("qcloud_secret_id"),
		SecretKey: os.Getenv("qcloud_secret_key"),
	}
}

var client = NewClient(readConf())

func TestParseBucketURL(t *testing.T) {
	testBuckets := []struct {
		url  string
		want Bucket
	}{
		{
			url:  "https://test-123-1203324242.cos.ap-shanghai.myqcloud.com",
			want: Bucket{Name: "test-123", AppId: "1203324242", Region: "ap-shanghai"},
		},
		{
			url:  "https://test-1203324242.cos.ap-beijing.myqcloud.com",
			want: Bucket{Name: "test", AppId: "1203324242", Region: "ap-beijing"},
		},
	}
	for _, tb := range testBuckets {
		b, err := ParseBucketURL(tb.url)
		if err != nil {
			t.Fatal(err)
		}
		if g, e := b.String(), tb.want.String(); g != e {
			t.Errorf("expected %s but got %s", e, g)
		}
	}
}
func TestListBucket(t *testing.T) {
	_, err := client.ListBuckets()
	if err != nil {
		t.Fatal(err)
	}
}

func putBucket(name, region string) (*Bucket, error) {
	return client.PutBucket(name, region, nil)
}

func deleteBucket(name, region string) error {
	return client.DeleteBucket(name, region)
}

func TestBucketOperations(t *testing.T) {
	region := "ap-shanghai" // 上海区域
	name := fmt.Sprintf("test-%d", time.Now().Unix())
	_, err := putBucket(name, region)
	if err != nil {
		t.Fatal(err)
	}
	err = deleteBucket(name, region)
	if err != nil {
		t.Fatal(err)
	}
}

func getObject(resourceURL string) ([]byte, error) {
	rc, _, err := client.GetObject(resourceURL)
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return ioutil.ReadAll(rc)
}

func putObject(resourceURL string, body io.Reader) error {
	return client.PutObject(resourceURL, body, nil)
}

func deleteObject(resourceURL string) error {
	return client.DeleteObject(resourceURL)
}

func TestObjectOperations(t *testing.T) {
	// 临时创建一个测试桶。
	bucket, err := putBucket(fmt.Sprintf("test-%d", time.Now().Unix()), "ap-shanghai")
	if err != nil {
		t.Fatalf("err should be nil but got %v", err)
	}
	defer deleteBucket(bucket.Name, bucket.Region)

	// 创建新的对象。
	body := "HELLO,WORLD!"
	var objURL = bucket.URL() + "/1.txt" //格式化转为绝对URL路径。
	if err := putObject(objURL, strings.NewReader(body)); err != nil {
		t.Fatal(err)
	}
	// 读取内容
	b, err := getObject(objURL)
	if err != nil {
		t.Fatal(err)
	}
	if g, e := string(b), body; g != e {
		t.Fatalf("expected %s; but got %s", e, g)
	}
	if err := deleteObject(objURL); err != nil {
		t.Fatal(err)
	}
}
