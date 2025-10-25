package main

import (
	"flag"
	"fmt"
	"os"

	tffumpt "github.com/AleksaC/tffumpt"
)

var version = "dev" // This will be set by the linker during build

func main() {
	flag.Usage = func() {
		fmt.Print("Usage: tffumpt [options] [files...]\n\n")

		fmt.Print(
			"By default, scans the current directory for terraform files. " +
				"If you provide a directory, then it will scan that directory instead. " +
				"If you provide a file, then just that file will be processed. " +
				"If you provide a single dash ('-'), then tffumpt will read from standard input (STDIN)." +
				"\n\nOptions:\n",
		)

		flag.PrintDefaults()
	}

	list := flag.Bool("list", true, "Whether to list files whose formatting differs")
	write := flag.Bool("write", true, "Whether to write result to source file instead of stdout")
	diff := flag.Bool("diff", false, "Whether to show a diff of formatting changes")
	check := flag.Bool("check", false, "Exit with nonzero status code if formatting isn't correct")
	recursive := flag.Bool("recursive", false, "Whether to format files in subdirectories")
	versionFlag := flag.Bool("version", false, "Display version information")
	flag.Bool(
		"no-color",
		true,
		"Currently only exists for compatibility with terraform fmt as there's no color in the output anyway",
	)

	flag.Parse()

	if *versionFlag {
		fmt.Println(version)
		return
	}

	files := flag.Args()

	os.Exit(
		tffumpt.Fumpt(files, &tffumpt.Options{
			List:      *list,
			Write:     *write,
			Diff:      *diff,
			Check:     *check,
			Recursive: *recursive,
		}),
	)
}
