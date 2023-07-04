package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"

	"github.com/profiprog/res/filter"
)

var fullOutput bool
var accpetMoreOptions bool
var files []string
var filters []*filter.ResourceFilter
var colorOutput bool
var showFileRef bool

func matchAnyFilter(doc yaml.MapSlice) bool {
	if len(filters) == 0 {
		return true
	}
	for _, filter := range filters {
		if filter.Match(doc) {
			return true
		}
	}
	return false
}

func isDir(name string) bool {
	if info, err := os.Stat(name); err == nil {
		return info.IsDir()
	}
	return false
}

func isStdoutTerminal() bool {
	info, err := os.Stdout.Stat()
	return err == nil && (info.Mode()&os.ModeCharDevice != 0)
}

func appendFiles(files []string, file string) []string {
	// check if file is a directory
	if isDir(file) {
		filepath.Walk(file, func(path string, info os.FileInfo, err error) error {
			if err == nil && !info.IsDir() && (strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml")) {
				files = append(files, path)
			}
			return nil
		})
		if len(files) == 0 {
			fmt.Fprintf(os.Stderr, "error: no yaml files found in directory %s\n", file)
			os.Exit(1)
		}
		// check if file exists
	} else if _, err := os.Stat(file); err == nil {
		files = append(files, file)
	} else {
		fmt.Fprintf(os.Stderr, "error: %s: no such file or directory\n", file)
		os.Exit(1)
	}
	return files
}

func pritnHelpAndExit() {
	fmt.Printf("\x1b[1;4mUSAGE:\x1b[0m %s [-h] [-n] [-c] [-] [-i=FILE ...] [PATTERN ...]\n", os.Args[0])
	fmt.Printf("  -h, --help             display this help and exit\n")
	fmt.Printf("  -                      list maching resources without printing yaml content\n")
	fmt.Printf("  -n                     suppress comments referencing sources file\n")
	fmt.Printf("  -c                     suppress colors in output\n")
	fmt.Printf("  -i=<FILE>, -i <FILE>   input files or dirs, instead of stdin\n")
	fmt.Printf("  --                     stop processing options\n")
	fmt.Printf("\n\x1b[1;4mPATTERN:\x1b[0m\n")
	fmt.Printf(" * All paterns are comparing case insensitive. Hovever if patern does not\n")
	fmt.Printf("   contains \"/\" then is consider to matching kind if starts with cappital letter\n")
	fmt.Printf("   othervise is matching name.\n")
	fmt.Printf(" * Prefix \"^\" says that patern must match from begining,\n")
	fmt.Printf(" * suffix \"$\" says that patern must match to end.\n")
	fmt.Printf("\nExamples:\n")
	fmt.Printf("  \x1b[35mClass class\x1b[0m/           match if kind of resource contains \"class\"\n")
	fmt.Printf("  \x1b[33mname\x1b[0m /\x1b[33mName\x1b[0m             match if name of resource contains \"name\"\n")
	fmt.Printf("  \x1b[35mclass\x1b[0m/\x1b[33mname\x1b[0m \x1b[35mClass\x1b[0m/\x1b[33mName\x1b[0m  match if kind contains \"class\" and name contains \"name\"\n")
	fmt.Printf("  ^\x1b[35mPod\x1b[0m$ ^\x1b[35mpod\x1b[0m$/           match if kind equals to \"pod\"\n")
	fmt.Printf("  ^\x1b[33mname\x1b[0m$                 match if name equals to \"name\"\n")
	fmt.Printf("  ^\x1b[35mPod\x1b[0m ^\x1b[35mpod\x1b[0m/             match if kind starts with \"pod\"\n")
	fmt.Printf("  \x1b[33m-sufix\x1b[0m$                match if name ends with \"-sufix\"\n")
	fmt.Printf("  \x1b[35mclass\x1b[0m$/^\x1b[33mprefix-\x1b[0m        match if kind ends with \"class\" and name starts with \"prefix-\"\n")
	fmt.Printf("  /                      match all resources (same as empty pattern \"\")\n")
	os.Exit(0)
}

func init() {
	fullOutput = true
	accpetMoreOptions = true
	files = []string{}
	filters = []*filter.ResourceFilter{}
	colorOutput = isStdoutTerminal()
	showFileRef = true

	// iterate over command line arguments
	for i := 1; i < len(os.Args); i++ {
		if accpetMoreOptions {
			if os.Args[i] == "--" {
				accpetMoreOptions = false
				continue
			}
			if os.Args[i] == "-" {
				fullOutput = false
				continue
			}
			if os.Args[i] == "-h" || os.Args[i] == "--help" {
				pritnHelpAndExit()
			}
			if os.Args[i] == "-c" {
				colorOutput = false
				continue
			}
			if os.Args[i] == "-n" {
				showFileRef = false
				continue
			}
			if os.Args[i] == "-i" {
				i++
				files = appendFiles(files, os.Args[i])
				continue
			}
			if len(os.Args[i]) > 3 && os.Args[i][:3] == "-i=" {
				files = appendFiles(files, os.Args[i][3:])
				continue
			}
		}
		filters = append(filters, filter.NewResourceFilter(os.Args[i]))
	}

	if len(files) == 0 {
		files = append(files, "")
	}
	if len(files) == 1 {
		showFileRef = false
	}
}

func main() {
	matchCount := 0

	for _, file := range files {
		var decoder *yaml.Decoder
		if file == "" {
			// parse stream of yamls from stdin
			decoder = yaml.NewDecoder(os.Stdin)
		} else {
			// parse yaml file
			f, err := os.Open(file)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: %s: %s\n", file, err)
				os.Exit(1)
			}
			defer f.Close()
			decoder = yaml.NewDecoder(f)
		}
		for {
			doc := yaml.MapSlice{}
			if err := decoder.Decode(&doc); err != nil {
				if err == io.EOF {
					break
				}
				fmt.Fprintf(os.Stderr, "error: document decode failed: %s\n", err)
				os.Exit(1)
			}
			if isMaching := matchAnyFilter(doc); isMaching {
				matchCount++
				if fullOutput {
					d, err := yaml.Marshal(&doc)
					if err != nil {
						fmt.Fprintf(os.Stderr, "error: document encode failed: %s\n", err)
						os.Exit(1)
					}
					fmt.Println("---")
					if showFileRef {
						fmt.Printf("# file: %s\n", file)
					}
					fmt.Print(string(d))
				} else {
					printMachedResource(doc, file)
				}
			}
		}
	}
}

func printMachedResource(doc yaml.MapSlice, file string) {
	if colorOutput && showFileRef {
		fmt.Printf("\x1b[0;35m%s\x1b[0;2m/\x1b[0;33m%s\t\x1b[0;2;3;32m# %s\x1b[0m\n", colorKind(doc), colorName(doc), file)
	} else if colorOutput {
		fmt.Printf("\x1b[0;35m%s\x1b[0;2m/\x1b[0;33m%s\x1b[0m\n", colorKind(doc), colorName(doc))
	} else if showFileRef {
		fmt.Printf("%s/%s\t# %s\n", filter.GetKind(doc), filter.GetName(doc), file)
	} else {
		fmt.Printf("%s/%s\n", filter.GetKind(doc), filter.GetName(doc))
	}
}

func colorKind(doc yaml.MapSlice) string {
	kind := filter.GetKind(doc)
	result := kind
	rank := 0
	for _, filter := range filters {
		if s, b := filter.KindHighliter(kind); b > rank {
			result = s
			rank = b
		}
	}
	return result
}

func colorName(doc yaml.MapSlice) string {
	name := filter.GetName(doc)
	result := name
	rank := 0
	for _, filter := range filters {
		if s, b := filter.NameHighliter(name); b > rank {
			result = s
			rank = b
		}
	}
	return result
}
