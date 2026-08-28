package handler

import "testing"

func TestIsPlausibleUserAgent(t *testing.T) {
	valid := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) atomic/2.104.6 Chrome/142.0.7444.235 Safari/537.36"
	if !isPlausibleUserAgent(valid) {
		t.Fatalf("expected valid browser user agent")
	}

	rejects := []string{
		"",
		"curl/8.20.0",
		"python-requests/2.31.0",
		"Go-http-client/1.1",
		"Mozilla/5.0",
		"Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)",
	}
	for _, ua := range rejects {
		if isPlausibleUserAgent(ua) {
			t.Fatalf("expected reject for %q", ua)
		}
	}
}
