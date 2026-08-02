// Package documents maintains the language server's in-memory view of editor
// buffers: their versions, contents, and the overlay it derives for on-disk
// analysis. It owns the one canonical mapping between LSP UTF-16 positions and
// byte offsets so every feature agrees on coordinates.
package documents

import (
	"fmt"
	"net/url"
	"path/filepath"
	"runtime"
	"strings"
)

// URIToPath converts a file:// URI to an absolute filesystem path, handling
// percent-encoding, Windows drive letters, and UNC paths. Non-file URIs are
// rejected so virtual documents never masquerade as files.
func URIToPath(uri string) (string, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return "", fmt.Errorf("documents: invalid URI %q: %w", uri, err)
	}
	if u.Scheme != "file" {
		return "", fmt.Errorf("documents: not a file URI: %q", uri)
	}
	p, err := url.PathUnescape(u.Path)
	if err != nil {
		return "", fmt.Errorf("documents: undecodable URI path %q: %w", uri, err)
	}
	if runtime.GOOS == "windows" {
		// file:///C:/x -> C:/x ; file://server/share -> \\server\share
		if u.Host != "" {
			return filepath.FromSlash(`\\` + u.Host + p), nil
		}
		p = strings.TrimPrefix(p, "/")
		return filepath.FromSlash(p), nil
	}
	if u.Host != "" {
		return "", fmt.Errorf("documents: unexpected host in file URI %q", uri)
	}
	return filepath.Clean(p), nil
}

// PathToURI converts an absolute filesystem path to a file:// URI.
func PathToURI(path string) string {
	p := filepath.ToSlash(path)
	if runtime.GOOS == "windows" {
		if strings.HasPrefix(p, "//") {
			// UNC path \\server\share -> file://server/share
			return "file:" + (&url.URL{Path: p}).String()
		}
		p = "/" + p
	}
	u := url.URL{Scheme: "file", Path: p}
	return u.String()
}
