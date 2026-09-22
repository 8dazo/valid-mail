package main

import "regexp"

func init() {
	linkedInProfileURLRE = regexp.MustCompile(`(?i)(?:https?://)?(?:www\.)?linkedin\.com/in/[a-z0-9%._~-]+/?`)
}
