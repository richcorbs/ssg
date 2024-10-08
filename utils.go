package main

import (
	"strings"
)

func sliceContains(str string, arr []string) bool {
	for _, v := range arr {
		if v == str {
			return true
		}
	}
	return false
}

func replaceAWithB(haystack string, A string, B string) string {
	return strings.Replace(haystack, A, B, -1)
}
