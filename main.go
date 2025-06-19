package main

import (
	"fmt"
	"log"
	"os"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	// Parse command line arguments
	var command, help string
	var outputFile string

	// Handle both long and short forms
	if strings.HasPrefix(os.Args[1], "--command=") {
		// Long form: --command=serve --help="description"
		for i := 1; i < len(os.Args); i++ {
			arg := os.Args[i]
			if strings.HasPrefix(arg, "--command=") {
				command = strings.TrimPrefix(arg, "--command=")
			} else if strings.HasPrefix(arg, "--help=") {
				help = strings.TrimPrefix(arg, "--help=")
				// Handle case where quoted argument is split across multiple args
				if strings.HasPrefix(help, `"`) && !strings.HasSuffix(help, `"`) {
					// Collect remaining parts until we find the closing quote
					for j := i + 1; j < len(os.Args); j++ {
						help += " " + os.Args[j]
						if strings.HasSuffix(os.Args[j], `"`) {
							i = j // Skip the args we've consumed
							break
						}
					}
				}
				// Remove surrounding quotes if present
				if strings.HasPrefix(help, `"`) && strings.HasSuffix(help, `"`) {
					help = strings.Trim(help, `"`)
				}
			} else if strings.HasPrefix(arg, "--output=") {
				outputFile = strings.TrimPrefix(arg, "--output=")
			}
		}
	} else {
		// Short form: serve "description"
		if len(os.Args) < 3 {
			printUsage()
			os.Exit(1)
		}
		command = os.Args[1]
		help = os.Args[2]
		if len(os.Args) > 3 {
			outputFile = os.Args[3]
		}
	}

	if command == "" {
		log.Fatal("Command name is required")
	}

	if outputFile == "" {
		outputFile = fmt.Sprintf("cmd/%s/main.go", command)
	}

	// Get the source file from GOFILE environment variable (set by go generate)
	sourceFile := os.Getenv("GOFILE")
	if sourceFile == "" {
		log.Fatal("GOFILE environment variable not set. This tool should be run via go generate")
	}

	// Parse the source file and generate CLI code
	generator := &Generator{
		SourceFile: sourceFile,
		Command:    command,
		Help:       help,
		OutputFile: outputFile,
	}

	if err := generator.Generate(); err != nil {
		log.Fatalf("Failed to generate CLI code: %v", err)
	}

	fmt.Printf("Generated CLI code in %s\n", outputFile)
}

func printUsage() {
	fmt.Println("Usage:")
	fmt.Println("  cligen --command=<name> --help=\"<description>\" [--output=<file>]")
	fmt.Println("  cligen <command> \"<description>\" [output_file]")
	fmt.Println()
	fmt.Println("This tool should be run via go generate with a comment like:")
	fmt.Println("  //go:generate cligen serve \"Starts an http server\"")
}
