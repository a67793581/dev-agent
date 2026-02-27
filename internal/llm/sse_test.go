package llm

import (
	"strings"
	"testing"
)

func TestNewSSEScanner(t *testing.T) {
	r := strings.NewReader("data: hello\n")
	s := NewSSEScanner(r)
	if s == nil || s.scanner == nil {
		t.Fatal("NewSSEScanner should return non-nil scanner")
	}
}

func TestSSEScanner_ScanAndData(t *testing.T) {
	input := "data: line1\n\ndata: line2\n"
	r := strings.NewReader(input)
	s := NewSSEScanner(r)

	if !s.Scan() {
		t.Fatal("first Scan should return true")
	}
	if s.Data() != "line1" {
		t.Errorf("Data() = %q, want line1", s.Data())
	}
	if !s.Scan() {
		t.Fatal("second Scan should return true")
	}
	if s.Data() != "line2" {
		t.Errorf("Data() = %q, want line2", s.Data())
	}
	if s.Scan() {
		t.Error("third Scan should return false")
	}
}

func TestSSEScanner_IgnoresNonDataLines(t *testing.T) {
	input := "event: message\nid: 1\ndata: payload\n\n"
	r := strings.NewReader(input)
	s := NewSSEScanner(r)
	if !s.Scan() {
		t.Fatal("Scan should find data line")
	}
	if s.Data() != "payload" {
		t.Errorf("Data() = %q, want payload", s.Data())
	}
}

func TestSSEScanner_Err(t *testing.T) {
	r := strings.NewReader("data: ok\n")
	s := NewSSEScanner(r)
	for s.Scan() {
	}
	if s.Err() != nil {
		t.Errorf("Err() = %v, want nil", s.Err())
	}
}
