package main

import "log"

func main() {
	if err := run(); err != nil {
		log.Fatalf("caldo exited with error: %v", err)
	}
}
