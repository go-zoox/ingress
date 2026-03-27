package service

import "testing"

func TestTarget(t *testing.T) {
	s := &Service{
		Name: "portainer",
		Port: 8080,
	}

	if s.Target() != "http://portainer:8080" {
		t.Fatalf("expected http://portainer:8080, got %s", s.Target())
	}

	s.Protocol = "https"
	if s.Target() != "https://portainer:8080" {
		t.Fatalf("expected https://portainer:8080, got %s", s.Target())
	}

	s.Port = 80
	if s.Target() != "https://portainer:80" {
		t.Fatalf("expected https://portainer:80, got %s", s.Target())
	}

	s = &Service{Name: "api", Port: 80, Protocol: "http"}
	if s.Target() != "http://api" {
		t.Fatalf("expected http://api (default port omitted), got %s", s.Target())
	}

	s = &Service{Name: "api", Port: 443, Protocol: "https"}
	if s.Target() != "https://api" {
		t.Fatalf("expected https://api (default port omitted), got %s", s.Target())
	}
}
