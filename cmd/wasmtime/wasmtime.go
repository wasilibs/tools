package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// main is a wrapper around wazero that can be invoked by tinygo test.
func main() {
	tempDir, err := os.MkdirTemp("", "tinygo")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tempDir)
	opts := []string{"-mount=.:/", fmt.Sprintf("-mount=%s:/tmp", tempDir)}
	var args []string
	osArgs := os.Args[1:]
	for i, arg := range osArgs {
		if arg == "run" {
			continue
		}
		if arg == "--" {
			args = append(args, osArgs[i:]...)
			break
		}
		if !strings.HasPrefix(arg, "--") {
			args = append(args, arg)
			continue
		}
		// Ignore other flags, we add what's needed for wazero and tinygo test to work
		// manually.
	}

	goCmd := filepath.Join(runtime.GOROOT(), "bin", "go")
	cmdArgs := append([]string{"run"}, "github.com/tetratelabs/wazero/cmd/wazero@v1.0.0-pre.8")
	cmdArgs = append(cmdArgs, "run")
	cmdArgs = append(cmdArgs, opts...)
	cmdArgs = append(cmdArgs, args...)
	cmd := exec.Command(goCmd, cmdArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	_ = cmd.Run()
	os.Exit(cmd.ProcessState.ExitCode())
}
