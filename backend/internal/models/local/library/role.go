package library

import (
	"errors"
	"regexp"
)

var helperFile = regexp.MustCompile(`(?i)(^|[/\\])(mtp|draft|dspark)([-_. /\\]|$)`)
var projectorFile = regexp.MustCompile(`(?i)(^|[/\\])mmproj([-_. /\\]|$)`)

var encoderFile = regexp.MustCompile(`(?i)(^|[-_./\\])(vision|text|audio)[-_]encoder([-_. /\\]|$)`)
var calibrationFile = regexp.MustCompile(`(?i)(^|[/\\])imatrix[-_].*\.gguf$`)
var quantizedFile = regexp.MustCompile(`(?i)(IQ[1-4]_|Q[2-8]_|BF16|F16|F32)`)

func ValidateChatFile(name string) error {
	if encoderFile.MatchString(name) || (calibrationFile.MatchString(name) && !quantizedFile.MatchString(name)) {
		return errors.New("This file is a model component. Choose a main model file")
	}
	if helperFile.MatchString(name) {
		return errors.New("This is a prediction helper, not a standalone chat model. Choose a main model file in Browse models")
	}
	if projectorFile.MatchString(name) {
		return errors.New("This is a vision projector, not a standalone chat model. Choose a main model file in Browse models")
	}
	return nil
}
