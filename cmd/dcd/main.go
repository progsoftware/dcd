package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) == 2 && os.Args[1] == "image-usage-message" {
		fmt.Println("This image should be used as a base image, not run directly - see README.md for more information.")
		os.Exit(1)
	}
	fmt.Println("Hello, World!")
}
