package testutil

import (
	"errors"
	"io"

	"metarang/storage-service/internal/ftp"
)

// FailingFTPClient implements ftp.FTPClientInterface with injectable errors.
type FailingFTPClient struct {
	UploadErr    error
	DownloadErr  error
	DeleteErr    error
	DownloadBody io.ReadCloser
	BaseURL      string
}

func (c *FailingFTPClient) UploadFile(string, io.Reader) error {
	if c.UploadErr != nil {
		return c.UploadErr
	}
	return nil
}

func (c *FailingFTPClient) DownloadFile(string) (io.ReadCloser, error) {
	if c.DownloadErr != nil {
		return nil, c.DownloadErr
	}
	if c.DownloadBody != nil {
		return c.DownloadBody, nil
	}
	return nil, errors.New("file not found")
}

func (c *FailingFTPClient) DeleteFile(string) error {
	if c.DeleteErr != nil {
		return c.DeleteErr
	}
	return nil
}

func (c *FailingFTPClient) GenerateURL(remotePath string) string {
	base := c.BaseURL
	if base == "" {
		base = "http://ftp.test"
	}
	return base + "/" + remotePath
}

func (c *FailingFTPClient) Close() error { return nil }

// ErrReadCloser is an io.ReadCloser whose Read always fails.
type ErrReadCloser struct{ Err error }

func (e ErrReadCloser) Read([]byte) (int, error) {
	if e.Err != nil {
		return 0, e.Err
	}
	return 0, errors.New("read failed")
}

func (e ErrReadCloser) Close() error { return nil }

var _ ftp.FTPClientInterface = (*FailingFTPClient)(nil)
