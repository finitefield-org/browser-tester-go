package runtime

import (
	"fmt"
	neturl "net/url"
	"strings"

	"browsertester/internal/script"
)

func browserURLInstanceReferenceValue(id string) script.Value {
	if strings.TrimSpace(id) == "" {
		return script.NullValue()
	}
	return script.HostObjectReference("url:" + strings.TrimSpace(id))
}

func browserURLConstructor(session *Session, args []script.Value) (script.Value, error) {
	if session == nil {
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "URL is unavailable in this bounded classic-JS slice")
	}
	if len(args) == 0 || len(args) > 2 {
		return script.UndefinedValue(), fmt.Errorf("URL expects 1 or 2 arguments")
	}

	href, err := scriptStringArg("URL", args, 0)
	if err != nil {
		return script.UndefinedValue(), err
	}
	resolved := strings.TrimSpace(href)
	if len(args) == 2 {
		base, err := scriptStringArg("URL", args, 1)
		if err != nil {
			return script.UndefinedValue(), err
		}
		resolved = resolveHyperlinkURL(base, href)
	}
	parsed, err := neturl.Parse(resolved)
	if err != nil {
		return script.UndefinedValue(), err
	}
	if len(args) == 1 && !parsed.IsAbs() {
		return script.UndefinedValue(), fmt.Errorf("URL constructor requires an absolute URL or a base URL")
	}
	id := session.allocateBrowserURLState(parsed, resolved)
	if strings.TrimSpace(id) == "" {
		return script.UndefinedValue(), fmt.Errorf("URL constructor could not allocate state")
	}
	return browserURLInstanceReferenceValue(id), nil
}

func resolveURLInstanceReference(session *Session, path string) (script.Value, error) {
	id, suffix, ok := parseBrowserURLInstancePath(path)
	if !ok || strings.TrimSpace(id) == "" {
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("unsupported browser surface %q in this bounded classic-JS slice", path))
	}
	state, ok := session.browserURLStateByID(id)
	if !ok || state == nil {
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("invalid URL reference %q in this bounded classic-JS slice", path))
	}

	if suffix == "" {
		return browserURLInstanceReferenceValue(id), nil
	}

	switch suffix {
	case "href":
		return script.StringValue(state.hrefString()), nil
	case "origin":
		return script.StringValue(browserURLOriginString(state.parsed)), nil
	case "protocol":
		return script.StringValue(protocolFromURL(state.parsed)), nil
	case "host":
		if state.parsed == nil {
			return script.StringValue(""), nil
		}
		return script.StringValue(state.parsed.Host), nil
	case "hostname":
		if state.parsed == nil {
			return script.StringValue(""), nil
		}
		return script.StringValue(state.parsed.Hostname()), nil
	case "port":
		if state.parsed == nil {
			return script.StringValue(""), nil
		}
		return script.StringValue(state.parsed.Port()), nil
	case "pathname":
		return script.StringValue(pathnameFromURL(state.parsed)), nil
	case "search":
		return script.StringValue(state.searchString()), nil
	case "hash":
		return script.StringValue(hashFromURL(state.parsed)), nil
	case "searchParams":
		return browserURLSearchParamsValueFromState(state.ensureSearchParams()), nil
	case "toString", "valueOf", "toJSON":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			if len(args) > 0 {
				return script.UndefinedValue(), fmt.Errorf("URL.%s accepts no arguments", suffix)
			}
			return script.StringValue(state.hrefString()), nil
		}), nil
	default:
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("unsupported browser surface %q in this bounded classic-JS slice", "url:"+id+"."+suffix))
	}
}

func setURLInstanceReferenceValue(session *Session, path string, value script.Value) error {
	id, suffix, ok := parseBrowserURLInstancePath(path)
	if !ok || strings.TrimSpace(id) == "" || suffix == "" {
		return script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("assignment to %q is unsupported in this bounded classic-JS slice", path))
	}
	state, ok := session.browserURLStateByID(id)
	if !ok || state == nil {
		return script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("invalid URL reference %q in this bounded classic-JS slice", path))
	}

	switch suffix {
	case "href":
		return state.setHref(script.ToJSString(value))
	case "search":
		state.setRawQuery(script.ToJSString(value))
		return nil
	default:
		return script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("assignment to %q is unsupported in this bounded classic-JS slice", path))
	}
}

func browserURLOriginString(parsed *neturl.URL) string {
	if parsed == nil || parsed.Scheme == "" || parsed.Host == "" {
		return ""
	}
	return parsed.Scheme + "://" + parsed.Host
}
