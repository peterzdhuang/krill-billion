package supabank

import (
	"errors"
	"regexp"
	"strings"
)

var passwordRE = regexp.MustCompile(`(?i)(password=)[^\s]+`)

func redact(err error, secrets ...string) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	for _, s := range secrets {
		if len(s) < 4 {
			continue
		}
		msg = strings.ReplaceAll(msg, s, "[redacted]")
	}
	msg = passwordRE.ReplaceAllString(msg, "${1}[redacted]")
	if msg == err.Error() {
		return err
	}
	return errors.New(msg)
}
