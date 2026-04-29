package main

import "github.com/bssn1337/dorkscan/cmd"

var version = "dev"

func main() {
	cmd.SetVersion(version)
	cmd.Execute()
}
