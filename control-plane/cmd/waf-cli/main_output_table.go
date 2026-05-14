package main

import (
	"fmt"
)

func printTableFromMaps(items []map[string]any, columns []string) error {
	rows := buildTableRows(items, columns)
	headers := copyHeaders(columns)
	renderTable(headers, rows)
	return nil
}

func renderTable(headers []string, rows [][]string) {
	if len(headers) == 0 {
		return
	}
	width := initColumnWidths(headers)
	expandColumnWidths(width, headers, rows)

	printRow := func(cols []string) {
		for i := 0; i < len(headers); i++ {
			value := tableCellValue(cols, i)
			printTableCell(width[i], value)
			fmt.Print(tableBetweenColumns(i, len(headers)))
		}
		printTableLineBreak()
	}
	printSep := func() {
		for i := range headers {
			fmt.Print(tableSeparatorCell(width[i]))
			fmt.Print(tableBetweenColumns(i, len(headers)))
		}
		printTableLineBreak()
	}

	renderTableHeader(headers, printRow, printSep)
	renderTableRows(rows, printRow)
}
