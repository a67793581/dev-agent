package tools

import (
	"devagent/internal/skill"
	"fmt"
)

// ReadSkillTool loads the full body of a skill by name from the discovered skills list.
type ReadSkillTool struct {
	skills []skill.Skill
}

// NewReadSkillTool creates a ReadSkillTool that uses the given skills slice.
func NewReadSkillTool(skills []skill.Skill) *ReadSkillTool {
	return &ReadSkillTool{skills: skills}
}

func (t *ReadSkillTool) Name() string { return "read_skill" }

func (t *ReadSkillTool) Execute(args map[string]string) Result {
	name := args["name"]
	if name == "" {
		return Result{Success: false, Output: "read_skill requires \"name\" argument"}
	}

	var found *skill.Skill
	for i := range t.skills {
		if t.skills[i].Name == name {
			found = &t.skills[i]
			break
		}
	}
	if found == nil {
		return Result{
			Success: false,
			Output:  fmt.Sprintf("skill not found: %q. Use the available skills list from the context.", name),
		}
	}

	if err := skill.LoadBody(found); err != nil {
		return Result{Success: false, Output: fmt.Sprintf("load skill: %v", err)}
	}

	return Result{Success: true, Output: found.Body}
}
