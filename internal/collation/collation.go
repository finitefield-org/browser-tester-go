package collation

import (
	"strings"
	"unicode"
)

func Compare(left, right, locale string, numeric bool) int {
	leftRunes := []rune(left)
	rightRunes := []rune(right)
	leftIndex := 0
	rightIndex := 0

	for leftIndex < len(leftRunes) && rightIndex < len(rightRunes) {
		leftRune := leftRunes[leftIndex]
		rightRune := rightRunes[rightIndex]

		if numeric && isASCIIDigit(leftRune) && isASCIIDigit(rightRune) {
			leftDigits, nextLeft := digitRun(leftRunes, leftIndex)
			rightDigits, nextRight := digitRun(rightRunes, rightIndex)
			if cmp := compareDigits(leftDigits, rightDigits); cmp != 0 {
				return cmp
			}
			leftIndex = nextLeft
			rightIndex = nextRight
			continue
		}

		if cmp := compareRune(leftRune, rightRune, locale); cmp != 0 {
			return cmp
		}
		leftIndex++
		rightIndex++
	}

	switch {
	case leftIndex < len(leftRunes):
		return 1
	case rightIndex < len(rightRunes):
		return -1
	default:
		return 0
	}
}

func digitRun(runes []rune, start int) (string, int) {
	end := start
	for end < len(runes) && isASCIIDigit(runes[end]) {
		end++
	}
	return string(runes[start:end]), end
}

func compareDigits(left, right string) int {
	left = strings.TrimLeft(left, "0")
	right = strings.TrimLeft(right, "0")
	if left == "" {
		left = "0"
	}
	if right == "" {
		right = "0"
	}
	if len(left) < len(right) {
		return -1
	}
	if len(left) > len(right) {
		return 1
	}
	if left < right {
		return -1
	}
	if left > right {
		return 1
	}
	return 0
}

func compareRune(left, right rune, locale string) int {
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(locale)), "sv") {
		leftWeight := swedishWeight(left)
		rightWeight := swedishWeight(right)
		if leftWeight < rightWeight {
			return -1
		}
		if leftWeight > rightWeight {
			return 1
		}
		return 0
	}

	leftLower := unicode.ToLower(left)
	rightLower := unicode.ToLower(right)
	if leftLower < rightLower {
		return -1
	}
	if leftLower > rightLower {
		return 1
	}
	return 0
}

func swedishWeight(r rune) int {
	lower := unicode.ToLower(r)
	switch lower {
	case 'å':
		return 27
	case 'ä':
		return 28
	case 'ö':
		return 29
	}
	if lower >= 'a' && lower <= 'z' {
		return int(lower-'a') + 1
	}
	return 1000 + int(lower)
}

func isASCIIDigit(r rune) bool {
	return r >= '0' && r <= '9'
}
