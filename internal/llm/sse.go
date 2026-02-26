package llm

import (
	"bufio"
	"io"
	"strings"
)

type SSEScanner struct {
	scanner *bufio.Scanner
	data    string
}

func NewSSEScanner(r io.Reader) *SSEScanner {
	return &SSEScanner{
		scanner: bufio.NewScanner(r),
	}
}

func (s *SSEScanner) Scan() bool {
	for s.scanner.Scan() {
		line := s.scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			s.data = strings.TrimPrefix(line, "data: ")
			return true
		}
	}
	return false
}

func (s *SSEScanner) Data() string {
	return s.data
}

func (s *SSEScanner) Err() error {
	return s.scanner.Err()
}
