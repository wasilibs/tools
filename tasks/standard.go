package tasks

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/curioswitch/go-build"
	"github.com/google/go-github/v74/github"
	"github.com/goyek/goyek/v2"
	"github.com/goyek/x/cmd"
)

var flagTestMode *string

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
	var opts []build.Option
	if params.EnableTestModes {
		opts = append(opts, build.ExcludeTasks("lint-go", "test-go"))
	}
	build.DefineTasks(opts...)

	runGoReleaser := "go run github.com/goreleaser/goreleaser/v2@" + verGoReleaser

	if params.EnableTestModes {
		flagTestMode = flag.String("test-mode", "wazero", "Test mode to use (wazero, cgo, tinygo).")

		build.RegisterLintTask(goyek.Define(goyek.Task{
			Name:  "lint-go",
			Usage: "Executes linters against the code.",
			Action: func(a *goyek.A) {
				cmd.Exec(a, fmt.Sprintf("go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@%s run --build-tags %s --timeout=30m", verGolangCILint, buildTags(params.LibraryName)))
			},
		}))

		build.RegisterTestTask(goyek.Define(goyek.Task{
			Name:  "test-go",
			Usage: "Runs unit tests. test-mode flag is used to specify the mode for running tests.",
			Action: func(a *goyek.A) {
				mode := testMode()
				if mode != modeTinyGo {
					cmd.Exec(a, fmt.Sprintf("go test -v -timeout=20m -tags %s ./...", buildTags(params.LibraryName)))
				} else {
					cmd.Exec(a, fmt.Sprintf("tinygo test -target=wasi -v -tags %s ./...", buildTags(params.LibraryName)))
				}
			},
		}))
	}

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
				a.Fatal(err)
			}
			curr := strings.TrimSpace(string(currBytes))

			gh, err := api.DefaultRESTClient()
			if err != nil {
				a.Fatal(err)
			}

			var latest string
			var release *github.RepositoryRelease
			if err := gh.Get(fmt.Sprintf("repos/%s/releases/latest", params.LibraryRepo), &release); err != nil {
				a.Log(err)
			}

			if release != nil {
				latest = release.GetTagName()
			} else {
				a.Log("could not find releases, falling back to tag")

				var tags []github.RepositoryTag
				if err := gh.Get(fmt.Sprintf("repos/%s/tags", params.LibraryRepo), &tags); err != nil {
					a.Error(err)
				}
				if len(tags) == 0 {
					a.Fatal("could not find tags")
				}
				latest = tags[0].GetName()
			}

			if latest == curr {
				fmt.Println("up to date")
				return
			}

			fmt.Println("updating to", latest)
			if err := os.WriteFile(filepath.Join("buildtools", "wasm", "version.txt"), []byte(latest), 0o644); err != nil { //nolint:gosec
				a.Error(err)
			}

			buildWasm(a)
		},
	})

	if params.GoReleaser {
		build.RegisterCommandDownloads(runGoReleaser + " --version")
		goyek.Define(goyek.Task{
			Name:  "release",
			Usage: "Builds and uploads executables to a GitHub release.",
			Action: func(a *goyek.A) {
				cmd.Exec(a, runGoReleaser+" release --clean")
			},
		})

		goyek.Define(goyek.Task{
			Name:  "snapshot",
			Usage: "Builds the executables.",
			Action: func(a *goyek.A) {
				cmd.Exec(a, runGoReleaser+" release --snapshot --clean")
			},
		})
	}
}

func buildWasm(a *goyek.A) {
	if !cmd.Exec(a, fmt.Sprintf("docker build -t wasilibs-build -f %s .", filepath.Join("buildtools", "wasm", "Dockerfile"))) {
		return
	}
	wd, err := os.Getwd()
	if err != nil {
		a.Fatal(err)
	}
	wasmDir := filepath.Join(wd, "internal", "wasm")
	if err := os.MkdirAll(wasmDir, 0o755); err != nil { //nolint:gosec
		a.Fatal(err)
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
	if flagTestMode == nil {
		return modeWazero
	}

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
		tags = append(tags, libraryName+"_cgo")
	}

	return strings.Join(tags, ",")
}
