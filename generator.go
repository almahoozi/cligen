package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"text/template"
)

// Generator handles the parsing and code generation
type Generator struct {
	SourceFile string
	Command    string
	Help       string
	OutputFile string
}

// FieldInfo represents a CLI field with its metadata
type FieldInfo struct {
	Name         string
	Type         string
	CLIName      string
	ShortFlag    string
	DefaultValue string
	Required     bool
	Options      []string
	Help         string
}

// Generate parses the source file and generates CLI code
func (g *Generator) Generate() error {
	// Parse the Go source file
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, g.SourceFile, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("failed to parse source file: %w", err)
	}

	// Find the struct that corresponds to our command
	var targetStruct *ast.StructType
	var structName string

	ast.Inspect(node, func(n ast.Node) bool {
		if genDecl, ok := n.(*ast.GenDecl); ok && genDecl.Tok == token.TYPE {
			for _, spec := range genDecl.Specs {
				if typeSpec, ok := spec.(*ast.TypeSpec); ok {
					if structType, ok := typeSpec.Type.(*ast.StructType); ok {
						// Check if this struct has the right naming pattern
						name := typeSpec.Name.Name
						if strings.Contains(strings.ToLower(name), strings.ToLower(g.Command)) &&
							strings.Contains(strings.ToLower(name), "args") {
							targetStruct = structType
							structName = name
							return false
						}
					}
				}
			}
		}
		return true
	})

	if targetStruct == nil {
		return fmt.Errorf("could not find struct for command %s", g.Command)
	}

	// Parse struct fields and their tags
	fields, err := g.parseStructFields(targetStruct)
	if err != nil {
		return fmt.Errorf("failed to parse struct fields: %w", err)
	}

	// Generate the CLI code
	return g.generateCLICode(structName, fields)
}

// parseStructFields extracts field information from struct fields
func (g *Generator) parseStructFields(structType *ast.StructType) ([]FieldInfo, error) {
	var fields []FieldInfo

	for _, field := range structType.Fields.List {
		if len(field.Names) == 0 {
			continue // Skip embedded fields
		}

		fieldName := field.Names[0].Name
		fieldType := g.getTypeString(field.Type)

		// Parse the struct tag
		var tag string
		if field.Tag != nil {
			tag = field.Tag.Value
			tag = strings.Trim(tag, "`")
		}

		fieldInfo := g.parseFieldTag(fieldName, fieldType, tag)
		fields = append(fields, fieldInfo)
	}

	return fields, nil
}

// getTypeString converts an ast.Expr to a type string
func (g *Generator) getTypeString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.ArrayType:
		return "[]" + g.getTypeString(t.Elt)
	case *ast.StarExpr:
		return "*" + g.getTypeString(t.X)
	default:
		return "interface{}"
	}
}

// parseFieldTag parses the cli struct tag
func (g *Generator) parseFieldTag(fieldName, fieldType, tag string) FieldInfo {
	field := FieldInfo{
		Name:    fieldName,
		Type:    fieldType,
		CLIName: strings.ToLower(fieldName),
	}

	if tag == "" {
		return field
	}

	// Parse the cli tag
	cliTag := g.extractTag(tag, "cli")
	if cliTag == "" {
		return field
	}

	parts := strings.Split(cliTag, ",")
	if len(parts) > 0 && parts[0] != "" {
		field.CLIName = parts[0]
	}

	for i := 1; i < len(parts); i++ {
		part := strings.TrimSpace(parts[i])

		if len(part) == 1 {
			// Single character is a short flag
			field.ShortFlag = part
		} else if strings.HasPrefix(part, "default:") {
			field.DefaultValue = strings.TrimPrefix(part, "default:")
		} else if part == "required" {
			field.Required = true
		} else if strings.HasPrefix(part, "options:") {
			optionsStr := strings.TrimPrefix(part, "options:")
			field.Options = strings.Split(optionsStr, "|")
		}
	}

	return field
}

// extractTag extracts a specific tag from a struct tag string
func (g *Generator) extractTag(tag, key string) string {
	tagMap := make(map[string]string)

	// Simple tag parsing - split by spaces and parse key:"value" pairs
	parts := strings.Fields(tag)
	for _, part := range parts {
		if strings.Contains(part, ":") {
			keyValue := strings.SplitN(part, ":", 2)
			if len(keyValue) == 2 {
				tagKey := keyValue[0]
				tagValue := strings.Trim(keyValue[1], `"`)
				tagMap[tagKey] = tagValue
			}
		}
	}

	return tagMap[key]
}

// generateCLICode generates the CLI code using templates
func (g *Generator) generateCLICode(structName string, fields []FieldInfo) error {
	tmpl := template.Must(template.New("cli").Funcs(template.FuncMap{
		"title": strings.Title,
		"join":  strings.Join,
	}).Parse(cliTemplate))

	data := struct {
		Command    string
		Help       string
		StructName string
		Fields     []FieldInfo
	}{
		Command:    g.Command,
		Help:       g.Help,
		StructName: structName,
		Fields:     fields,
	}

	file, err := os.Create(g.OutputFile)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	return tmpl.Execute(file, data)
}

const cliTemplate = `// Code generated by cligen. DO NOT EDIT.
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/pflag"
)

// {{title .Command}}Command represents the {{.Command}} command
type {{title .Command}}Command struct {
	{{range .Fields}}{{.Name}} {{.Type}}
	{{end}}
}

// Execute runs the {{.Command}} command
func (c *{{title .Command}}Command) Execute() error {
	// TODO: Implement your command logic here
	fmt.Printf("Executing {{.Command}} command with args: %+v\n", c)
	return nil
}

// New{{title .Command}}Command creates and configures the {{.Command}} command
func New{{title .Command}}Command() *{{title .Command}}Command {
	cmd := &{{title .Command}}Command{}
	
	// Define flags
	{{range .Fields}}{{$help := .CLIName}}{{if .Required}}{{$help = printf "%s (required)" .CLIName}}{{end}}{{if .Options}}{{$help = printf "%s [%s]" $help (join .Options "|")}}{{end}}{{if eq .Type "string"}}pflag.StringVarP(&cmd.{{.Name}}, "{{.CLIName}}", "{{.ShortFlag}}", "{{.DefaultValue}}", "{{$help}}")
	{{else if eq .Type "int"}}pflag.IntVarP(&cmd.{{.Name}}, "{{.CLIName}}", "{{.ShortFlag}}", {{if .DefaultValue}}{{.DefaultValue}}{{else}}0{{end}}, "{{$help}}")
	{{else if eq .Type "bool"}}pflag.BoolVarP(&cmd.{{.Name}}, "{{.CLIName}}", "{{.ShortFlag}}", {{if .DefaultValue}}{{.DefaultValue}}{{else}}false{{end}}, "{{$help}}")
	{{else if eq .Type "[]string"}}pflag.StringSliceVarP(&cmd.{{.Name}}, "{{.CLIName}}", "{{.ShortFlag}}", {{if .DefaultValue}}[]string{{"{{.DefaultValue}}"}}{{else}}nil{{end}}, "{{$help}}")
	{{end}}{{end}}
	
	// Parse flags
	pflag.Parse()
	
	// Validate required fields
	{{range .Fields}}{{if .Required}}if cmd.{{.Name}} == {{if eq .Type "string"}}"" {{else if eq .Type "int"}}0 {{else if eq .Type "bool"}}false {{else}}nil {{end}}{
		fmt.Fprintf(os.Stderr, "Error: --%s is required\n", "{{.CLIName}}")
		pflag.Usage()
		os.Exit(1)
	}
	{{end}}{{end}}
	
	// Validate options
	{{range .Fields}}{{if .Options}}if cmd.{{.Name}} != "" {
		validOptions := []string{ {{range .Options}}"{{.}}", {{end}} }
		valid := false
		for _, opt := range validOptions {
			if cmd.{{.Name}} == opt {
				valid = true
				break
			}
		}
		if !valid {
			fmt.Fprintf(os.Stderr, "Error: --%s must be one of: %s\n", "{{.CLIName}}", strings.Join(validOptions, ", "))
			pflag.Usage()
			os.Exit(1)
		}
	}
	{{end}}{{end}}
	
	return cmd
}

func main() {
	// Check for help flags
	if len(os.Args) > 1 && (os.Args[1] == "--help" || os.Args[1] == "-h") {
		fmt.Println({{printf "%q" .Help}})
		fmt.Println()
		pflag.Usage()
		return
	}
	
	cmd := New{{title .Command}}Command()
	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
`
