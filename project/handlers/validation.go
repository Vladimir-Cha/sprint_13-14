package handlers

import (
	"strconv"
	"strings"
)

// isValidRepeatRule проверяет, поддерживается ли правило повторения
func isValidRepeatRule(repeat string) bool {
	if repeat == "" {
		return true
	}
	if repeat == "y" {
		return true
	}
	if strings.HasPrefix(repeat, "d ") {
		days := repeat[2:]
		_, err := strconv.Atoi(days)
		return err == nil && len(days) > 0
	}
	if strings.HasPrefix(repeat, "w ") {
		days := strings.Split(repeat[2:], ",")
		for _, day := range days {
			_, err := strconv.Atoi(day)
			if err != nil || len(day) != 1 || day < "1" || day > "7" {
				return false
			}
		}
		return true
	}
	if strings.HasPrefix(repeat, "m ") {
		parts := strings.Split(repeat[2:], " ")
		if len(parts) > 2 {
			return false
		}
		days := strings.Split(parts[0], ",")
		for _, day := range days {
			if day == "-1" || day == "-2" {
				continue
			}
			_, err := strconv.Atoi(day)
			if err != nil || len(day) > 2 || day < "1" || day > "31" {
				return false
			}
		}
		if len(parts) == 2 {
			months := strings.Split(parts[1], ",")
			for _, month := range months {
				_, err := strconv.Atoi(month)
				if err != nil || len(month) > 2 || month < "1" || month > "12" {
					return false
				}
			}
		}
		return true
	}
	return false
}
