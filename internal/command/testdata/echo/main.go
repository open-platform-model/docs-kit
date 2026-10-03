// Command echo is the test helper of package command: its first argument
// picks what it prints.
package main

import (
	"fmt"
	"os"
	"strings"
	"time"
)

func main() {
	switch os.Args[1] {
	case "ok":
		fmt.Println(`{"schema": "test/v1", "value": 1}`)
	case "env":
		fmt.Printf(`{"schema": "test/v1", "env": %q, "pwd": %q}`+"\n",
			strings.Join([]string{os.Getenv("OPM_DOCS"), os.Getenv("OPM_DOCS_PROJECT"), os.Getenv("OPM_DOCS_VERSION")}, ","), cwd())
	case "fail":
		fmt.Fprintln(os.Stderr, "failing")
		os.Exit(1)
	case "trailing":
		fmt.Println(`{"schema": "test/v1"}`)
		fmt.Println("a log line")
	case "schema":
		fmt.Println(`{"schema": "other/v1"}`)
	case "sleep":
		time.Sleep(time.Minute)
	case "big":
		fmt.Printf(`{"schema": "test/v1", "pad": %q}`+"\n", strings.Repeat("x", 4096))
	case "random":
		fmt.Printf(`{"schema": "test/v1", "now": %d}`+"\n", time.Now().UnixNano())
	}
}

func cwd() string {
	d, _ := os.Getwd()
	return d
}
