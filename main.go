package main

import (
	"flag"
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
		flag.StringVar(&command, "command", "", "Command name")
		flag.StringVar(&help, "help", "", "Help description")
		flag.StringVar(&outputFile, "output", "", "Output file (default: cmd_<command>.go)")
		flag.Parse()
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
		outputFile = fmt.Sprintf("cmd_%s.go", command)
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
