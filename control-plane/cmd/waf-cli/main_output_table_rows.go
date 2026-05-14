package main

func buildTableRows(items []map[string]any, columns []string) [][]string {
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		row := make([]string, 0, len(columns))
		for _, column := range columns {
			row = append(row, stringify(item[column]))
		}
		rows = append(rows, row)
	}
	return rows
}
