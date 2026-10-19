//go:build windows

package runner

import (
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/sys/windows"
)

const launcherEnvPrefix = "MERLIN_RUNNER_LAUNCH_"

// Batch launchers need cmd's quoting rules, not CommandLineToArgvW. Pass literal
// values through environment variables so percent signs are not interpolated as
// part of our command text. No user/model command string is accepted here.
func windowsLaunchCommand(executable string, args, env []string) (string, string, []string, error) {
	ext := strings.ToLower(filepath.Ext(executable))
	if ext != ".cmd" && ext != ".bat" {
		return executable, windows.ComposeCommandLine(append([]string{executable}, args...)), env, nil
	}
	systemDir, err := windows.GetSystemDirectory()
	if err != nil {
		return "", "", nil, err
	}
	var launchEnv []string
	for _, item := range env {
		key, _, _ := strings.Cut(item, "=")
		if !strings.HasPrefix(strings.ToUpper(key), launcherEnvPrefix) {
			launchEnv = append(launchEnv, item)
		}
	}
	var command strings.Builder
	shell := filepath.Join(systemDir, "cmd.exe")
	command.WriteString(windows.EscapeArg(shell) + ` /d /s /v:off /c "`)
	values := append([]string{executable}, args...)
	for index, value := range values {
		if strings.ContainsAny(value, "\x00\r\n\"") {
			return "", "", nil, policy("El lanzador de Windows no admite comillas ni saltos de línea en sus argumentos.")
		}
		key := launcherEnvPrefix + strconv.Itoa(index)
		if index > 0 {
			command.WriteByte(' ')
			// %* forwards quoted arguments to the executable behind the batch shim.
			// Preserve trailing backslashes when its CRT parses those quotes.
			value += strings.Repeat(`\`, len(value)-len(strings.TrimRight(value, `\`)))
		}
		if value == "" {
			command.WriteString(`""`)
			continue
		}
		launchEnv = append(launchEnv, key+"="+value)
		command.WriteString(`"%` + key + `%"`)
	}
	command.WriteByte('"')
	return shell, command.String(), launchEnv, nil
}
