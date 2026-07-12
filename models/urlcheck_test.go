package models

import "testing"

func TestValidateURLCheckMaxSecondary(t *testing.T) {
	cfg := URLCheck{
		Primary: URLTarget{URL: "https://example.com"},
		Secondary: []URLTarget{
			{URL: "https://a.example.com"},
			{URL: "https://b.example.com"},
			{URL: "https://c.example.com"},
			{URL: "https://d.example.com"},
			{URL: "https://e.example.com"},
			{URL: "https://f.example.com"},
		},
	}
	if err := ValidateURLCheck(cfg); err == nil {
		t.Fatal("expected max secondary error")
	}
}

func TestValidateURLCheckInvalidScheme(t *testing.T) {
	cfg := URLCheck{Primary: URLTarget{URL: "ftp://example.com"}}
	if err := ValidateURLCheck(cfg); err == nil {
		t.Fatal("expected invalid scheme error")
	}
}
