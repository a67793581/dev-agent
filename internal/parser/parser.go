package parser

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

type Command struct {
	Name   string            `json:"command"`
	Args   map[string]string `json:"args"`
	Reason string            `json:"reason"`
}

var (
	commandBlockRe = regexp.MustCompile("(?s)```json\\s*\\n(.*?)\\n```")
	thinkBlockRe   = regexp.MustCompile("(?s)<think>(.*?)</think>")
	codeBlockRe    = regexp.MustCompile("(?s)```(\\w*)\\s*\\n(.*?)\\n```")
)

func ParseCommands(text string) ([]Command, string, error) {
	thinking := ""
	if matches := thinkBlockRe.FindStringSubmatch(text); len(matches) > 1 {
		thinking = strings.TrimSpace(matches[1])
	}

	jsonMatches := commandBlockRe.FindAllStringSubmatch(text, -1)
	if len(jsonMatches) == 0 {
		return nil, thinking, nil
	}

	var allCommands []Command
	for _, match := range jsonMatches {
		raw := match[1]
		raw = repairJSON(raw)

		if strings.HasPrefix(strings.TrimSpace(raw), "[") {
			var cmds []Command
			if err := json.Unmarshal([]byte(raw), &cmds); err != nil {
				single, singleErr := tryParseSingle(raw)
				if singleErr != nil {
					return nil, thinking, fmt.Errorf("parse command array: %w (original: %v)", singleErr, err)
				}
				allCommands = append(allCommands, single)
				continue
			}
			allCommands = append(allCommands, cmds...)
		} else {
			cmd, err := tryParseSingle(raw)
			if err != nil {
				return nil, thinking, fmt.Errorf("parse single command: %w", err)
			}
			allCommands = append(allCommands, cmd)
		}
	}

	return allCommands, thinking, nil
}

func tryParseSingle(raw string) (Command, error) {
	var cmd Command
	if err := json.Unmarshal([]byte(raw), &cmd); err != nil {
		return Command{}, err
	}
	return cmd, nil
}

func ParseCodeBlock(text, lang string) string {
	pattern := fmt.Sprintf("(?s)```%s\\s*\\n(.*?)\\n```", regexp.QuoteMeta(lang))
	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(text)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}

	allMatches := codeBlockRe.FindAllStringSubmatch(text, -1)
	for _, m := range allMatches {
		if lang == "" || m[1] == lang || m[1] == "" {
			return strings.TrimSpace(m[2])
		}
	}
	return ""
}

func ParseAllCodeBlocks(text string) map[string]string {
	result := make(map[string]string)
	matches := codeBlockRe.FindAllStringSubmatch(text, -1)
	for i, m := range matches {
		lang := m[1]
		if lang == "" {
			lang = fmt.Sprintf("block_%d", i)
		}
		result[lang] = strings.TrimSpace(m[2])
	}
	return result
}

func ParseSections(text string) map[string]string {
	sections := make(map[string]string)
	re := regexp.MustCompile(`(?m)^##\s+(.+)$`)
	matches := re.FindAllStringSubmatchIndex(text, -1)

	for i, match := range matches {
		title := strings.TrimSpace(text[match[2]:match[3]])
		start := match[1]
		end := len(text)
		if i+1 < len(matches) {
			end = matches[i+1][0]
		}
		content := strings.TrimSpace(text[start:end])
		sections[title] = content
	}
	return sections
}

func ExtractTextBetweenTags(text, openTag, closeTag string) string {
	start := strings.Index(text, openTag)
	if start == -1 {
		return ""
	}
	start += len(openTag)
	end := strings.Index(text[start:], closeTag)
	if end == -1 {
		return strings.TrimSpace(text[start:])
	}
	return strings.TrimSpace(text[start : start+end])
}

func repairJSON(raw string) string {
	raw = strings.TrimSpace(raw)

	if strings.HasSuffix(raw, ",") {
		raw = raw[:len(raw)-1]
	}

	trailingCommaRe := regexp.MustCompile(`,\s*([}\]])`)
	raw = trailingCommaRe.ReplaceAllString(raw, "$1")

	if strings.Count(raw, "{") > strings.Count(raw, "}") {
		diff := strings.Count(raw, "{") - strings.Count(raw, "}")
		for i := 0; i < diff; i++ {
			raw += "}"
		}
	}
	if strings.Count(raw, "[") > strings.Count(raw, "]") {
		diff := strings.Count(raw, "[") - strings.Count(raw, "]")
		for i := 0; i < diff; i++ {
			raw += "]"
		}
	}

	raw = strings.ReplaceAll(raw, "\t", "\\t")

	return raw
}
