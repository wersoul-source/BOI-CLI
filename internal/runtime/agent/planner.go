package agent

import (
	"fmt"
	"strings"
)

type Planner struct{}

func NewPlanner() *Planner {
	return &Planner{}
}

func (p *Planner) Plan(query string) *TaskPlan {
	goal := strings.TrimSpace(query)
	return &TaskPlan{SchemaVersion: 1, ID: fmt.Sprintf("plan_%x", stableTaskID(goal)), Revision: 1, Goal: goal, Status: PlanPending, Steps: []PlannedStep{
		{Number: 1, Description: "Observe task and trusted runtime context", Phase: PhaseObserve, Status: PlanPending},
		{Number: 2, Description: "Decide the next bounded action", Phase: PhaseDecide, DependsOn: []int{1}, Status: PlanPending},
		{Number: 3, Description: "Authorize and execute any required host action", Phase: PhaseAct, DependsOn: []int{2}, Status: PlanPending},
		{Number: 4, Description: "Verify observable results", Phase: PhaseVerify, DependsOn: []int{2}, Status: PlanPending},
		{Number: 5, Description: "Return the verified outcome", Phase: PhaseStopped, DependsOn: []int{4}, Status: PlanPending},
	}}
}

func stableTaskID(value string) uint64 {
	var hash uint64 = 1469598103934665603
	for _, b := range []byte(value) {
		hash ^= uint64(b)
		hash *= 1099511628211
	}
	return hash
}

func (p *TaskPlan) Validate() error {
	if p == nil || p.SchemaVersion != 1 || p.Revision <= 0 || strings.TrimSpace(p.ID) == "" || strings.TrimSpace(p.Goal) == "" || len(p.Steps) == 0 {
		return fmt.Errorf("invalid Task Plan identity or steps")
	}
	seen := map[int]bool{}
	for _, step := range p.Steps {
		if step.Number <= 0 || seen[step.Number] || strings.TrimSpace(step.Description) == "" {
			return fmt.Errorf("invalid or duplicate Plan step %d", step.Number)
		}
		for _, dependency := range step.DependsOn {
			if !seen[dependency] {
				return fmt.Errorf("Plan step %d has unresolved dependency %d", step.Number, dependency)
			}
		}
		seen[step.Number] = true
	}
	return nil
}

func (p *TaskPlan) Revise(phase AgentPhase) {
	if p == nil {
		return
	}
	p.Revision++
	p.Revisions = append(p.Revisions, PlanRevision{Number: p.Revision, Phase: phase, Reason: "bounded recovery requested a new decision"})
	p.Status = PlanRunning
	setPlanPhase(p, PhaseDecide, PlanPending)
}
