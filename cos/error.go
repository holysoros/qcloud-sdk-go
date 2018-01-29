package cos

import "fmt"

// COS请求失败。
type RequestFailure struct {
	HttpMethod     string
	HttpStatusCode int
	ErrorCode      string
	Message        string
	ResourceURL    string
	RequestId      string
	TraceId        string
}

func (e RequestFailure) Error() string {
	return fmt.Sprintf("%s %s - %d[%s]", e.HttpMethod, e.ResourceURL, e.HttpStatusCode, e.ErrorCode)
}
