package preflight

// Finding classifies repository-local state that deserves attention.
type Finding string

const (
	FindingDirty     Finding = "dirty"
	FindingError     Finding = "error"
	FindingStaged    Finding = "staged"
	FindingDetached  Finding = "detached"
	FindingNoRemote  Finding = "noremote"
	FindingUnpushed  Finding = "unpushed"
	FindingUnstaged  Finding = "unstaged"
	FindingStash     Finding = "stash"
	FindingOperation Finding = "operation"
)

var findingOrder = []Finding{
	FindingDirty,
	FindingError,
	FindingStaged,
	FindingDetached,
	FindingNoRemote,
	FindingUnpushed,
	FindingUnstaged,
	FindingStash,
	FindingOperation,
}

func orderedFindings(seen map[Finding]bool) []Finding {
	out := make([]Finding, 0, len(seen))
	for _, finding := range findingOrder {
		if seen[finding] {
			out = append(out, finding)
		}
	}
	return out
}

func hasFinding(findings []Finding, target Finding) bool {
	for _, finding := range findings {
		if finding == target {
			return true
		}
	}
	return false
}
