package tasks

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/google/go-github/github"
	"github.com/goyek/goyek/v2"
	"github.com/goyek/x/cmd"
)

var (
	flagTestMode *string
)

// Params are parameters for the defined tasks.
type Params struct {
	// LibraryName is the name of the library, generally used to determine build tags.
	LibraryName string

	// LibraryRepo is the upstream repository of the library for determining updates.
	LibraryRepo string

	// EnableTestModes indicates whether non-wazero test modes are enabled.
	EnableTestModes bool

	// GoReleaser indicates tasks using goreleaser are enabled.
	GoReleaser bool
}

// Define defines the tasks for a wasilibs build.
func Define(params Params) {
	tags := buildTags(params.LibraryName)
	goyek.Define(goyek.Task{
		Name:  "format",
		Usage: "Formats the code.",
		Action: func(a *goyek.A) {
			cmd.Exec(a, fmt.Sprintf("go run mvdan.cc/gofumpt@%s -l -w .", verGoFumpt))
			cmd.Exec(a, fmt.Sprintf("go run github.com/rinchsan/gosimports/cmd/gosimports@%s -w -local github.com/wasilibs .", verGosImports))
		},
	})

	lint := goyek.Define(goyek.Task{
		Name:  "lint",
		Usage: "Executes linters against the code.",
		Action: func(a *goyek.A) {
			cmd.Exec(a, fmt.Sprintf("go run github.com/golangci/golangci-lint/cmd/golangci-lint@%s run --build-tags %s --timeout 30m", verGolangCILint, tags))
		},
	})

	var test *goyek.DefinedTask
	if params.EnableTestModes {
		flagTestMode = flag.String("test-mode", "wazero", "Test mode to use (wazero, cgo, tinygo).")
		test = goyek.Define(goyek.Task{
			Name:  "test",
			Usage: "Runs unit tests. test-mode flag is used to specify the mode for running tests.",
			Action: func(a *goyek.A) {
				mode := testMode()
				if mode != modeTinyGo {
					cmd.Exec(a, fmt.Sprintf("go test -v -timeout=20m -tags %s ./...", tags))
				} else {
					cmd.Exec(a, fmt.Sprintf("tinygo test -target=wasi -v -tags %s ./...", tags))
				}
			},
		})
	} else {
		test = goyek.Define(goyek.Task{
			Name:  "test",
			Usage: "Runs unit tests.",
			Action: func(a *goyek.A) {
				cmd.Exec(a, "go test -v -timeout=20m ./...")
			},
		})
	}

	goyek.Define(goyek.Task{
		Name:  "check",
		Usage: "Runs all checks.",
		Deps:  goyek.Deps{lint, test},
	})

	goyek.Define(goyek.Task{
		Name:  "wasm",
		Usage: "Builds the WebAssembly module.",
		Action: func(a *goyek.A) {
			buildWasm(a)
		},
	})

	goyek.Define(goyek.Task{
		Name:  "update",
		Usage: "Checks upstream repo for new version and updates and builds if so.",
		Action: func(a *goyek.A) {
			currBytes, err := os.ReadFile(filepath.Join("buildtools", "wasm", "version.txt"))
			if err != nil {
				a.Error(err)
			}
			curr := strings.TrimSpace(string(currBytes))

			gh, err := api.DefaultRESTClient()
			if err != nil {
				a.Error(err)
			}

			var release *github.RepositoryRelease
			if err := gh.Get(fmt.Sprintf("repos/%s/releases/latest", params.LibraryRepo), &release); err != nil {
				a.Error(err)
			}

			if release == nil {
				a.Error("could not find releases")
			}

			latest := release.GetTagName()
			if latest == curr {
				fmt.Println("up to date")
				return
			}

			fmt.Println("updating to", latest)
			if err := os.WriteFile(filepath.Join("buildtools", "wasm", "version.txt"), []byte(latest), 0o644); err != nil {
				a.Error(err)
			}

			buildWasm(a)
		},
	})

	if params.GoReleaser {
		goyek.Define(goyek.Task{
			Name:  "release",
			Usage: "Builds and uploads executables to a GitHub release.",
			Action: func(a *goyek.A) {
				cmd.Exec(a, fmt.Sprintf("go run github.com/goreleaser/goreleaser@%s release --clean", verGoReleaser))
			},
		})

		goyek.Define(goyek.Task{
			Name:  "snapshot",
			Usage: "Builds the executables.",
			Action: func(a *goyek.A) {
				cmd.Exec(a, fmt.Sprintf("go run github.com/goreleaser/goreleaser@%s release --snapshot --clean", verGoReleaser))
			},
		})
	}
}

func buildWasm(a *goyek.A) {
	cmd.Exec(a, fmt.Sprintf("docker build -t wasilibs-build -f %s .", filepath.Join("buildtools", "wasm", "Dockerfile")))
	wd, err := os.Getwd()
	if err != nil {
		a.Error(err)
	}
	wasmDir := filepath.Join(wd, "internal", "wasm")
	if err := os.MkdirAll(wasmDir, 0o755); err != nil {
		a.Error(err)
	}
	cmd.Exec(a, fmt.Sprintf("docker run --rm -v %s:/out wasilibs-build", wasmDir))
}

type mode byte

const (
	modeWazero mode = iota
	modeCgo
	modeTinyGo
)

func testMode() mode {
	mode := strings.ToLower(*flagTestMode)

	switch mode {
	case "wazero":
		return modeWazero
	case "cgo":
		return modeCgo
	case "tinygo":
		return modeTinyGo
	default:
		return modeWazero
	}
}

func buildTags(libraryName string) string {
	var tags []string

	if testMode() == modeCgo {
		tags = append(tags, fmt.Sprintf("%s_cgo", libraryName))
	}

	return strings.Join(tags, ",")
}
