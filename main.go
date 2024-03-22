package main

import (
	"fmt"
	"log"
)

// nolint:forbidigo
func usage() {
	fmt.Println("Usage: gh extension-template --flag value (--flag value)")
	fmt.Println("example: gh extension-template --flag value (--flag value) // Description")
}

func run() error {
	fmt.Println("Hello, World!")
	return nil
}

func main() {

	err := run()
	if err != nil {
		log.Fatal(err) //nolint:forbidigo
	}
}
