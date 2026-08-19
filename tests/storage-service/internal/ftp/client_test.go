package ftp_test

import (
	"bufio"
	"bytes"
	"io"
	"net"
	"strings"
	"testing"
	"time"

	"metarang/storage-service/internal/ftp"
)

func startFakeFTPServer(t *testing.T, loginOK bool) (host, port string) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go serveFakeFTP(conn, loginOK)
		}
	}()

	host, port, err = net.SplitHostPort(ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	return host, port
}

func serveFakeFTP(conn net.Conn, loginOK bool) {
	defer func() { _ = conn.Close() }()
	_ = conn.SetDeadline(time.Now().Add(3 * time.Second))
	_, _ = conn.Write([]byte("220 fake ftp\r\n"))
	reader := bufio.NewReader(conn)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		cmd := strings.ToUpper(strings.TrimSpace(strings.SplitN(line, " ", 2)[0]))
		switch cmd {
		case "USER":
			_, _ = conn.Write([]byte("331 Password required\r\n"))
		case "PASS":
			if loginOK {
				_, _ = conn.Write([]byte("230 Logged in\r\n"))
			} else {
				_, _ = conn.Write([]byte("530 Login incorrect\r\n"))
			}
		case "FEAT":
			_, _ = conn.Write([]byte("211-Features\r\n UTF8\r\n211 End\r\n"))
		case "OPTS", "TYPE", "NOOP", "PWD", "SYST":
			_, _ = conn.Write([]byte("200 OK\r\n"))
		case "QUIT":
			_, _ = conn.Write([]byte("221 Bye\r\n"))
			return
		case "PASV", "EPSV", "PORT", "EPRT", "STOR", "RETR", "DELE", "CWD", "MKD":
			_, _ = conn.Write([]byte("550 requested action failed\r\n"))
		default:
			_, _ = conn.Write([]byte("502 Command not implemented\r\n"))
		}
	}
}

func TestFTPClient_GenerateURL(t *testing.T) {
	client := ftp.NewFTPClient("127.0.0.1", "21", "user", "pass", "http://cdn.example.com/files")
	got := client.GenerateURL("avatars/photo.jpg")
	want := "http://cdn.example.com/files/avatars/photo.jpg"
	if got != want {
		t.Fatalf("GenerateURL = %q, want %q", got, want)
	}
}

func TestFTPClient_Connect_ClosedPort(t *testing.T) {
	client := ftp.NewFTPClient("127.0.0.1", "1", "user", "pass", "http://example.com")
	if err := client.Connect(); err == nil {
		t.Fatal("expected connect error against closed port")
	}
}

func TestFTPClient_UploadDownloadDelete_ClosedPort(t *testing.T) {
	client := ftp.NewFTPClient("127.0.0.1", "1", "user", "pass", "http://example.com")

	if err := client.UploadFile("x.txt", bytes.NewReader([]byte("x"))); err == nil {
		t.Fatal("expected UploadFile error")
	}
	if _, err := client.DownloadFile("x.txt"); err == nil {
		t.Fatal("expected DownloadFile error")
	}
	if err := client.DeleteFile("x.txt"); err == nil {
		t.Fatal("expected DeleteFile error")
	}
}

func TestFTPClient_Connect_LoginRejected(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	done := make(chan struct{})
	go func() {
		defer close(done)
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()
		_ = conn.SetDeadline(time.Now().Add(2 * time.Second))
		_, _ = conn.Write([]byte("220 test ftp\r\n"))
		buf := make([]byte, 512)
		_, _ = conn.Read(buf)
		_, _ = conn.Write([]byte("331 Password required\r\n"))
		_, _ = conn.Read(buf)
		_, _ = conn.Write([]byte("530 Login incorrect\r\n"))
		_, _ = io.Copy(io.Discard, conn)
	}()

	_, port, err := net.SplitHostPort(ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	client := ftp.NewFTPClient("127.0.0.1", port, "user", "wrong", "http://example.com")
	err = client.Connect()
	_ = ln.Close()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
	}
	if err == nil {
		t.Fatal("expected login error")
	}
	if !strings.Contains(err.Error(), "failed to login to FTP") && !strings.Contains(err.Error(), "failed to connect to FTP") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFTPClient_ConnectAndClose_FakeServer(t *testing.T) {
	host, port := startFakeFTPServer(t, true)
	client := ftp.NewFTPClient(host, port, "user", "pass", "http://example.com")
	if err := client.Connect(); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	if err := client.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func TestFTPClient_UploadDownloadDelete_AfterLogin(t *testing.T) {
	host, port := startFakeFTPServer(t, true)
	client := ftp.NewFTPClient(host, port, "user", "pass", "http://example.com")
	if err := client.Connect(); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })

	if err := client.UploadFile("x.txt", bytes.NewReader([]byte("x"))); err == nil {
		t.Fatal("expected UploadFile error after login")
	}
	if _, err := client.DownloadFile("x.txt"); err == nil {
		t.Fatal("expected DownloadFile error after login")
	}
	if err := client.DeleteFile("x.txt"); err == nil {
		t.Fatal("expected DeleteFile error after login")
	}
}
