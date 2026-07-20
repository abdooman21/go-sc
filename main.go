package main

import (
	"fmt"
	"os"
)

func main() {

	args := os.Args
	if len(args) != 2 { // outer condition to not repeat operations for correct programs which represent majorities of programs
		if len(args) < 2 {
			fmt.Printf("no Args, provide url \n")
			os.Exit(1)
		} else if len(args) > 2 {
			fmt.Printf("Too many args were provided only URL needed     #FOR_CURRENT_VERSION\n")
			os.Exit(1)
		}
	}
	baseURL := args[1]
	fmt.Printf("starting crawl of: \"%s\"\n", baseURL)
	// 	inputURL: "https://www.boot.dev/blog/path",
	// expected: "www.boot.dev/blog/path",

}
