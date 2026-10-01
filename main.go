package tffumpt

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2"

	diff "github.com/AleksaC/tffumpt/diff"
	format "github.com/AleksaC/tffumpt/format"
)

var supportedExts = []string{
	".tf",
	".tfvars",
	".tftest.hcl",
	".tfmock.hcl",
}

type Options struct {
	List      bool
	Write     bool
	Diff      bool
	Check     bool
	Recursive bool
}

func Fumpt(filenames []string, f *Options) int {
	return fumpt(filenames, f, true)
}

func fumpt(filenames []string, f *Options, preflight bool) int {
	status := 0

	if len(filenames) == 0 {
		filenames = []string{"."}
	}

	if preflight && f.Recursive && f.Write && !f.Check {
		if status := preflightRecursive(filenames); status != 0 {
			return status
		}
	}

	for _, filename := range filenames {
		var src []byte
		if filename == "-" {
			if len(filenames) > 1 {
				fmt.Println(
					"Extra filenames specified with -, which is only meant to be used for reading the input from stdin",
				)
				return 1
			}
			var err error
			src, err = io.ReadAll(os.Stdin)
			if err != nil {
				log.Fatalf("Failed to read from stdin: %v", err)
			}
		} else {
			fileInfo, err := os.Stat(filename)
			if err != nil {
				log.Fatalf("Failed to stat file `%s`; %v", filename, err)
			}

			if fileInfo.IsDir() {
				entries, err := os.ReadDir(filename)
				if err != nil {
					log.Fatalf("Failed to read directory: %v", err)
				}

				filePaths := make([]string, 0)
				for _, entry := range entries {
					filePath := filepath.Join(filename, entry.Name())

					info, err := os.Stat(filePath)
					if err != nil {
						log.Fatalf("Failed to stat file %s; %v", filePath, err)
					}

					if (info.IsDir() && !f.Recursive) || (!info.IsDir() && !isTerraformFile(filePath)) {
						continue
					}

					filePaths = append(filePaths, filePath)
				}

				if len(filePaths) > 0 {
					status |= fumpt(filePaths, f, false)
				}

				continue
			}

			src, err = os.ReadFile(filename)
			if err != nil {
				log.Fatalf("Failed to read file %s; %v", filename, err)
			}
		}

		res, diags := format.Format(src, filename)

		printDiagnostics(diags)
		if diags.HasErrors() {
			return 2
		}

		if filename == "-" {
			if f.Check {
				status = 3
			} else if f.Diff {
				if !bytes.Equal(src, res) {
					diff, err := diff.BytesDiff(src, res, filename)
					if err != nil {
						log.Fatalf("Could not create a diff: %v", err)
					}
					fmt.Println(string(diff))
				}
			} else {
				fmt.Print(string(res))
			}
		} else if !bytes.Equal(src, res) {
			if f.Check {
				status = 3
			} else {
				if f.Write {
					err := os.WriteFile(filename, res, 0o644)
					if err != nil {
						log.Fatalf("Failed to write file: %v", err)
					}
				}
			}

			if f.List {
				fmt.Println(filename)
			}

			if f.Diff {
				diff, err := diff.BytesDiff(src, res, filename)
				if err != nil {
					log.Fatalf("Could not create a diff: %v", err)
				}
				fmt.Println(string(diff))
			}
		}
	}

	return status
}

func preflightRecursive(filenames []string) int {
	for _, filename := range filenames {
		fileInfo, err := os.Stat(filename)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to stat file `%s`; %v\n", filename, err)
			return 2
		}

		if fileInfo.IsDir() {
			entries, err := os.ReadDir(filename)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to read directory `%s`; %v\n", filename, err)
				return 2
			}

			filePaths := make([]string, 0)
			for _, entry := range entries {
				filePath := filepath.Join(filename, entry.Name())
				info, err := os.Stat(filePath)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Failed to stat file `%s`; %v\n", filePath, err)
					return 2
				}

				if info.IsDir() || isTerraformFile(filePath) {
					filePaths = append(filePaths, filePath)
				}
			}

			if status := preflightRecursive(filePaths); status != 0 {
				return status
			}
			continue
		}

		src, err := os.ReadFile(filename)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to read file `%s`; %v\n", filename, err)
			return 2
		}
		_, diags := format.Format(src, filename)
		if diags.HasErrors() {
			printDiagnostics(diags)
			return 2
		}
	}

	return 0
}

func printDiagnostics(diags hcl.Diagnostics) {
	for _, diag := range diags {
		diagType := "ERROR"
		if diag.Severity == hcl.DiagWarning {
			diagType = "WARNING"
		}
		fmt.Printf("[%s] %s: %s\n\n%s\n\n", diagType, diag.Subject, diag.Summary, diag.Detail)
	}
}

func isTerraformFile(filename string) bool {
	for _, ext := range supportedExts {
		if strings.HasSuffix(filename, ext) {
			return true
		}
	}
	return false
}
