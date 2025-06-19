package main

//go:generate cligen serve "Starts an HTTP server"
type ServeCLIArgs struct {
	Port int    `cli:"port,p,default:8080,usage:Port to listen on"`
	Env  string `cli:"env,e,required,options:dev|staging|prod|local,usage:Environment to run in"`
}

//go:generate cligen --command=build --help="Builds the application"
type BuildCLIArgs struct {
	Output   string   `cli:"output,o,default:./dist,usage:Output directory for build artifacts"`
	Verbose  bool     `cli:"verbose,v,usage:Enable verbose output"`
	Tags     []string `cli:"tags,t,usage:Build tags to include (comma-separated)"`
	Platform string   `cli:"platform,required,options:linux|darwin|windows,usage:Target platform for build"`
}
