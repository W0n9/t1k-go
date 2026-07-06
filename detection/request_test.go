package detection

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

// truncatedReader 返回部分数据后以 io.ErrUnexpectedEOF 结束，模拟客户端
// 中途断开（Content-Length 不符 / 截断）导致 io.ReadAll 失败的情形。
type truncatedReader struct {
	data []byte
	off  int
}

func (r *truncatedReader) Read(p []byte) (int, error) {
	if r.off >= len(r.data) {
		return 0, io.ErrUnexpectedEOF
	}
	n := copy(p, r.data[r.off:])
	r.off += n
	return n, nil
}

func TestHttpRequestBodyWrapsClientReadError(t *testing.T) {
	req := &http.Request{
		Body:          io.NopCloser(&truncatedReader{data: []byte("hello")}),
		ContentLength: 10, // 故意大于实际可读字节，触发 io.ErrUnexpectedEOF
	}
	_, _, err := (&HttpRequest{req: req}).Body()
	if err == nil {
		t.Fatal("expected error from truncated body, got nil")
	}
	if !strings.Contains(err.Error(), "read request body") {
		t.Errorf("err.Error() = %q, want 'read request body' prefix", err)
	}
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Errorf("errors.Is(err, io.ErrUnexpectedEOF) = false; error chain broken by wrap")
	}
}
