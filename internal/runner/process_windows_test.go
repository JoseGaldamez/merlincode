//go:build windows

package runner

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"golang.org/x/sys/windows"
)

func TestRunHelperProcess(t *testing.T) {
	if os.Getenv("MERLIN_RUN_HELPER") != "1" {
		return
	}
	if len(os.Args) > 2 && os.Args[len(os.Args)-2] == "parent" {
		command := exec.Command(os.Args[0], "-test.run=^TestRunHelperProcess$", "--", "child")
		if err := command.Start(); err != nil {
			os.Exit(3)
		}
		if err := os.WriteFile(os.Args[len(os.Args)-1], []byte(strconv.Itoa(command.Process.Pid)), 0600); err != nil {
			os.Exit(4)
		}
		fmt.Println("server ready")
	}
	for {
		time.Sleep(time.Hour)
	}
}

func TestWindowsJobStopsDescendantsAndDrainsOutput(t *testing.T) {
	directory := t.TempDir()
	pidPath := filepath.Join(directory, "child.pid")
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	process, err := startProcess(executable, []string{"-test.run=^TestRunHelperProcess$", "--", "parent", pidPath}, directory, append(os.Environ(), "MERLIN_RUN_HELPER=1"), &output)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { process.Stop() })
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var pid int
	for pid == 0 {
		data, err := os.ReadFile(pidPath)
		if err == nil {
			pid, _ = strconv.Atoi(string(data))
		}
		if pid != 0 {
			break
		}
		select {
		case <-ctx.Done():
			t.Fatal("helper did not start")
		case <-time.After(10 * time.Millisecond):
		}
	}
	child, err := windows.OpenProcess(windows.SYNCHRONIZE, false, uint32(pid))
	if err != nil {
		t.Fatal(err)
	}
	defer windows.CloseHandle(child)
	if err := process.Stop(); err != nil {
		t.Fatal(err)
	}
	waitDone := make(chan struct{})
	go func() { process.Wait(); close(waitDone) }()
	select {
	case <-waitDone:
	case <-ctx.Done():
		t.Fatal("process/log reader survived stop")
	}
	state, err := windows.WaitForSingleObject(child, 3000)
	if err != nil || state != windows.WAIT_OBJECT_0 {
		t.Fatalf("descendant survived: %d %v", state, err)
	}
	if !strings.Contains(output.String(), "server ready") {
		t.Fatalf("missing stdout: %q", output.String())
	}
}
