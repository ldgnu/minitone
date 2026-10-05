package ui

import (
	"fmt"
	"strings"
	"time"
)

// noticeKind drives the colour and the urgency of the status line.
type noticeKind int

const (
	noticeInfo noticeKind = iota
	noticeWarn
	noticeError
)

// notice is the single status message shown under the results.
//
// Errors must always tell the user how to continue, so a notice carries an
// optional action hint such as "[r] retry".
type notice struct {
	text   string
	detail string
	action string
	kind   noticeKind
	at     time.Time
	ttl    time.Duration
	sticky bool
	// retryable marks a notice whose action key is "r".
	retryable bool
}

func (n notice) empty() bool { return n.text == "" }

// active reports whether the notice should currently be shown.
func (n notice) active() bool { return !n.empty() && !n.expired(time.Now()) }

// expired reports whether a temporary notice should be gone by now.
func (n notice) expired(now time.Time) bool {
	if n.empty() || n.sticky {
		return false
	}
	if n.ttl <= 0 {
		return false
	}
	return now.Sub(n.at) > n.ttl
}

// newNotice stamps the creation time: without it every notice would look
// expired the moment it is created.
func newNotice(text, detail, action string, kind noticeKind, ttl time.Duration) notice {
	return notice{
		text:   text,
		detail: detail,
		action: action,
		kind:   kind,
		at:     time.Now(),
		ttl:    ttl,
	}
}

func infoNotice(text, detail, action string) notice {
	return newNotice(text, detail, action, noticeInfo, 4*time.Second)
}

func okNotice(text, detail, action string) notice {
	return newNotice(text, detail, action, noticeInfo, 3*time.Second)
}

func warnNotice(text, detail, action string) notice {
	return newNotice(text, detail, action, noticeWarn, 8*time.Second)
}

func errorNotice(text, detail, action string) notice {
	n := newNotice(text, detail, action, noticeError, 0)
	n.sticky = true
	return n
}

// retryNotice is an error the user can try again right away.
func retryNotice(text string, err error) notice {
	n := errorNotice(text, errText(err), "[r] retry  [esc] dismiss")
	n.retryable = true
	return n
}

func errText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// hintForSource turns a source failure into a short, actionable message.
func hintForSource(source string, err error) string {
	msg := errText(err)
	switch {
	case strings.Contains(msg, "yt-dlp"):
		return "install yt-dlp or search another source"
	case strings.Contains(msg, "radio browser"), strings.Contains(msg, "Radio Browser"):
		return "check your network connection"
	case strings.Contains(msg, "context canceled"):
		return "cancelled"
	case strings.Contains(msg, "no such host"), strings.Contains(msg, "timeout"):
		return "check your network connection"
	}
	return "press r to retry, esc to dismiss"
}

// sourceFailureNotice summarises one or more failing sources.
func sourceFailureNotice(failures int, first string) notice {
	if failures == 1 {
		return retryNotice(first, nil)
	}
	return retryNotice(fmt.Sprintf("%d sources unavailable", failures), nil)
}
