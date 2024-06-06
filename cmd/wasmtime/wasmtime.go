package main

import (
	"context"
	"os"
	"strings"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
	"github.com/tetratelabs/wazero/sys"
)

// main is a wrapper around wazero that can be invoked by tinygo test.
func main() {
	os.Exit(doMain())
}

func doMain() int {
	tempDir, err := os.MkdirTemp("", "tinygo")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tempDir)
	var args []string
	osArgs := os.Args[1:]
	var wasmPath string
	for i, arg := range osArgs {
		if wasmPath != "" {
			args = append(args, osArgs[i:]...)
			break
		}

		if arg == "run" || strings.HasPrefix(arg, "--") {
			continue
		}

		wasmPath = arg
	}

	wasm, err := os.ReadFile(wasmPath)
	if err != nil {
		panic(err)
	}

	ctx := context.Background()
	rt := wazero.NewRuntime(ctx)
	defer rt.Close(ctx)

	wasi_snapshot_preview1.MustInstantiate(ctx, rt)

	cfg := wazero.NewModuleConfig().
		WithArgs(args...).
		WithStdout(os.Stdout).
		WithStderr(os.Stderr).
		WithFSConfig(wazero.NewFSConfig().
			WithDirMount(".", "/").
			WithDirMount(tempDir, "/tmp"))

	for _, kv := range os.Environ() {
		if k, v, ok := strings.Cut(kv, "="); ok {
			cfg = cfg.WithEnv(k, v)
		}
	}

	_, err = rt.InstantiateWithConfig(ctx, wasm, cfg)
	if err != nil {
		if exitErr, ok := err.(*sys.ExitError); ok { //nolint:errorlint
			exitCode := exitErr.ExitCode()
			return int(exitCode)
		}
		panic(err)
	}

	return 0
}
