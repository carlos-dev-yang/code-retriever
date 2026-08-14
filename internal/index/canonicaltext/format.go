// Package canonicaltext owns the Phase 05 v1 document-input byte contract.
package canonicaltext

import (
	"bytes"
	"fmt"
	"strings"
)

type Input struct {
	Path, Kind, QualifiedSymbol, Signature string
	BodyParts                              [][]byte
}

func Format(in Input) ([]byte, error) {
	if in.Path == "" || strings.ContainsAny(in.Path, "\r\n\x00") {
		return nil, fmt.Errorf("canonical path is ambiguous")
	}
	if in.Kind == "" || in.QualifiedSymbol == "" {
		return nil, fmt.Errorf("canonical identity is required")
	}
	var b bytes.Buffer
	for _, field := range []struct{ label, value string }{{"path", in.Path}, {"kind", in.Kind}, {"symbol", in.QualifiedSymbol}, {"signature", in.Signature}} {
		b.WriteString(field.label)
		b.WriteString(": ")
		b.Write(normalize([]byte(field.value)))
		b.WriteByte('\n')
	}
	b.WriteString("body:\n")
	for i, p := range in.BodyParts {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.Write(normalize(p))
	}
	if b.Len() == 0 || b.Bytes()[b.Len()-1] != '\n' {
		b.WriteByte('\n')
	}
	return b.Bytes(), nil
}
func normalize(b []byte) []byte {
	b = bytes.ReplaceAll(b, []byte("\r\n"), []byte("\n"))
	return bytes.ReplaceAll(b, []byte("\r"), []byte("\n"))
}
