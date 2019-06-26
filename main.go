package main

import (
	"os"
)

func main() {
	if testFlagPresence("-client") {
		startClient()
	} else {
		startServer()
	}
}

func testFlagPresence(f string) bool {
	for _, fl := range os.Args {
		if fl == f {
			return true
		}
	}

	return false
}
