package web

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/syrm/maille/internal"
)

type importerStub struct {
	count int
	err   error
}

func (s importerStub) Import(context.Context, io.Reader) (int, error) { return s.count, s.err }

type classifierStub struct {
	err error
}

func (s classifierStub) Classify(context.Context) error { return s.err }

func uploadRequest(t *testing.T) *http.Request {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	file, err := writer.CreateFormFile("file", "statement.ofx")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write([]byte("OFX")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/upload", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}

func TestUploadPostRedirectsToDashboardAfterImport(t *testing.T) {
	upload := Upload{
		Importer: importerStub{count: 2}, Classifier: classifierStub{},
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	response := httptest.NewRecorder()

	upload.Post(response, uploadRequest(t))

	if response.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusSeeOther)
	}
	if location := response.Header().Get("Location"); location != "/?count=2&status=imported" {
		t.Fatalf("Location = %q", location)
	}
}

func TestUploadPostReportsDuplicate(t *testing.T) {
	upload := Upload{
		Importer: importerStub{err: internal.ErrDuplicateImport}, Classifier: classifierStub{},
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	response := httptest.NewRecorder()

	upload.Post(response, uploadRequest(t))

	if location := response.Header().Get("Location"); location != "/upload?error=duplicate" {
		t.Fatalf("Location = %q", location)
	}
}

func TestUploadPostKeepsSuccessfulImportWhenClassificationFails(t *testing.T) {
	upload := Upload{
		Importer: importerStub{count: 1}, Classifier: classifierStub{err: errors.New("classifier unavailable")},
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	response := httptest.NewRecorder()

	upload.Post(response, uploadRequest(t))

	if location := response.Header().Get("Location"); location != "/?count=1&status=imported&warning=classification" {
		t.Fatalf("Location = %q", location)
	}
}
