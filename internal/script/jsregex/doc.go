// Package jsregex owns the Go ECMAScript regular-expression engine used by the
// Script subsystem.
//
// It parses patterns into a native syntax tree first and falls back to
// regexp2 only for constructs the native tree cannot express, such as
// variable-width lookbehind.
package jsregex
