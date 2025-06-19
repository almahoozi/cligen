package main

//go:generate cligen serve "Starts an HTTP server"
type ServeCLIArgs struct {
	Port int    `cli:"port,p,default:8080"`
	Env  string `cli:"env,e,required,options:dev|staging|prod|local"`
}

//go:generate cligen --command=build --help="Builds the application"
type BuildCLIArgs struct {
	Output   string   `cli:"output,o,default:./dist"`
	Verbose  bool     `cli:"verbose,v"`
	Tags     []string `cli:"tags,t"`
	Platform string   `cli:"platform,required,options:linux|darwin|windows"`
}
