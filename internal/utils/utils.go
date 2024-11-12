package utils

import "regexp"

func SplitString(s string, sep string) []string {
	var result []string
	start := 0
	for i := 0; i < len(s); i++ {
		if i == len(s)-1 || s[i:i+len(sep)] == sep {
			if i == len(s)-1 {
				i++
			}
			result = append(result, s[start:i])
			start = i + len(sep)
			i += len(sep) - 1
		}
	}
	return result
}

func ValidateLuhn(number string) bool {
	// Проверяем, что строка содержит только цифры
	if !regexp.MustCompile(`^\d+$`).MatchString(number) {
		return false
	}

	sum := 0
	nDigits := len(number)
	parity := nDigits % 2

	for i := 0; i < nDigits; i++ {
		digit := int(number[i] - '0')

		if i%2 == parity {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}
		sum += digit
	}

	return sum%10 == 0
}
