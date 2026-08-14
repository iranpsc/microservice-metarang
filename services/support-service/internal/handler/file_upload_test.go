package handler

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestUploadTicketAttachment_StorageNotConfigured(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	_, err := uploadTicketAttachment(req, "", "http://app")
	if err == nil || !strings.Contains(err.Error(), "storage service not configured") {
		t.Fatalf("err=%v", err)
	}
}

func TestUploadTicketAttachment_MissingFileReturnsEmpty(t *testing.T) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	_ = w.WriteField("title", "t")
	_ = w.Close()
	req := httptest.NewRequest(http.MethodPost, "/", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	url, err := uploadTicketAttachment(req, "storage:9000", "http://app")
	if err != nil || url != "" {
		t.Fatalf("url=%q err=%v", url, err)
	}
}

func TestUploadTicketAttachment_InvalidType(t *testing.T) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, err := w.CreateFormFile("attachment", "malware.exe")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = part.Write([]byte("MZ"))
	_ = w.Close()
	req := httptest.NewRequest(http.MethodPost, "/", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	_, err = uploadTicketAttachment(req, "storage:9000", "http://app")
	if err == nil || !strings.Contains(err.Error(), "invalid attachment type") {
		t.Fatalf("err=%v", err)
	}
}

func TestUploadTicketAttachment_ExceedsSize(t *testing.T) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, err := w.CreateFormFile("attachment", "big.png")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = part.Write(bytes.Repeat([]byte("x"), maxTicketAttachmentSize+1))
	_ = w.Close()
	req := httptest.NewRequest(http.MethodPost, "/", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	_, err = uploadTicketAttachment(req, "storage:9000", "http://app")
	if err == nil || !strings.Contains(err.Error(), "5MB") {
		t.Fatalf("err=%v", err)
	}
}

func TestUploadBytesToStorage_StubSuccessAndFailures(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/upload" {
			t.Fatalf("path=%s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"path": "uploads/tickets/", "name": "a.png", "done": 1, "success": true,
		})
	}))
	defer srv.Close()
	host := strings.TrimPrefix(srv.URL, "http://")

	got, err := uploadBytesToStorage(host, "http://app.test", "tickets", "a.png", "image/png", []byte("hi"))
	if err != nil || got != "http://app.test/uploads/tickets/a.png" {
		t.Fatalf("got=%q err=%v", got, err)
	}

	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("nope"))
	}))
	defer bad.Close()
	_, err = uploadBytesToStorage(strings.TrimPrefix(bad.URL, "http://"), "http://app", "tickets", "a.png", "image/png", []byte("hi"))
	if err == nil || !strings.Contains(err.Error(), "status 500") {
		t.Fatalf("err=%v", err)
	}

	invalid := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{`))
	}))
	defer invalid.Close()
	_, err = uploadBytesToStorage(strings.TrimPrefix(invalid.URL, "http://"), "http://app", "tickets", "a.png", "image/png", []byte("hi"))
	if err == nil || !strings.Contains(err.Error(), "invalid storage response") {
		t.Fatalf("err=%v", err)
	}

	noname := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
	}))
	defer noname.Close()
	_, err = uploadBytesToStorage(strings.TrimPrefix(noname.URL, "http://"), "http://app", "tickets", "a.png", "image/png", []byte("hi"))
	if err == nil || !strings.Contains(err.Error(), "did not return file path") {
		t.Fatalf("err=%v", err)
	}
}

func TestParseTicketFormFields_MultipartURLEncodedAndInvalidReceiver(t *testing.T) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	_ = w.WriteField("title", "T")
	_ = w.WriteField("content", "C")
	_ = w.WriteField("department", "technical_support")
	_ = w.WriteField("reciever", "9")
	_ = w.Close()
	req := httptest.NewRequest(http.MethodPost, "/", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	title, content, dept, rid, err := parseTicketFormFields(req)
	if err != nil || title != "T" || content != "C" || dept != "technical_support" || rid == nil || *rid != 9 {
		t.Fatalf("multipart title=%s content=%s dept=%s rid=%v err=%v", title, content, dept, rid, err)
	}

	form := url.Values{}
	form.Set("title", "T")
	form.Set("content", "C")
	form.Set("reciever", "abc")
	req = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	_, _, _, _, err = parseTicketFormFields(req)
	if err == nil || !strings.Contains(err.Error(), "invalid reciever") {
		t.Fatalf("err=%v", err)
	}

	form = url.Values{}
	form.Set("title", "T")
	form.Set("content", "C")
	form.Set("reciever", "4")
	req = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	title, content, _, rid, err = parseTicketFormFields(req)
	if err != nil || title != "T" || rid == nil || *rid != 4 {
		t.Fatalf("urlencoded err=%v title=%s rid=%v", err, title, rid)
	}

	req = httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Content-Type", "application/json")
	title, content, dept, rid, err = parseTicketFormFields(req)
	if err != nil || title != "" || content != "" || dept != "" || rid != nil {
		t.Fatalf("json content-type should skip form parse")
	}
}

func TestParseNoteAndReportFormFields(t *testing.T) {
	form := url.Values{}
	form.Set("title", "N")
	form.Set("content", "C")
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	title, content, err := parseNoteFormFields(req)
	if err != nil || title != "N" || content != "C" {
		t.Fatalf("note form err=%v title=%s content=%s", err, title, content)
	}

	rform := url.Values{}
	rform.Set("title", "T")
	rform.Set("content", "C")
	rform.Set("subject", "displayError")
	rform.Set("url", "https://x")
	req = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(rform.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	title, content, subject, u, err := parseReportFormFields(req)
	if err != nil || title != "T" || subject != "displayError" || u != "https://x" {
		t.Fatalf("report form err=%v title=%s subject=%s url=%s", err, title, subject, u)
	}

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	_ = w.WriteField("title", "N")
	_ = w.WriteField("content", "C")
	_ = w.Close()
	req = httptest.NewRequest(http.MethodPost, "/", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	title, content, err = parseNoteFormFields(req)
	if err != nil || title != "N" {
		t.Fatalf("note multipart err=%v", err)
	}
}

func TestUploadReportAttachments_TooManyAndInvalidType(t *testing.T) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	for i := 0; i < 6; i++ {
		part, err := w.CreateFormFile("attachments[]", string(rune('a'+i))+".png")
		if err != nil {
			t.Fatal(err)
		}
		_, _ = part.Write([]byte("x"))
	}
	_ = w.Close()
	req := httptest.NewRequest(http.MethodPost, "/", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	_, err := uploadReportAttachments(req, "storage:9000", "http://app")
	if err == nil || !strings.Contains(err.Error(), "more than 5") {
		t.Fatalf("err=%v", err)
	}

	buf.Reset()
	w = multipart.NewWriter(&buf)
	part, err := w.CreateFormFile("attachments[]", "x.exe")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = part.Write([]byte("x"))
	_ = w.Close()
	req = httptest.NewRequest(http.MethodPost, "/", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	_, err = uploadReportAttachments(req, "storage:9000", "http://app")
	if err == nil || !strings.Contains(err.Error(), "invalid attachment type") {
		t.Fatalf("err=%v", err)
	}

	req = httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Content-Type", "application/json")
	paths, err := uploadReportAttachments(req, "storage:9000", "http://app")
	if err != nil || paths != nil {
		t.Fatalf("non-multipart paths=%v err=%v", paths, err)
	}
}

func TestResolveNoteAttachmentURL_InvalidTypeAndClear(t *testing.T) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, err := w.CreateFormFile("attachment", "x.exe")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = part.Write([]byte("x"))
	_ = w.Close()
	req := httptest.NewRequest(http.MethodPost, "/", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	_, _, err = resolveNoteAttachmentURL(req, "storage:9000", "http://app")
	if err == nil || !strings.Contains(err.Error(), "invalid attachment type") {
		t.Fatalf("err=%v", err)
	}

	buf.Reset()
	w = multipart.NewWriter(&buf)
	_ = w.WriteField("attachments[]", "")
	_ = w.Close()
	req = httptest.NewRequest(http.MethodPost, "/", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	url, clear, err := resolveNoteAttachmentURL(req, "storage:9000", "http://app")
	if err != nil || url != "" || !clear {
		t.Fatalf("clear url=%q clear=%v err=%v", url, clear, err)
	}

	buf.Reset()
	w = multipart.NewWriter(&buf)
	_ = w.WriteField("current_attachments[]", "keep.png")
	_ = w.Close()
	req = httptest.NewRequest(http.MethodPost, "/", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	url, clear, err = resolveNoteAttachmentURL(req, "storage:9000", "http://app")
	if err != nil || url != "keep.png" || clear {
		t.Fatalf("current url=%q clear=%v err=%v", url, clear, err)
	}

	req = httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Content-Type", "application/json")
	url, clear, err = resolveNoteAttachmentURL(req, "storage:9000", "http://app")
	if err != nil || url != "" || clear {
		t.Fatalf("non-multipart url=%q clear=%v err=%v", url, clear, err)
	}
}

func TestUploadReportFileHeader_ExceedsSize(t *testing.T) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, err := w.CreateFormFile("attachments[]", "big.png")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = part.Write(bytes.Repeat([]byte("x"), maxReportAttachmentSize+1))
	_ = w.Close()
	req := httptest.NewRequest(http.MethodPost, "/", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	_, err = uploadReportAttachments(req, "storage:9000", "http://app")
	if err == nil || !strings.Contains(err.Error(), "1MB") {
		t.Fatalf("err=%v", err)
	}
}

func TestUploadBytesToStorageWithRelativePath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"path": "uploads/reports/", "name": "a.png", "success": true,
		})
	}))
	defer srv.Close()
	full, rel, err := uploadBytesToStorageWithRelativePath(strings.TrimPrefix(srv.URL, "http://"), "http://app", "reports", "a.png", "image/png", []byte("hi"))
	if err != nil || !strings.Contains(full, "a.png") || rel != "reports/a.png" {
		t.Fatalf("full=%q rel=%q err=%v", full, rel, err)
	}
}
