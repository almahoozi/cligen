package main

import (
	"embed"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"reflect"
	"strings"
	"text/template"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

//go:embed templates/*.tmpl
var templateFS embed.FS

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
	Usage        string // New field for per-option help
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
		} else if strings.HasPrefix(part, "usage:") {
			field.Usage = strings.TrimPrefix(part, "usage:")
		}
	}

	return field
}

// extractTag extracts a specific tag from a struct tag string
func (g *Generator) extractTag(tag, key string) string {
	// Use Go's reflect.StructTag for proper parsing
	structTag := reflect.StructTag(tag)
	return structTag.Get(key)
}

// generateCLICode generates the CLI code using templates
func (g *Generator) generateCLICode(structName string, fields []FieldInfo) error {
	// Create directory if it doesn't exist
	if err := os.MkdirAll(strings.TrimSuffix(g.OutputFile, "/main.go"), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	caser := cases.Title(language.English)

	// Load CLI template from embedded file
	cliTemplateContent, err := templateFS.ReadFile("templates/cli.go.tmpl")
	if err != nil {
		return fmt.Errorf("failed to read CLI template: %w", err)
	}

	tmpl := template.Must(template.New("cli").Funcs(template.FuncMap{
		"title": caser.String,
		"join":  strings.Join,
	}).Parse(string(cliTemplateContent)))

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
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close file %s: %v\n", g.OutputFile, closeErr)
		}
	}()

	if err := tmpl.Execute(file, data); err != nil {
		return err
	}

	// Generate go.mod file for the command
	if err := g.generateGoMod(); err != nil {
		return err
	}

	// Generate implementation stub file if it doesn't exist
	return g.generateImplementationStub(structName, fields)
}

// generateGoMod creates a go.mod file for the command
func (g *Generator) generateGoMod() error {
	dir := strings.TrimSuffix(g.OutputFile, "/main.go")
	goModPath := fmt.Sprintf("%s/go.mod", dir)

	goModContent := fmt.Sprintf(`module %s

go 1.24

require github.com/spf13/pflag v1.0.6
`, g.Command)

	return os.WriteFile(goModPath, []byte(goModContent), 0644)
}

// generateImplementationStub creates an implementation stub file if it doesn't exist
func (g *Generator) generateImplementationStub(structName string, fields []FieldInfo) error {
	dir := strings.TrimSuffix(g.OutputFile, "/main.go")
	implPath := fmt.Sprintf("%s/%s_impl.go", dir, g.Command)

	// Don't overwrite existing implementation
	if _, err := os.Stat(implPath); err == nil {
		return nil // File already exists, don't overwrite
	}

	caser := cases.Title(language.English)

	// Load implementation template from embedded file
	implTemplateContent, err := templateFS.ReadFile("templates/impl.go.tmpl")
	if err != nil {
		return fmt.Errorf("failed to read implementation template: %w", err)
	}

	tmpl := template.Must(template.New("impl").Funcs(template.FuncMap{
		"title": caser.String,
	}).Parse(string(implTemplateContent)))

	data := struct {
		Command    string
		StructName string
		Fields     []FieldInfo
	}{
		Command:    g.Command,
		StructName: structName,
		Fields:     fields,
	}

	file, err := os.Create(implPath)
	if err != nil {
		return fmt.Errorf("failed to create implementation file: %w", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close file %s: %v\n", implPath, closeErr)
		}
	}()

	return tmpl.Execute(file, data)
}
