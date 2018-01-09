package main

import "github.com/tajtiattila/cmdmain"

var cli struct {
	verbose bool
	inacc   bool
}

func main() {
	cmdmain.Globals.BoolVar(&cli.verbose, "v", false, "verbose mode")
	cmdmain.Globals.BoolVar(&cli.inacc, "inacc", false, "skip accuracy checks when loading tracks")
	cmdmain.Main()
}
