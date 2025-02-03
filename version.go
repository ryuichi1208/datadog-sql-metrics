package main

import "fmt"

var (
	version  string
	revision string
	build    string
)

func _version() {
	fmt.Printf("Version : %s\n", version)
	fmt.Printf("Revision: %s\n", revision)
	fmt.Printf("Build   : %s\n", build)
}
