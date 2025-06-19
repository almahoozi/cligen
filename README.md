# cligen - CLI Generator for Go

`cligen` is a Go code generator that creates CLI application skeletons from command input structs and struct tags. It automatically generates flag parsing, validation, help text, and command handling using the `pflag` library.

## Installation

```bash
go install github.com/almahoozi/cligen@latest
```

## Usage

### Basic Usage

Add a `//go:generate` comment above your struct definition:

```go
//go:generate cligen serve "Starts an HTTP server"
type ServeCLIArgs struct {
    Port int    `cli:"port,p,default:8080"`
    Env  string `cli:"env,e,required,options:dev|staging|prod|local"`
}
```

Then run:

```bash
go generate
```

This will generate a `cmd_serve.go` file with a complete CLI application.

### Struct Tag Format

The `cli` struct tag supports the following options:

- **Flag name**: First parameter (e.g., `port`)
- **Short flag**: Single character (e.g., `p` for `-p`)
- **default:value**: Set default value (e.g., `default:8080`)
- **required**: Mark field as required
- **options:val1|val2**: Restrict to specific values

### Examples

#### Simple Server Command

```go
//go:generate cligen serve "Starts an HTTP server"
type ServeCLIArgs struct {
    Port int    `cli:"port,p,default:8080"`
    Env  string `cli:"env,e,required,options:dev|staging|prod|local"`
}
```

Generated usage:
```bash
$ ./cmd_serve --help
Starts an HTTP server

Usage of ./cmd_serve:
  -e, --env string      env (required) [dev|staging|prod|local]
  -p, --port int        port (default 8080)

$ ./cmd_serve --env dev --port 3000
Executing serve command with args: &{Port:3000 Env:dev}
```

#### Build Command with Multiple Types

```go
//go:generate cligen --command=build --help="Builds the application"
type BuildCLIArgs struct {
    Output   string   `cli:"output,o,default:./dist"`
    Verbose  bool     `cli:"verbose,v"`
    Tags     []string `cli:"tags,t"`
    Platform string   `cli:"platform,required,options:linux|darwin|windows"`
}
```

Generated usage:
```bash
$ ./cmd_build --help
Builds the application

Usage of ./cmd_build:
  -o, --output string      output (default "./dist")
      --platform string    platform (required) [linux|darwin|windows]
  -t, --tags strings       tags
  -v, --verbose            verbose

$ ./cmd_build --platform linux --verbose --tags=release,prod
Executing build command with args: &{Output:./dist Verbose:true Tags:[release prod] Platform:linux}
```

### Command Line Formats

You can use either format:

1. **Short form**: `//go:generate cligen <command> "<description>"`
2. **Long form**: `//go:generate cligen --command=<command> --help="<description>"`

### Supported Types

- `string` - String flags
- `int` - Integer flags  
- `bool` - Boolean flags
- `[]string` - String slice flags (comma-separated)

### Features

- ✅ Automatic flag parsing with `pflag`
- ✅ Short and long flag support
- ✅ Default values
- ✅ Required field validation
- ✅ Options validation (enum-like)
- ✅ Help text generation
- ✅ Type-safe command structures
- ✅ Customizable output files

### Generated Code Structure

The generated code includes:

1. **Command struct** - Holds all the parsed flags
2. **Execute method** - Placeholder for your command logic
3. **NewCommand function** - Sets up flags and validation
4. **main function** - Entry point with help handling

### Customizing Generated Code

After generation, you can modify the `Execute` method to implement your command logic:

```go
// Execute runs the serve command
func (c *ServeCommand) Execute() error {
    // Your implementation here
    server := &http.Server{
        Addr: fmt.Sprintf(":%d", c.Port),
    }
    
    log.Printf("Starting server on port %d in %s environment", c.Port, c.Env)
    return server.ListenAndServe()
}
```

## Dependencies

- `github.com/spf13/pflag` - Advanced flag parsing (automatically added to generated code)

## License

MIT License 