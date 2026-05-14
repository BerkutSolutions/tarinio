package main

func renderTableHeader(headers []string, printRow func([]string), printSep func()) {
	printRow(headers)
	printSep()
}
