package main

func renderTableRows(rows [][]string, printRow func([]string)) {
	for _, row := range rows {
		printRow(row)
	}
}
