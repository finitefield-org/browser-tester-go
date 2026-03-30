// Package jsregex owns the Go ECMAScript regular-expression engine used by the
// Script subsystem.
//
// It parses patterns into a native syntax tree and returns explicit
// unsupported errors for constructs the current native slice cannot express.
package jsregex
