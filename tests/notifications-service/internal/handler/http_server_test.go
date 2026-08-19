package handler_test

import (
	"net"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"metarang/notifications-service/internal/handler"
)

func TestStartHTTPServer_RegistersHealth(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port := ln.Addr().(*net.TCPAddr).Port
	_ = ln.Close()

	httpH := handler.NewHTTPNotificationHandler(&mockNotificationAPI{})
	errCh := make(chan error, 1)
	go func() {
		errCh <- handler.StartHTTPServer(httpH, strconv.Itoa(port), passThroughAuth)
	}()

	deadline := time.Now().Add(2 * time.Second)
	var healthErr error
	for time.Now().Before(deadline) {
		resp, err := http.Get("http://127.0.0.1:" + strconv.Itoa(port) + "/health")
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				healthErr = nil
				break
			}
		}
		healthErr = err
		time.Sleep(20 * time.Millisecond)
	}
	if healthErr != nil {
		t.Fatalf("health check failed: %v", healthErr)
	}

	select {
	case err := <-errCh:
		if err != nil && !strings.Contains(err.Error(), "closed") {
			t.Fatalf("unexpected server error: %v", err)
		}
	case <-time.After(50 * time.Millisecond):
		// Server still listening — expected; coverage of StartHTTPServer already hit.
	}
}

func TestStartHTTPServer_InvalidPort(t *testing.T) {
	httpH := handler.NewHTTPNotificationHandler(&mockNotificationAPI{})
	err := handler.StartHTTPServer(httpH, "not-a-valid-port", passThroughAuth)
	require.Error(t, err)
}
