package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestVersionFlagPrintsTheStampedVersion(t *testing.T) {
	version = "1.2.3"
	defer func() { version = "dev" }()
	var stdout, stderr bytes.Buffer
	if code := run(context.Background(), []string{"--version"}, strings.NewReader(""), &stdout, &stderr); code != 0 || stdout.String() != "1.2.3\n" {
		t.Fatalf("exit %d, stdout %q, stderr %q", code, stdout.String(), stderr.String())
	}
}

func TestAnUnknownFlagFails(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run(context.Background(), []string{"--nope"}, strings.NewReader(""), &stdout, &stderr); code != 2 {
		t.Fatalf("exit %d", code)
	}
}

// The real stdio path: a framed ping on stdin is answered on stdout, and the engine
// exits cleanly when the editor closes stdin.
func TestServesStdinUntilItEnds(t *testing.T) {
	stdinReader, stdinWriter := io.Pipe()
	stdoutReader, stdoutWriter := io.Pipe()
	var stderr bytes.Buffer
	exited := make(chan int, 1)
	go func() { exited <- run(context.Background(), nil, stdinReader, stdoutWriter, &stderr) }()

	body := `{"jsonrpc":"2.0","id":1,"method":"ping"}`
	if _, err := fmt.Fprintf(stdinWriter, "Content-Length: %d\r\n\r\n%s", len(body), body); err != nil {
		t.Fatal(err)
	}
	reply := bufio.NewReader(stdoutReader)
	header, err := reply.ReadString('\n')
	if err != nil || !strings.HasPrefix(header, "Content-Length: ") {
		t.Fatalf("header %q %v", header, err)
	}
	length, _ := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(header, "Content-Length: ")))
	_, _ = reply.ReadString('\n')
	message := make([]byte, length)
	if _, err := io.ReadFull(reply, message); err != nil || !strings.Contains(string(message), `"pong":"pong"`) {
		t.Fatalf("reply %q %v", message, err)
	}

	_ = stdinWriter.Close()
	select {
	case code := <-exited:
		if code != 0 {
			t.Fatalf("exit %d, stderr %q", code, stderr.String())
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the engine did not exit when stdin closed")
	}
}
