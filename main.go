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
	rawbaseURL := args[1]
	fmt.Printf("starting crawl of: \"%s\"\n", rawbaseURL)
	// 	inputURL: "https://www.boot.dev/blog/path",
	// expected: "www.boot.dev/blog/path",
	// resp, err := getHTML(baseURL)
	// if err != nil {
	// 	fmt.Printf("failed %s\n", err)
	// 	os.Exit(1)
	// }
	// fmt.Print(resp)
	pages := make(map[string]int)
	baseURL, _ := NormalizeURL(rawbaseURL)
	crawlPage(baseURL, rawbaseURL, pages)
	fmt.Println(pages)

}
