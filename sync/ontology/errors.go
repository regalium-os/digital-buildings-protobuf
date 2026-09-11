// Copyright 2026 RegaliumOS™.
// SPDX-License-Identifier: Apache-2.0

package ontology

import (
	"fmt"
	"sort"
	"strings"
)

// errorList accumulates diagnostics so one run reports every problem rather
// than the first. Rule 15: the generator fails loudly, and a bump that breaks
// forty things should say so once.
type errorList struct{ msgs []string }

func (e *errorList) addf(format string, args ...any) {
	e.msgs = append(e.msgs, fmt.Sprintf(format, args...))
}

func (e *errorList) err(summary string) error {
	if len(e.msgs) == 0 {
		return nil
	}
	sort.Strings(e.msgs)
	const max = 25
	shown := e.msgs
	extra := 0
	if len(shown) > max {
		extra = len(shown) - max
		shown = shown[:max]
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s (%d problems)", summary, len(e.msgs))
	for _, m := range shown {
		b.WriteString("\n  - ")
		b.WriteString(m)
	}
	if extra > 0 {
		fmt.Fprintf(&b, "\n  ... and %d more", extra)
	}
	return fmt.Errorf("%s", b.String())
}
