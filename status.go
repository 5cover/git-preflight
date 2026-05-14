package preflight

import (
	"strconv"
	"strings"
)

type statusInfo struct {
	Head        string
	Upstream    string
	Ahead       int
	HasBranchAB bool
	Staged      bool
	Unstaged    bool
	Dirty       bool
}

func parseStatusPorcelainV2(out string) statusInfo {
	var info statusInfo
	for _, line := range strings.Split(out, "\n") {
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "# ") {
			parseStatusHeader(&info, strings.TrimPrefix(line, "# "))
			continue
		}
		parseStatusEntry(&info, line)
	}
	return info
}

func parseStatusHeader(info *statusInfo, header string) {
	switch {
	case strings.HasPrefix(header, "branch.head "):
		info.Head = strings.TrimPrefix(header, "branch.head ")
	case strings.HasPrefix(header, "branch.upstream "):
		info.Upstream = strings.TrimPrefix(header, "branch.upstream ")
	case strings.HasPrefix(header, "branch.ab "):
		info.HasBranchAB = true
		fields := strings.Fields(strings.TrimPrefix(header, "branch.ab "))
		for _, field := range fields {
			if strings.HasPrefix(field, "+") {
				if n, err := strconv.Atoi(strings.TrimPrefix(field, "+")); err == nil {
					info.Ahead = n
				}
			}
		}
	}
}

func parseStatusEntry(info *statusInfo, line string) {
	if strings.HasPrefix(line, "? ") {
		info.Unstaged = true
		return
	}
	if len(line) < 4 {
		info.Dirty = true
		return
	}
	switch line[0] {
	case '1', '2':
		x, y := line[2], line[3]
		if x != '.' {
			info.Staged = true
		}
		if y != '.' {
			info.Unstaged = true
		}
	case 'u':
		info.Dirty = true
	default:
		info.Dirty = true
	}
}
