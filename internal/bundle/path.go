package bundle

import "strings"

func trimSlash(s string) string { return strings.TrimSuffix(s, "/") }

func splitSlash(s string) []string { return strings.Split(s, "/") }

func cut(s string) (before, after string, found bool) { return strings.Cut(s, "/") }
