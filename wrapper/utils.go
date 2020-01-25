package main

import "regexp"

func RemoveExtension(from string) string {
	matcher := regexp.MustCompile("^(.*)\\.([^.]*)$")

	result := matcher.FindAllStringSubmatch(from, -1)
	if result == nil {
		return from
	} else {
		return result[0][1]
	}
}
