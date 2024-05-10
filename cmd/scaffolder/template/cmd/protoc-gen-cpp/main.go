package main

import (
	"os"

	"github.com/wasilibs/go-protoc-gen-builtins/internal/runner"
	"github.com/wasilibs/go-protoc-gen-builtins/internal/wasm"
)

func main() {
	os.Exit(runner.Run("sql-formatter-cli", os.Args[1:], wasm.SqlFormatterCLI, os.Stdin, os.Stdout, os.Stderr, "."))
}
