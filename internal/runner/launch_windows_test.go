//go:build windows

package runner

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestWindowsBatchArgumentHelper(t *testing.T) {
	if os.Getenv("MERLIN_BATCH_ARGUMENT_HELPER") != "1" {
		return
	}
	for index, arg := range os.Args {
		if arg == "--" {
			data, err := json.Marshal(os.Args[index+1:])
			if err != nil {
				os.Exit(2)
			}
			fmt.Println(string(data))
			os.Exit(0)
		}
	}
	os.Exit(3)
}

func waitLauncher(t *testing.T, process childProcess) (int, error) {
	t.Helper()
	type result struct {
		code int
		err  error
	}
	done := make(chan result, 1)
	go func() { code, err := process.Wait(); done <- result{code, err} }()
	select {
	case outcome := <-done:
		return outcome.code, outcome.err
	case <-time.After(10 * time.Second):
		if err := process.Stop(); err != nil {
			t.Error(err)
		}
		select {
		case <-done:
		case <-time.After(time.Second):
		}
		t.Fatal("launcher did not finish")
		return 0, nil
	}
}

func TestResolveExecutableFromPath(t *testing.T) {
	for _, extension := range []string{".exe", ".com", ".cmd", ".bat"} {
		t.Run(extension, func(t *testing.T) {
			directory := t.TempDir()
			path := filepath.Join(directory, "custom-runtime"+extension)
			if err := os.WriteFile(path, []byte("test fixture"), 0600); err != nil {
				t.Fatal(err)
			}
			t.Setenv("PATH", directory)
			t.Setenv("PATHEXT", ".EXE;.COM;.CMD;.BAT")
			args := []string{"run", "dev"}
			resolved, actualArgs, err := resolveExecutable("custom-runtime", args, nil, directory)
			if err != nil || !strings.EqualFold(resolved, path) || !reflect.DeepEqual(actualArgs, args) {
				t.Fatalf("PATH launcher was not preserved: %q %#v %v", resolved, actualArgs, err)
			}
		})
	}
}

func TestWindowsBatchLaunchersPreserveArguments(t *testing.T) {
	helper, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for _, extension := range []string{".cmd", ".bat"} {
		t.Run(extension, func(t *testing.T) {
			directory := t.TempDir()
			tools := filepath.Join(directory, "Tools & (shared) %TEMP% !")
			if err := os.Mkdir(tools, 0700); err != nil {
				t.Fatal(err)
			}
			shim := filepath.Join(tools, "sample-runner"+extension)
			body := "@echo off\r\n\"%MERLIN_BATCH_HELPER%\" -test.run=TestWindowsBatchArgumentHelper -- %*\r\nexit /b %errorlevel%\r\n"
			if err := os.WriteFile(shim, []byte(body), 0600); err != nil {
				t.Fatal(err)
			}
			t.Setenv("PATH", tools)
			t.Setenv("PATHEXT", ".EXE;.CMD;.BAT")
			args := []string{"run", "dev", "value with spaces", "a&b|c<d>e^f", "%PATH%", "!NAME!", "", `tail\`, "ñandú"}
			path, resolvedArgs, err := resolveExecutable("sample-runner", args, nil, directory)
			if err != nil {
				t.Fatal(err)
			}
			env := append(executionEnvironment(), "MERLIN_BATCH_HELPER="+helper, "MERLIN_BATCH_ARGUMENT_HELPER=1", launcherEnvPrefix+"1=overridden")
			var output bytes.Buffer
			process, err := startProcess(path, resolvedArgs, directory, env, &output)
			if err != nil {
				t.Fatal(err)
			}
			code, err := waitLauncher(t, process)
			if err != nil || code != 0 {
				t.Fatalf("batch launcher failed: %d %v %s", code, err, output.String())
			}
			var received []string
			if err := json.Unmarshal(bytes.TrimSpace(output.Bytes()), &received); err != nil || !reflect.DeepEqual(received, args) {
				t.Fatalf("arguments were interpolated or changed: %#v %v; output=%s", received, err, output.String())
			}
		})
	}
}

func TestWindowsBatchRejectsCommandText(t *testing.T) {
	for _, argument := range []string{"quoted\" & echo unexpected", "line\r\nbreak", "zero\x00byte"} {
		if _, _, _, err := windowsLaunchCommand(`C:\tools\runner.cmd`, []string{argument}, nil); err == nil {
			t.Fatal("accepted batch control text")
		}
	}
}

func TestInstalledPnpmLauncher(t *testing.T) {
	if os.Getenv("MERLIN_TEST_INSTALLED_PNPM") != "1" {
		t.Skip("opt-in check of the installed host launcher")
	}
	directory := t.TempDir()
	path, args, err := resolveExecutable("pnpm", []string{"--version"}, nil, directory)
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	process, err := startProcess(path, args, directory, executionEnvironment(), &output)
	if err != nil {
		t.Fatal(err)
	}
	code, err := waitLauncher(t, process)
	if err != nil || code != 0 || !strings.Contains(output.String(), ".") {
		t.Fatalf("installed pnpm did not start: %d %v %s", code, err, output.String())
	}
	t.Logf("installed pnpm version: %s", strings.TrimSpace(output.String()))
}
