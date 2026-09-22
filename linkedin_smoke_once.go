package main

import (
	"context"
	"log"
	"time"
)

func init() {
	go func() {
		time.Sleep(2 * time.Second)
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		resolution, err := discoverLinkedInProfileByName(ctx, "Patrick Collison")
		if err != nil {
			log.Printf("LINKEDIN_SMOKETEST failed: err=%v alternatives=%d confidence=%d ambiguous=%t", err, len(resolution.Alternatives), resolution.Confidence, resolution.Ambiguous)
			return
		}
		if resolution.Selected == nil {
			log.Printf("LINKEDIN_SMOKETEST failed: no selected profile alternatives=%d", len(resolution.Alternatives))
			return
		}
		log.Printf("LINKEDIN_SMOKETEST ok: url=%s name=%q company=%q status=%s score=%d alternatives=%d ambiguous=%t", resolution.Selected.URL, resolution.Selected.Name, resolution.Selected.Company, resolution.Selected.Status, resolution.Selected.Score, len(resolution.Alternatives), resolution.Ambiguous)
	}()
}
