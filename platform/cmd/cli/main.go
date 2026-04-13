package main

import (
	"errors"
	"fmt"
	"log"
	"os"
)

func main() {
	// Redirect log output to a file so it doesn't corrupt the TUI.
	if logFile, err := os.OpenFile("molecli.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600); err == nil {
		log.SetOutput(logFile)
		defer logFile.Close()
	} else {
		log.SetOutput(os.Stderr)
	}

	if err := buildRootCmd().Execute(); err != nil {
		var exitErr exitCoder
		if errors.As(err, &exitErr) {
			if msg := err.Error(); msg != "" {
				fmt.Fprintln(os.Stderr, msg)
			}
			os.Exit(exitErr.ExitCode())
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func baseURL() string {
	if u := os.Getenv("MOLECLI_URL"); u != "" {
		return u
	}
	return "http://localhost:8080"
}
