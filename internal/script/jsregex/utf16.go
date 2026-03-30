package jsregex

import "unicode/utf8"

// UTF16Length reports the number of UTF-16 code units required to encode s.
func UTF16Length(s string) int {
	units := 0
	for _, r := range s {
		units += utf16UnitsForRune(r)
	}
	return units
}

// UTF16IndexToByteOffset maps a UTF-16 code-unit index to a byte offset in the
// original Go string. The result is clamped to the input length.
func UTF16IndexToByteOffset(s string, index int) int {
	if index <= 0 {
		return 0
	}
	if index >= UTF16Length(s) {
		return len(s)
	}
	units := 0
	for offset := 0; offset < len(s); {
		r, size := utf8.DecodeRuneInString(s[offset:])
		nextUnits := units + utf16UnitsForRune(r)
		if nextUnits > index {
			return offset
		}
		units = nextUnits
		offset += size
	}
	return len(s)
}

// ByteOffsetToUTF16Index maps a byte offset in the Go string to the
// corresponding UTF-16 code-unit index.
func ByteOffsetToUTF16Index(s string, offset int) int {
	if offset <= 0 {
		return 0
	}
	if offset >= len(s) {
		return UTF16Length(s)
	}
	units := 0
	for i := 0; i < offset; {
		r, size := utf8.DecodeRuneInString(s[i:])
		units += utf16UnitsForRune(r)
		i += size
	}
	return units
}

func utf16UnitsForRune(r rune) int {
	if r >= 0x10000 {
		return 2
	}
	return 1
}
