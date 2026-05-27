package envsetup

import "strings"

func hasStructuredLog(logs []LogMessage, component, stage, action string) bool {
	for _, entry := range logs {
		if strings.EqualFold(strings.TrimSpace(entry.Component), strings.TrimSpace(component)) &&
			strings.EqualFold(strings.TrimSpace(entry.Stage), strings.TrimSpace(stage)) &&
			strings.EqualFold(strings.TrimSpace(entry.Action), strings.TrimSpace(action)) {
			return true
		}
	}
	return false
}
