package agent

import (
	"path"
	"regexp"
	"strings"

	"arx/internal/tool"
)

func hardDeny(req Request) (bool, string) {
	if req.Tool != "bash" {
		return false, ""
	}
	command := tool.Command(req.Args)
	if strings.TrimSpace(command) == "" {
		return false, ""
	}
	cmd := normalizeCommand(command)

	if privEscRe.MatchString(cmd) {
		return true, "privilege escalation"
	}
	if pipeToInterp.MatchString(cmd) {
		return true, "pipes into an interpreter"
	}
	if netToInterp(cmd) {
		return true, "runs downloaded code"
	}
	if evalRe.MatchString(cmd) {
		return true, "evaluates a constructed command"
	}
	if recursiveRemoveOutside(cmd) {
		return true, "recursive delete outside the workspace"
	}
	if deviceWrite.MatchString(cmd) {
		return true, "writes a raw device"
	}
	if p, hit := touchesSensitive(cmd); hit {
		return true, "touches " + p
	}
	if destroyerRe.MatchString(cmd) || chmodRecursiveRoot(cmd) {
		return true, "destructive to disk or system"
	}
	return false, ""
}

func normalizeCommand(s string) string {
	s = strings.ToLower(s)
	s = braceVar.ReplaceAllString(s, "$$${1}")
	return spaceRun.ReplaceAllString(s, " ")
}

var (
	braceVar = regexp.MustCompile(`\$\{(\w+)\}`)
	spaceRun = regexp.MustCompile(`[ \t]+`)

	privEscRe = regexp.MustCompile("(^|[\\s;&|(`])\\\\?(sudo|doas|su)(\\s|$)")

	pipeToInterp = regexp.MustCompile(`\|&?\s*(sudo\s+)?(\S*/)?(ba|z|da|k|c|tc|a)?sh\b|\|&?\s*(sudo\s+)?(\S*/)?(python3?|perl|ruby|node|php)\b`)

	evalRe = regexp.MustCompile("(^|[\\s;&|(`])eval(\\s|$)")

	deviceWrite = regexp.MustCompile(`(of=|>\s*)['"]?/dev/(sd|nvme|vd|hd|mmcblk|disk|rdisk)`)

	destroyerRe = regexp.MustCompile(`\b(mkfs|mke2fs|wipefs|shred|fdisk|parted)\b|:\s*\(\s*\)\s*\{`)
)

func netToInterp(cmd string) bool {
	if !strings.Contains(cmd, "curl") && !strings.Contains(cmd, "wget") {
		return false
	}
	return strings.Contains(cmd, "<(") || strings.Contains(cmd, "<<<")
}

var sensitiveBasenames = map[string]bool{
	".bashrc": true, ".zshrc": true, ".zshenv": true, ".zprofile": true,
	".zlogin": true, ".bash_profile": true, ".profile": true,
	".netrc": true, ".npmrc": true, ".gitconfig": true, ".pgpass": true,
}

var sensitiveDirs = []string{
	"/.ssh/", "/.aws/", "/.kube/", "/.gnupg/", "/.config/gh/",
	"/library/launchagents/", "/.git/hooks/", "/.git/config",
}

func touchesSensitive(cmd string) (string, bool) {
	for _, tok := range tokenize(cmd) {
		t := stripQuotes(tok)
		if sensitiveBasenames[path.Base(t)] {
			return path.Base(t), true
		}
		for _, d := range sensitiveDirs {
			if strings.Contains(t, d) {
				return strings.Trim(d, "/"), true
			}
		}
	}
	return "", false
}

func chmodRecursiveRoot(cmd string) bool {
	for _, stmt := range splitStatements(cmd) {
		fields := strings.Fields(stmt)
		head, rest := resolveHead(fields)
		if head != "chmod" && head != "chown" {
			continue
		}
		recursive := false
		root := false
		for _, f := range rest {
			t := stripQuotes(f)
			if strings.HasPrefix(t, "-") && strings.Contains(t, "r") {
				recursive = true
			}
			if t == "/" || t == "~" || t == "$home" || strings.HasPrefix(t, "/*") {
				root = true
			}
		}
		if recursive && root {
			return true
		}
	}
	return false
}

func splitStatements(cmd string) []string {
	return strings.FieldsFunc(cmd, func(r rune) bool {
		switch r {
		case ';', '&', '|', '\n', '(', ')', '`':
			return true
		}
		return false
	})
}

var wrapperWords = map[string]bool{
	"command": true, "env": true, "time": true, "exec": true,
	"nice": true, "nohup": true, "setsid": true, "builtin": true,
	"sudo": true, "doas": true, "xargs": true, "then": true, "do": true,
}

func resolveHead(fields []string) (string, []string) {
	i := 0
	for i < len(fields) {
		f := fields[i]
		if isAssignment(f) {
			i++
			continue
		}
		name := headName(f)
		if wrapperWords[name] {
			i++
			continue
		}
		return name, fields[i+1:]
	}
	return "", nil
}

func headName(f string) string {
	return path.Base(strings.TrimPrefix(stripQuotes(f), `\`))
}

func isAssignment(f string) bool {
	eq := strings.IndexByte(f, '=')
	if eq <= 0 {
		return false
	}
	for _, r := range f[:eq] {
		if r != '_' && !(r >= 'a' && r <= 'z') && !(r >= '0' && r <= '9') {
			return false
		}
	}
	return true
}

func stripQuotes(s string) string {
	return strings.NewReplacer(`"`, "", `'`, "").Replace(s)
}

func tokenize(cmd string) []string {
	return strings.FieldsFunc(cmd, func(r rune) bool {
		switch r {
		case ' ', '\n', '>', '<', ';', '|', '&', '(', ')', '`':
			return true
		}
		return false
	})
}

func recursiveRemoveOutside(cmd string) bool {
	for _, stmt := range splitStatements(cmd) {
		fields := strings.Fields(stmt)
		head, rest := resolveHead(fields)
		if head != "rm" {
			continue
		}
		recursive := false
		var targets []string
		for _, f := range rest {
			t := stripQuotes(f)
			switch {
			case strings.HasPrefix(t, "--"):
				if t == "--recursive" {
					recursive = true
				}
			case strings.HasPrefix(t, "-"):
				if strings.Contains(t, "r") {
					recursive = true
				}
			default:
				targets = append(targets, t)
			}
		}
		if !recursive {
			continue
		}
		for _, tgt := range targets {
			if tgt == "~" || tgt == ".." || tgt == "$home" ||
				strings.HasPrefix(tgt, "/") || strings.HasPrefix(tgt, "~/") ||
				strings.HasPrefix(tgt, "../") || strings.HasPrefix(tgt, "$") {
				return true
			}
		}
	}
	return false
}
