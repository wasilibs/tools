package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func main() {
	var opts []string
	var args []string
	for i, arg := range os.Args[1:] {
		if arg == "run" {
			continue
		}
		if arg == "--" {
			args = append(args, os.Args[i:]...)
			break
		}
		if !strings.HasPrefix(arg, "--") {
			args = append(args, arg)
			continue
		}
		if strings.HasPrefix(arg, "--dir=") {
			if strings.Contains(arg, "..") {
				// Ignore paths other than current since they're not allowed.
				continue
			}
			if arg == "--dir=." {
				arg = "--mount=.:/"
			} else {
				arg = strings.Replace(arg, "--dir", "--mount", 1)
			}
		}
		if strings.HasPrefix(arg, "--mapdir=") {
			mapping := arg[len("--mapdir="):]
			guest, host, _ := strings.Cut(mapping, "::")
			arg = "--mount=" + host + ":" + guest
		}
		opts = append(opts, arg)
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
