package main

import "github.com/tajtiattila/cmdmain"

var cli struct {
	verbose bool
}

func main() {
	cmdmain.Globals.BoolVar(&cli.verbose, "v", false, "verbose mode")
	cmdmain.Main()
}
