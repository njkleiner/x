// Program rfc3339 prints the current timestamp in RFC 3339 format.
package main

import (
	"flag"
	"fmt"
	"os"
	"time"
)

var (
	zulu = flag.Bool("u", false, "convert time zone to UTC")
)

func main() {
	flag.Parse()

	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	t := time.Now()

	if *zulu {
		t = t.UTC()
	}

	fmt.Println(t.Format(time.RFC3339))

	return nil
}
