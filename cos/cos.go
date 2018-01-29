package cos

import (
	"crypto/hmac"
	"crypto/sha1"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/antchfx/xmlquery"
)

// App密钥配置。
type Config struct {
	AppId     string
	SecretId  string
	SecretKey string
}

// 存储桶对象。
type Bucket struct {
	AppId, Name, Region string
}

// Address返回桶的访问域名地址(XML API)，URL不包含`/`路径。
func (b *Bucket) Address() string {
	return fmt.Sprintf("https://%s-%s.cos.%s.myqcloud.com", b.Name, b.AppId, b.Region)
}

// 客户端，执行COS操作请求。
type Client struct {
	ci *http.Client
	cf Config
}

// NewClient创建新的Client对象。
func NewClient(conf Config) *Client {
	return &Client{
		ci: &http.Client{},
		cf: conf,
	}
}

func hashHMACSHA1(key string, data []byte) string {
	h := hmac.New(sha1.New, []byte(key))
	h.Write(data)
	return fmt.Sprintf("%x", h.Sum(nil))
}

func hashSHA1(data []byte) string {
	h := sha1.New()
	h.Write(data)
	return fmt.Sprintf("%x", h.Sum(nil))
}

// https://cloud.tencent.com/document/product/436/7778
func (c *Client) signRequest(req *http.Request) string {
	type pair struct {
		key, value string
	}
	// The HTTP URL query list.
	var sortedQuerys []pair
	for key, value := range req.URL.Query() {
		sortedQuerys = append(sortedQuerys, pair{strings.ToLower(key), strings.ToLower(value[0])})
	}
	sort.Slice(sortedQuerys, func(i, j int) bool { return sortedQuerys[i].key < sortedQuerys[j].key })
	// The HTTP Request Header list.
	var sortedHeaders []pair
	for key, value := range req.Header {
		sortedHeaders = append(sortedHeaders, pair{strings.ToLower(key), strings.ToLower(url.QueryEscape(value[0]))})
	}
	sort.Slice(sortedHeaders, func(i, j int) bool { return sortedHeaders[i].key < sortedHeaders[j].key })

	var (
		reqPayloadStr            string
		headerList, paramList, s []string
	)

	reqPayloadStr = strings.ToLower(req.Method) + "\n"
	reqPayloadStr += req.URL.Path + "\n"

	for _, p := range sortedQuerys {
		s = append(s, p.key+"="+p.value)
		paramList = append(paramList, p.key)
	}
	reqPayloadStr += strings.Join(s, "&") + "\n"

	s = s[:0]
	for _, p := range sortedHeaders {
		s = append(s, p.key+"="+p.value)
		headerList = append(headerList, p.key)
	}
	reqPayloadStr += strings.Join(s, "&") + "\n"

	// Sign
	now := time.Now()
	signTime := fmt.Sprintf("%d;%d", now.Unix(), now.Add(30*time.Second).Unix())
	payloadStr := "sha1\n" + signTime + "\n" + hashSHA1([]byte(reqPayloadStr)) + "\n"
	signKey := hashHMACSHA1(c.cf.SecretKey, []byte(signTime))
	signature := hashHMACSHA1(signKey, []byte(payloadStr))

	m := map[string]string{
		"q-sign-algorithm": "sha1",
		"q-ak":             c.cf.SecretId,
		"q-sign-time":      signTime,
		"q-key-time":       signTime,
		"q-header-list":    strings.Join(headerList, ";"),
		"q-url-param-list": strings.Join(paramList, ";"),
		"q-signature":      signature,
	}
	s = s[:0]
	for key, value := range m {
		s = append(s, key+"="+value)
	}
	return strings.Join(s, "&")
}

func createRequest(method, urlStr string, header map[string]string, body io.Reader) (req *http.Request, err error) {
	req, err = http.NewRequest(method, urlStr, body)
	if err != nil {
		return
	}
	for key, value := range header {
		req.Header.Set(key, value)
	}
	return
}

// https://cloud.tencent.com/document/product/436/7730
func requestFailure(method string, resp *http.Response) error {
	doc, err := xmlquery.Parse(resp.Body)
	if err != nil {
		return err
	}
	root := xmlquery.FindOne(doc, "Error")
	return RequestFailure{
		HttpMethod:     method,
		HttpStatusCode: resp.StatusCode,
		ErrorCode:      xmlquery.FindOne(root, "Code").InnerText(),
		Message:        xmlquery.FindOne(root, "Message").InnerText(),
		ResourceURL:    xmlquery.FindOne(root, "Resource").InnerText(),
		RequestId:      xmlquery.FindOne(root, "RequestId").InnerText(),
		TraceId:        xmlquery.FindOne(root, "TraceId").InnerText(),
	}
}

func (c *Client) send(req *http.Request) (*http.Response, error) {
	if req.URL.Path == "" {
		req.URL.Path = "/"
	}
	req.Header.Set("Host", req.URL.Host)
	req.Header.Set("Authorization", c.signRequest(req))

	return c.ci.Do(req)
}

// ListBuckets返回所有存储空间列表。
// https://cloud.tencent.com/document/product/436/8291
func (c *Client) ListBuckets() ([]*Bucket, error) {
	const (
		endpoint = "https://service.cos.myqcloud.com/"
	)
	req, _ := createRequest("GET", endpoint, nil, nil)
	resp, err := c.send(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, requestFailure(req.Method, resp)
	}
	doc, err := xmlquery.Parse(resp.Body)
	if err != nil {
		return nil, err
	}
	var buckets []*Bucket
	for _, elem := range xmlquery.Find(doc, "//Buckets/Bucket") {
		name := xmlquery.FindOne(elem, "Name").InnerText()
		a := strings.Split(name, "-")
		buckets = append(buckets, &Bucket{
			Name:   a[0],
			AppId:  a[1],
			Region: xmlquery.FindOne(elem, "Location").InnerText(),
		})
	}
	return buckets, nil
}

// PutBucket创建一个新的存储桶(Bucket)。
// header参数自定义HTTP请求的标头内容，可以配置Bucket的访问权限。
// https://cloud.tencent.com/document/product/436/7738
func (c *Client) PutBucket(name, region string, header map[string]string) (*Bucket, error) {
	endpoint := fmt.Sprintf("https://%s-%s.cos.%s.myqcloud.com/", name, c.cf.AppId, region)
	req, _ := createRequest("PUT", endpoint, header, nil)
	resp, err := c.send(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == 200 {
		return &Bucket{
			Name:   name,
			AppId:  c.cf.AppId,
			Region: region,
		}, nil
	}
	return nil, requestFailure(req.Method, resp)
}

// DeleteBucket删除一个指定的存储桶(Bucket)。
func (c *Client) DeleteBucket(name, region string) error {
	endpoint := fmt.Sprintf("https://%s-%s.cos.%s.myqcloud.com/", name, c.cf.AppId, region)
	req, _ := createRequest("DELETE", endpoint, nil, nil)
	resp, err := c.send(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case 200, 204:
		return nil
	}
	return requestFailure(req.Method, resp)
}

// PutObject上传文件到指定的URL。
// header参数自定义HTTP请求的标头内容，可以配置Object的访问权限。
// https://cloud.tencent.com/document/product/436/7749
func (c *Client) PutObject(url string, body io.Reader, header map[string]string) error {
	req, err := createRequest("PUT", url, header, body)
	if err != nil {
		return err
	}
	resp, err := c.send(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == 200 {
		return nil
	}
	return requestFailure(req.Method, resp)
}

// GetObject返回指定文件的响应内容。
// https://cloud.tencent.com/document/product/436/7753
func (c *Client) GetObject(url string) (io.ReadCloser, error) {
	req, err := createRequest("GET", url, nil, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.send(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == 200 {
		return resp.Body, nil
	}
	defer resp.Body.Close()
	return nil, requestFailure(req.Method, resp)
}

// DeleteObject删除指定的文件。
// https://cloud.tencent.com/document/product/436/7743
func (c *Client) DeleteObject(url string) error {
	req, err := createRequest("DELETE", url, nil, nil)
	if err != nil {
		return err
	}
	resp, err := c.send(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case 200, 204:
		// 请求中删除一个不存在的 Object，仍然认为是成功的，返回 204 No Content。
		return nil
	}
	return requestFailure(req.Method, resp)
}
