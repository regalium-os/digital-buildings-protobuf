// Copyright 2026 RegaliumOS™.
// SPDX-License-Identifier: Apache-2.0

package docs

import (
	"strings"
)

// table renders a GitHub-flavoured Markdown table. Columns are not padded:
// alignment costs a full pass over the rows and buys nothing a renderer shows,
// and an unpadded table keeps the diff on a spec bump to the rows that
// actually changed rather than to every row in the column.
func table(head []string, rows [][]string) string {
	if len(rows) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("| " + strings.Join(head, " | ") + " |\n")
	b.WriteString("|" + strings.Repeat(" --- |", len(head)) + "\n")
	for i, row := range rows {
		if i > 0 {
			b.WriteString("\n")
		}
		cells := make([]string, len(row))
		for j, c := range row {
			cells[j] = cell(c)
		}
		b.WriteString("| " + strings.Join(cells, " | ") + " |")
	}
	return b.String()
}

// cell makes a value safe inside a table. A pipe would end the column and a
// newline would end the row, and the ontology's prose contains both.
func cell(s string) string {
	s = strings.ReplaceAll(s, "|", "\\|")
	s = strings.Join(strings.Fields(s), " ")
	if s == "" {
		return "--"
	}
	return s
}

// code wraps an identifier in backticks.
func code(s string) string {
	if s == "" {
		return "--"
	}
	return "`" + s + "`"
}

// sentence trims the ontology's own prose to one sentence, for a table cell.
// The full description keeps its own paragraph elsewhere on the page.
func sentence(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	if i := strings.Index(s, ". "); i >= 0 {
		return s[:i+1]
	}
	return s
}

// plural is "1 field" / "2 fields", so counts read as English.
func plural(n int, one, many string) string {
	if n == 1 {
		return "1 " + one
	}
	return itoa(n) + " " + many
}

// itoa with thousands separators: a 1,587 reads faster than a 1587, and these
// counts are the numbers a reader is most likely to compare.
func itoa(n int) string {
	s := ""
	neg := n < 0
	if neg {
		n = -n
	}
	for {
		part := n % 1000
		n /= 1000
		if n == 0 {
			s = digits(part) + s
			break
		}
		s = "," + pad3(part) + s
	}
	if neg {
		return "-" + s
	}
	return s
}

func digits(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}

func pad3(n int) string {
	s := digits(n)
	return strings.Repeat("0", 3-len(s)) + s
}
