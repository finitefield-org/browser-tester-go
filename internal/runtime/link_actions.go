package runtime

import (
	"encoding/base64"
	"net/url"
	"path"
	"strings"

	"browsertester/internal/dom"
)

func (s *Session) applyHyperlinkDefaultAction(node *dom.Node) error {
	if s == nil || node == nil || node.Kind != dom.NodeKindElement {
		return nil
	}

	href, ok := attributeValue(node.Attrs, "href")
	if !ok {
		return nil
	}
	destination := resolveHyperlinkURL(s.URL(), href)
	if strings.TrimSpace(destination) == "" {
		return nil
	}

	if download, ok := attributeValue(node.Attrs, "download"); ok {
		fileName := hyperlinkDownloadFileName(destination, download)
		if bytes, ok := downloadBytesForDestination(s, destination); ok {
			s.Registry().Downloads().Capture(fileName, bytes)
		}
		return nil
	}

	target, _ := attributeValue(node.Attrs, "target")
	if strings.EqualFold(strings.TrimSpace(target), "_blank") {
		return s.Registry().Open().Invoke(destination)
	}

	return s.Navigate(destination)
}

func resolveHyperlinkURL(baseURL, href string) string {
	href = strings.TrimSpace(href)
	if href == "" {
		return ""
	}

	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		return href
	}

	base, err := url.Parse(baseURL)
	if err != nil {
		return href
	}
	ref, err := url.Parse(href)
	if err != nil {
		return href
	}
	return base.ResolveReference(ref).String()
}

func hyperlinkDownloadFileName(destination, suggested string) string {
	suggested = strings.TrimSpace(suggested)
	if suggested != "" {
		return suggested
	}

	parsed, err := url.Parse(destination)
	if err != nil {
		return "download"
	}
	name := path.Base(parsed.Path)
	if name == "." || name == "/" || name == "" {
		return "download"
	}
	return name
}

func downloadBytesForDestination(session *Session, destination string) ([]byte, bool) {
	trimmed := strings.TrimSpace(destination)
	if strings.HasPrefix(trimmed, "data:") {
		afterScheme := strings.TrimPrefix(trimmed, "data:")
		comma := strings.IndexByte(afterScheme, ',')
		if comma < 0 {
			return []byte(destination), true
		}

		meta := afterScheme[:comma]
		payload := afterScheme[comma+1:]
		if strings.Contains(strings.ToLower(meta), ";base64") {
			decoded, err := base64.StdEncoding.DecodeString(payload)
			if err != nil {
				return []byte(destination), true
			}
			return decoded, true
		}

		decoded, err := url.PathUnescape(payload)
		if err != nil {
			return []byte(destination), true
		}
		return []byte(decoded), true
	}
	if strings.HasPrefix(trimmed, "blob:") {
		if bytes, ok := downloadBytesForBlobDestination(session, trimmed); ok {
			return bytes, true
		}
		return nil, false
	}
	return []byte(destination), true
}
