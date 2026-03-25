package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/goccy/go-yaml/lexer"
	"github.com/goccy/go-yaml/printer"
	"github.com/profiprog/res/filter"
	"github.com/profiprog/res/version"
)

var fullOutput bool
var accpetMoreOptions bool
var files []string
var filters []*filter.ResourceFilter
var colorOutput bool
var showFileRef bool
var sortOutput bool
var normalizeOutput bool

func matchAnyFilter(doc *yaml.Node) bool {
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
	fmt.Printf("  -v, --version          prints version and exit\n")
	fmt.Printf("  -                      list maching resources without printing yaml content\n")
	fmt.Printf("  -n                     suppress comments referencing sources file\n")
	fmt.Printf("  -N                     Normalize output (remove unecessary quotas, comments, ensure list as bulleted syntax)\n")
	fmt.Printf("  -c                     suppress colors in output\n")
	fmt.Printf("  -C                     always use colors in output\n")
	fmt.Printf("  -s                     sort resources and keys\n")
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
	sortOutput = false

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
			if os.Args[i] == "-v" || os.Args[i] == "--version" {
				fmt.Printf("%s version %s\n", os.Args[0], version.Version)
				os.Exit(0)
			}
			if os.Args[i] == "-c" {
				colorOutput = false
				continue
			}
			if os.Args[i] == "-C" {
				colorOutput = true
				continue
			}
			if os.Args[i] == "-s" {
				sortOutput = true
				continue
			}
			if os.Args[i] == "-N" {
				normalizeOutput = true
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
}

func main() {
	nextComment := ""
	matchCount := 0
	keys := []string{}
	basket := make(map[string]interface{})

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
			doc := yaml.Node{}
			if err := decoder.Decode(&doc); err != nil {
				if err == io.EOF {
					break
				}
				fmt.Fprintf(os.Stderr, "error: document decode failed: %s\n", err)
				os.Exit(1)
			}
			if doc.Content[0].Kind == yaml.ScalarNode {
				nextComment = doc.FootComment
				continue
			}
			if isMaching := matchAnyFilter(&doc); isMaching {
				matchCount++
				source := extractCommentSource(&doc, nextComment)
				if sortOutput {
					key := fmt.Sprintf("%s/%s", filter.GetKind(&doc), filter.GetName(&doc))
					keys = append(keys, key)
					if normalizeOutput {
						normalizeNode(&doc)
					}
					if fullOutput {
						sortNode(&doc)
						basket[key] = formatYaml(&doc, file, source)
					} else {
						basket[key] = printMachedResource(&doc, file, source)
					}
				} else if fullOutput {
					fmt.Print(formatYaml(&doc, file, source))
				} else {
					fmt.Print(printMachedResource(&doc, file, source))
				}
			}
			nextComment = ""
		}
	}
	if sortOutput {
		sort.Strings(keys)
		for _, key := range keys {
			fmt.Print(basket[key])
		}
	}
}

func formatRefs(file string, source string) string {
	result := ""
	if file != "" {
		result = fmt.Sprintf("%s# file: %s\n", result, file)
	}
	if source != "" && source != file {
		result = fmt.Sprintf("%s# source: %s\n", result, source)
	}
	return result
}

func extractCommentSource(node *yaml.Node, preComment string) string {
	lines := []string{}
	result := ""
	if preComment != "" {
		for _, l := range strings.Split(preComment, "\n") {
			if l != "" {
				lines = append(lines, l)
			}
		}
	}
	for node.HeadComment == "" && (node.Kind == yaml.DocumentNode || node.Kind == yaml.MappingNode) && len(node.Content) > 0 {
		node = node.Content[0]
	}
	if node.HeadComment != "" {
		for _, l := range strings.Split(node.HeadComment, "\n") {
			if l != "" {
				lines = append(lines, l)
			}
		}
		node.HeadComment = ""
	}
	if len(lines) > 0 {
		rest := []string{}
		for _, line := range lines {
			if strings.HasPrefix(line, "# Source: ") {
				result = line[10:]
			} else {
				rest = append(rest, line)
			}
		}
		if len(rest) > 0 {
			node.HeadComment = strings.Join(rest, "\n")
		}
	}
	return result
}

func formatYaml(doc *yaml.Node, file string, source string) string {
	bytes, err := yaml.Marshal(doc)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: document encode failed: %s\n", err)
		os.Exit(1)
	}
	yaml := string(bytes)
	if colorOutput {
		tokens := lexer.Tokenize(yaml)
		var p printer.Printer
		p.Bool = func() *printer.Property {
			return &printer.Property{
				Prefix: "\x1b[0;33m",
				Suffix: "\x1b[0m",
			}
		}
		p.Number = func() *printer.Property {
			return &printer.Property{
				Prefix: "\x1b[0;35m",
				Suffix: "\x1b[0m",
			}
		}
		p.MapKey = func() *printer.Property {
			return &printer.Property{
				Prefix: "\x1b[0;36m",
				Suffix: "\x1b[0m",
			}
		}
		p.String = func() *printer.Property {
			return &printer.Property{
				Prefix: "\x1b[0;32m",
				Suffix: "\x1b[0m",
			}
		}
		yaml = p.PrintTokens(tokens) + "\n"
	}
	if showFileRef {
		return fmt.Sprintf("---\n%s%s", formatRefs(file, source), yaml)
	}
	return fmt.Sprintf("---\n%s", yaml)
}

func printMachedResource(doc *yaml.Node, file string, source string) string {
	ref := ""
	if showFileRef {
		if file != "" {
			ref = fmt.Sprintf("file: %s", file)
		} else if source != "" {
			ref = fmt.Sprintf("source: %s", source)
		}
	}
	if colorOutput && ref != "" {
		return fmt.Sprintf("\x1b[0;35m%s\x1b[0;2m/\x1b[0;33m%s\t\x1b[0;2;3;32m# %s\x1b[0m\n", colorKind(doc), colorName(doc), ref)
	} else if colorOutput {
		return fmt.Sprintf("\x1b[0;35m%s\x1b[0;2m/\x1b[0;33m%s\x1b[0m\n", colorKind(doc), colorName(doc))
	} else if ref != "" {
		return fmt.Sprintf("%s/%s\t# %s\n", filter.GetKind(doc), filter.GetName(doc), ref)
	} else {
		return fmt.Sprintf("%s/%s\n", filter.GetKind(doc), filter.GetName(doc))
	}
}

func colorKind(doc *yaml.Node) string {
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

func colorName(doc *yaml.Node) string {
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

func sortNode(node *yaml.Node) {
	// If it's a document, sort its root content
	if node.Kind == yaml.DocumentNode {
		for _, child := range node.Content {
			sortNode(child)
		}
		return
	}

	// If it's a map, sort the key/value pairs
	if node.Kind == yaml.MappingNode {
		// Create a temporary slice of indices for the pairs
		type pair struct {
			key   *yaml.Node
			value *yaml.Node
		}
		pairs := make([]pair, len(node.Content)/2)
		for i := 0; i < len(node.Content); i += 2 {
			pairs[i/2] = pair{node.Content[i], node.Content[i+1]}
		}

		// Sort pairs by the key's string value
		sort.Slice(pairs, func(i, j int) bool {
			return pairs[i].key.Value < pairs[j].key.Value
		})

		// Reconstruct the flattened Content slice
		newContent := make([]*yaml.Node, 0, len(node.Content))
		for _, p := range pairs {
			newContent = append(newContent, p.key, p.value)
		}
		node.Content = newContent
	}

	// Recursively sort children (items in a list or values in a map)
	for _, child := range node.Content {
		sortNode(child)
	}
}

func normalizeNode(n *yaml.Node) {
	n.HeadComment = ""
	n.LineComment = ""
	n.FootComment = ""
	switch n.Kind {
	case yaml.ScalarNode:
		n.Style = 0
	case yaml.SequenceNode:
		n.Style = 0
	}
	for _, child := range n.Content {
		normalizeNode(child)
	}
}
