package features

import "fmt"

// BranchAction defines what to do regarding git branches
type BranchAction int

const (
	BranchActionNone BranchAction = iota
	BranchActionStay
	BranchActionCreate
)

// DetermineBranchAction decides whether to create a branch, stay on current, or do nothing
func DetermineBranchAction(mode string, allowShared bool, same, isolate bool) (BranchAction, error) {
	if same && isolate {
		return BranchActionNone, fmt.Errorf("cannot use both --same and --isolate flags")
	}

	if mode == "trunk-based" {
		if same {
			return BranchActionNone, fmt.Errorf("--same flag only works in branch-per-feature mode")
		}
		if isolate {
			return BranchActionNone, fmt.Errorf("--isolate flag only works in branch-per-feature mode")
		}
		return BranchActionNone, nil
	}

	if mode == "branch-per-feature" {
		if same {
			if !allowShared {
				return BranchActionNone, fmt.Errorf("--same flag requires workflow.allow_shared_branches: true in config")
			}
			return BranchActionStay, nil
		}
		return BranchActionCreate, nil
	}

	return BranchActionNone, nil
}
