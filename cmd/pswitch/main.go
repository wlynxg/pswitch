package main

import (
	"fmt"
	"os"
)

func main() {
	if err := runServe(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "pswitch: %v\n", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `pswitch

Usage:
  pswitch [--listen ADDR] [--mode sequential|round_robin|least_failures] [--failure-threshold N] [--cooldown DURATION] [--health-check-interval DURATION] [--health-check-timeout DURATION] [--log-color[=true|false]]`)
}
