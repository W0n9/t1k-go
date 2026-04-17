package t1k

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/chaitin/t1k-go/detection"
	"github.com/chaitin/t1k-go/t1k"

	"github.com/chaitin/t1k-go/misc"
)

func TestWriteDetectRequest(t *testing.T) {
	sReq := "POST /form.php?id=3 HTTP/1.1\r\n" +
		"Host: a.com\r\n" +
		"Content-Length: 40\r\n" +
		"Content-Type: application/json\r\n\r\n" +
		"{\"name\": \"youcai\", \"password\": \"******\"}"
	req, err := http.ReadRequest(bufio.NewReader(bytes.NewBuffer([]byte(sReq))))
	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	err = writeDetectionRequest(&buf, detection.MakeHttpRequest(req))
	if err != nil {
		t.Fatal(err)
	}
	misc.PrintHex(buf.Bytes())
}

func TestWriteDetectRequestAndResponse(t *testing.T) {
	sReq := "POST /form.php HTTP/1.1\r\n" +
		"Host: a.com\r\n" +
		"Content-Length: 40\r\n" +
		"Content-Type: application/json\r\n\r\n" +
		"{\"name\": \"youcai\", \"password\": \"******\"}"
	req, err := http.ReadRequest(bufio.NewReader(bytes.NewBuffer([]byte(sReq))))
	if err != nil {
		t.Fatal(err)
	}

	sRsp := "HTTP/1.1 200 OK\r\n" +
		"Content-Length: 29\r\n" +
		"Content-Type: application/json\r\n\r\n" +
		"{\"err\": \"password-incorrect\"}"
	rsp, err := http.ReadResponse(bufio.NewReader(bytes.NewBuffer([]byte(sRsp))), req)
	if err != nil {
		t.Fatal(err)
	}

	dc := detection.New()
	dreq := detection.MakeHttpRequestInCtx(req, dc)
	drsp := detection.MakeHttpResponseInCtx(rsp, dc)

	var buf bytes.Buffer
	err = writeDetectionRequest(&buf, dreq)
	if err != nil {
		t.Fatal(err)
	}
	dc.T1KContext = []byte("sample-t1k-context")
	err = writeDetectionResponse(&buf, drsp)
	if err != nil {
		t.Fatal(err)
	}

	misc.PrintHex(buf.Bytes())
}

func TestReadDetectResult(t *testing.T) {
	data := []byte{
		0x41, 0x01, 0x00, 0x00, 0x00, 0x2e, 0xa5, 0x4d,
		0x00, 0x00, 0x00, 0x7b, 0x22, 0x65, 0x76, 0x65,
		0x6e, 0x74, 0x5f, 0x69, 0x64, 0x22, 0x3a, 0x22,
		0x38, 0x36, 0x63, 0x38, 0x33, 0x62, 0x35, 0x61,
		0x33, 0x66, 0x62, 0x32, 0x34, 0x31, 0x61, 0x32,
		0x38, 0x39, 0x39, 0x37, 0x64, 0x39, 0x34, 0x65,
		0x34, 0x62, 0x32, 0x39, 0x63, 0x61, 0x65, 0x33,
		0x22, 0x2c, 0x22, 0x72, 0x65, 0x71, 0x75, 0x65,
		0x73, 0x74, 0x5f, 0x68, 0x69, 0x74, 0x5f, 0x77,
		0x68, 0x69, 0x74, 0x65, 0x6c, 0x69, 0x73, 0x74,
		0x22, 0x3a, 0x66, 0x61, 0x6c, 0x73, 0x65, 0x7d,
	}
	rBuf := bytes.NewBuffer(data)
	ret, err := readDetectionResult(rBuf)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("%v\n", ret)
}

func TestWriteDetectionRequestBodyNotEmpty(t *testing.T) {
	sReq := "POST /form.php?id=3 HTTP/1.1\r\n" +
		"Host: a.com\r\n" +
		"Content-Length: 40\r\n" +
		"Content-Type: application/json\r\n\r\n" +
		"{\"name\": \"youcai\", \"password\": \"******\"}"
	req, err := http.ReadRequest(bufio.NewReader(bytes.NewBuffer([]byte(sReq))))
	bodySize := len([]byte("{\"name\": \"youcai\", \"password\": \"******\"}"))
	if err != nil {
		t.Fatal(err)
	}

	dc := detection.New()
	detReq := detection.MakeHttpRequestInCtx(req, dc)

	var buf bytes.Buffer
	err = writeDetectionRequest(&buf, detReq)
	if err != nil {
		t.Fatal(err)
	}

	// Attempt to read the body again
	bodyRc := req.Body
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(bodyRc)
	if err != nil {
		t.Fatal(err)
	}

	if int64(len(body)) != int64(bodySize) {
		t.Errorf("Expected body size %d, got %d", bodySize, len(body))
	}

	expectedBody := []byte("{\"name\": \"youcai\", \"password\": \"******\"}")
	if !bytes.Equal(body, expectedBody) {
		t.Errorf("Expected body %s, got %s", expectedBody, body)
	}
}

func TestReadDetectResultBotFields(t *testing.T) {
	// Build a fake engine response with TAG_BOT_QUERY and TAG_BOT_BODY sections.
	var buf bytes.Buffer
	sections := []struct {
		tag  t1k.Tag
		data []byte
	}{
		{t1k.TAG_HEADER, []byte{'.'}},
		{t1k.TAG_BOT_QUERY, []byte("bot-query-payload")},
		{t1k.TAG_BOT_BODY, []byte("bot-body-payload")},
	}
	for i, s := range sections {
		tag := s.tag
		if i == 0 {
			tag |= t1k.MASK_FIRST
		}
		if i == len(sections)-1 {
			tag |= t1k.MASK_LAST
		}
		sec := t1k.MakeSimpleSection(tag, s.data)
		if err := t1k.WriteSection(sec, &buf); err != nil {
			t.Fatal(err)
		}
	}

	result, err := readDetectionResult(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if !result.BotDetected() {
		t.Error("expected BotDetected true")
	}
	if string(result.BotQuery) != "bot-query-payload" {
		t.Errorf("BotQuery wrong: %q", result.BotQuery)
	}
	if string(result.BotBody) != "bot-body-payload" {
		t.Errorf("BotBody wrong: %q", result.BotBody)
	}
}
