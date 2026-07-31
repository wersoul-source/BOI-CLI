package agent

import "strings"

type Planner struct{}

func NewPlanner() *Planner {
	return &Planner{}
}

func (p *Planner) Plan(query string) *TaskPlan {
	plan := &TaskPlan{Goal: query}

	lower := strings.ToLower(query)

	if strings.Contains(lower, "read") || strings.Contains(lower, "show") || strings.Contains(lower, "list") {
		plan.Steps = append(plan.Steps, PlannedStep{
			Number: 1, Description: "Scan for relevant files", Tool: "glob",
		})
		plan.Steps = append(plan.Steps, PlannedStep{
			Number: 2, Description: "Read and analyze files", Tool: "read",
			DependsOn: 1,
		})
	}

	if strings.Contains(lower, "fix") || strings.Contains(lower, "edit") || strings.Contains(lower, "change") {
		plan.Steps = append(plan.Steps, PlannedStep{
			Number: 3, Description: "Identify the source of the issue", Tool: "search",
		})
		plan.Steps = append(plan.Steps, PlannedStep{
			Number: 4, Description: "Apply the fix", Tool: "edit",
			DependsOn: 3,
		})
	}

	if len(plan.Steps) == 0 {
		plan.Steps = append(plan.Steps, PlannedStep{
			Number: 1, Description: "Analyze and respond", Tool: "think",
		})
	}

	return plan
}
