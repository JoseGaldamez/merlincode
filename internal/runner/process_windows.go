//go:build windows

package runner

import (
	"errors"
	"io"
	"os"
	"sort"
	"strings"
	"sync"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows"
)

type windowsProcess struct {
	mu         sync.Mutex
	process    windows.Handle
	job        windows.Handle
	outputDone chan struct{}
	reader     *os.File
}

// The job list attribute attaches the child before its first instruction runs.
// Only stdio handles are inherited; the kill-on-close job handle never is.
func startProcess(executable string, args []string, directory string, env []string, output io.Writer) (childProcess, error) {
	executable, commandLine, env, err := windowsLaunchCommand(executable, args, env)
	if err != nil {
		return nil, err
	}
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return nil, err
	}
	owned := false
	defer func() {
		if !owned {
			windows.CloseHandle(job)
		}
	}()
	limits := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{}
	limits.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
	if _, err := windows.SetInformationJobObject(job, windows.JobObjectExtendedLimitInformation, uintptr(unsafe.Pointer(&limits)), uint32(unsafe.Sizeof(limits))); err != nil {
		return nil, err
	}
	reader, writer, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	defer writer.Close()
	defer func() {
		if !owned {
			reader.Close()
		}
	}()
	stdin, err := os.Open(os.DevNull)
	if err != nil {
		return nil, err
	}
	defer stdin.Close()
	handles := []windows.Handle{windows.Handle(writer.Fd()), windows.Handle(stdin.Fd())}
	for _, handle := range handles {
		if err := windows.SetHandleInformation(handle, windows.HANDLE_FLAG_INHERIT, windows.HANDLE_FLAG_INHERIT); err != nil {
			return nil, err
		}
	}
	attributes, err := windows.NewProcThreadAttributeList(2)
	if err != nil {
		return nil, err
	}
	defer attributes.Delete()
	if err := attributes.Update(windows.PROC_THREAD_ATTRIBUTE_HANDLE_LIST, unsafe.Pointer(&handles[0]), uintptr(len(handles))*unsafe.Sizeof(handles[0])); err != nil {
		return nil, err
	}
	const procThreadAttributeJobList = 0x0002000d // Windows 10+: PROC_THREAD_ATTRIBUTE_JOB_LIST.
	if err := attributes.Update(procThreadAttributeJobList, unsafe.Pointer(&job), unsafe.Sizeof(job)); err != nil {
		return nil, err
	}
	startup := windows.StartupInfoEx{
		StartupInfo:             windows.StartupInfo{Cb: uint32(unsafe.Sizeof(windows.StartupInfoEx{})), Flags: windows.STARTF_USESTDHANDLES, StdInput: handles[1], StdOutput: handles[0], StdErr: handles[0]},
		ProcThreadAttributeList: attributes.List(),
	}
	exe, err := windows.UTF16PtrFromString(executable)
	if err != nil {
		return nil, err
	}
	command, err := windows.UTF16PtrFromString(commandLine)
	if err != nil {
		return nil, err
	}
	cwd, err := windows.UTF16PtrFromString(directory)
	if err != nil {
		return nil, err
	}
	sort.Slice(env, func(i, j int) bool { return strings.ToUpper(env[i]) < strings.ToUpper(env[j]) })
	block := utf16.Encode([]rune(strings.Join(env, "\x00") + "\x00\x00"))
	var info windows.ProcessInformation
	flags := uint32(windows.CREATE_UNICODE_ENVIRONMENT | windows.EXTENDED_STARTUPINFO_PRESENT | windows.CREATE_NO_WINDOW)
	if err := windows.CreateProcess(exe, command, nil, nil, true, flags, &block[0], cwd, &startup.StartupInfo, &info); err != nil {
		return nil, err
	}
	windows.CloseHandle(info.Thread)
	process := &windowsProcess{process: info.Process, job: job, outputDone: make(chan struct{}), reader: reader}
	owned = true
	go func() { defer close(process.outputDone); defer reader.Close(); _, _ = io.Copy(output, reader) }()
	return process, nil
}

func (p *windowsProcess) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.job == 0 {
		return nil
	}
	return windows.TerminateJobObject(p.job, 1)
}

func (p *windowsProcess) Wait() (int, error) {
	_, waitErr := windows.WaitForSingleObject(p.process, windows.INFINITE)
	var code uint32
	codeErr := windows.GetExitCodeProcess(p.process, &code)
	p.mu.Lock()
	// Closing the job also removes descendants after an early launcher exit.
	jobErr := windows.CloseHandle(p.job)
	p.job = 0
	processErr := windows.CloseHandle(p.process)
	p.mu.Unlock()
	<-p.outputDone
	return int(code), errors.Join(waitErr, codeErr, jobErr, processErr)
}
