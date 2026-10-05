package agent

import (
	session "arx/internal/sessions"
	"strings"
	"unicode/utf8"
)

func interruptedOutput(tools []session.ToolRecord) string {
	var message strings.Builder
	message.WriteString("[Previous response reached its output limit. Continue the user's task using the saved partial reply. Any unfinished tool calls below were not executed. Re-read existing files before retrying; split large writes into smaller operations.] ")
	remaining := 16000
	for _, tool := range tools {
		if tool.Status != "preparing" {
			continue
		}
		arguments := tool.Arguments
		if len(arguments) > remaining {
			end := remaining
			for end > 0 && !utf8.RuneStart(arguments[end]) {
				end--
			}
			arguments = arguments[:end] + " [remaining arguments omitted]"
		}
		message.WriteString("\nUnfinished " + tool.Name + " arguments:\n" + arguments)
		remaining -= len(arguments)
		if remaining <= 0 {
			break
		}
	}
	return message.String()
}
