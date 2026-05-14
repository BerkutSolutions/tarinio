package main

import "fmt"

func formatFloatString(value float64) string {
	if float64(int64(value)) == value {
		return fmt.Sprintf("%d", int64(value))
	}
	return fmt.Sprintf("%.3f", value)
}
