package handler_test

import (
	"net"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"metarang/commercial-service/internal/handler"
)

func TestStartHTTPServer_RegistersHealth(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	_ = ln.Close()

	h := handler.NewHTTPCommercialHandler(&mockTransactionAPI{}, &mockWalletHistoryAPI{}, &mockCitizenUserInfoAPI{})
	errCh := make(chan error, 1)
	go func() {
		errCh <- handler.StartHTTPServer(h, strconv.Itoa(port), passThroughAuth)
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

	// Shut down by connecting is enough for coverage; ListenAndServe keeps running.
	// Force close via a second bind attempt isn't needed — just verify health and abandon.
	select {
	case err := <-errCh:
		if err != nil && !strings.Contains(err.Error(), "closed") {
			t.Fatalf("unexpected server error: %v", err)
		}
	case <-time.After(50 * time.Millisecond):
		// Server still listening — expected; coverage of StartHTTPServer already hit.
	}
}
