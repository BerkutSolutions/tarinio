package main

import "fmt"

func printTableCell(width int, value string) {
	fmt.Printf("%-*s", width, value)
}
