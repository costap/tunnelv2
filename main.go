package main

import "github.com/costap/tunnelv2/cmd"

var (
	version = "local"
	commit  = ""
	date    = ""
	builtBy = ""
)

func init() {
	cmd.Version = version
}

func main() {
	cmd.Execute()
}
