//go:build !windows

package main

import "fmt"

func messageBox(title, text string) {
	fmt.Printf("%s: %s\n", title, text)
}
