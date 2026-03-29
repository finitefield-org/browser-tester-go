package mocks

import (
	"fmt"
	"strings"
)

type ExternalJSCall struct {
	URL string
}

type ExternalJSSourceRule struct {
	URL    string
	Source string
}

type ExternalJSErrorRule struct {
	URL     string
	Message string
}

type ExternalJSFamily struct {
	sourceRules []ExternalJSSourceRule
	errorRules  []ExternalJSErrorRule
	calls       []ExternalJSCall
}

func (f *ExternalJSFamily) RespondSource(url string, source string) {
	if f == nil {
		return
	}
	url = strings.TrimSpace(url)
	f.sourceRules = append(f.sourceRules, ExternalJSSourceRule{
		URL:    url,
		Source: source,
	})
}

func (f *ExternalJSFamily) Fail(url string, message string) {
	if f == nil {
		return
	}
	url = strings.TrimSpace(url)
	f.errorRules = append(f.errorRules, ExternalJSErrorRule{
		URL:     url,
		Message: message,
	})
}

func (f *ExternalJSFamily) Resolve(url string) (string, error) {
	if f == nil {
		return "", fmt.Errorf("external JS mock registry is unavailable")
	}

	url = strings.TrimSpace(url)
	if url == "" {
		return "", fmt.Errorf("external JS load requires a non-empty URL")
	}

	f.calls = append(f.calls, ExternalJSCall{URL: url})

	for i := len(f.errorRules) - 1; i >= 0; i-- {
		rule := f.errorRules[i]
		if rule.URL == url {
			return "", fmt.Errorf("%s", rule.Message)
		}
	}

	for i := len(f.sourceRules) - 1; i >= 0; i-- {
		rule := f.sourceRules[i]
		if rule.URL == url {
			return rule.Source, nil
		}
	}

	return "", fmt.Errorf("no external JS mock configured for `%s`", url)
}

func (f *ExternalJSFamily) Calls() []ExternalJSCall {
	if f == nil {
		return nil
	}
	out := make([]ExternalJSCall, len(f.calls))
	copy(out, f.calls)
	return out
}

func (f *ExternalJSFamily) Sources() []ExternalJSSourceRule {
	if f == nil {
		return nil
	}
	out := make([]ExternalJSSourceRule, len(f.sourceRules))
	copy(out, f.sourceRules)
	return out
}

func (f *ExternalJSFamily) Errors() []ExternalJSErrorRule {
	if f == nil {
		return nil
	}
	out := make([]ExternalJSErrorRule, len(f.errorRules))
	copy(out, f.errorRules)
	return out
}

func (f *ExternalJSFamily) Reset() {
	if f == nil {
		return
	}
	f.sourceRules = nil
	f.errorRules = nil
	f.calls = nil
}
