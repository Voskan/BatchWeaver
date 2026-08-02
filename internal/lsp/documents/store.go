package documents

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"

	"github.com/Voskan/BatchWeaver/internal/lsp/protocol"
)

// Document is one open editor buffer. It is authoritative over the on-disk file
// while open.
type Document struct {
	URI        protocol.DocumentURI
	Path       string
	LanguageID string
	Version    int32
	Content    []byte
}

// Digest returns the SHA-256 hex digest of the current content, used in edit
// preconditions.
func (d *Document) Digest() string {
	sum := sha256.Sum256(d.Content)
	return hex.EncodeToString(sum[:])
}

// Mapper returns a position mapper for the current content.
func (d *Document) Mapper() *Mapper { return NewMapper(d.Content) }

// Store holds all open documents. It is safe for concurrent use. It rejects
// out-of-order versions so a late change never regresses newer content.
type Store struct {
	mu   sync.RWMutex
	docs map[protocol.DocumentURI]*Document
}

// NewStore returns an empty document store.
func NewStore() *Store {
	return &Store{docs: make(map[protocol.DocumentURI]*Document)}
}

// Open records a newly opened document.
func (s *Store) Open(item protocol.TextDocumentItem) (*Document, error) {
	path, err := URIToPath(item.URI)
	if err != nil {
		return nil, err
	}
	d := &Document{
		URI:        item.URI,
		Path:       path,
		LanguageID: item.LanguageID,
		Version:    item.Version,
		Content:    []byte(item.Text),
	}
	s.mu.Lock()
	s.docs[item.URI] = d
	s.mu.Unlock()
	return d, nil
}

// Change applies content changes, returning the updated document. Full and
// incremental changes are both supported. An out-of-order version is rejected.
func (s *Store) Change(id protocol.VersionedTextDocumentIdentifier, changes []protocol.TextDocumentContentChangeEvent) (*Document, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	d, ok := s.docs[id.URI]
	if !ok {
		return nil, fmt.Errorf("documents: change to unopened document %q", id.URI)
	}
	if id.Version < d.Version {
		return nil, fmt.Errorf("documents: out-of-order version %d < %d for %q", id.Version, d.Version, id.URI)
	}
	content := d.Content
	for _, ch := range changes {
		if ch.Range == nil {
			content = []byte(ch.Text) // full replacement
			continue
		}
		m := NewMapper(content)
		start, end := m.RangeToByteRange(*ch.Range)
		if start > end || start < 0 || end > len(content) {
			return nil, fmt.Errorf("documents: invalid change range for %q", id.URI)
		}
		next := make([]byte, 0, len(content)-(end-start)+len(ch.Text))
		next = append(next, content[:start]...)
		next = append(next, ch.Text...)
		next = append(next, content[end:]...)
		content = next
	}
	d.Content = content
	d.Version = id.Version
	return d, nil
}

// Close removes a document. It reports whether the document was open.
func (s *Store) Close(uri protocol.DocumentURI) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.docs[uri]
	delete(s.docs, uri)
	return ok
}

// Get returns a snapshot copy of a document, or (nil, false) if not open.
func (s *Store) Get(uri protocol.DocumentURI) (*Document, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	d, ok := s.docs[uri]
	if !ok {
		return nil, false
	}
	clone := *d
	clone.Content = append([]byte(nil), d.Content...)
	return &clone, true
}

// Overlay returns the go/packages overlay map for all open documents: absolute
// path to current in-memory bytes. This is what feeds unsaved buffers to
// analysis without writing to disk.
func (s *Store) Overlay() map[string][]byte {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string][]byte, len(s.docs))
	for _, d := range s.docs {
		out[d.Path] = append([]byte(nil), d.Content...)
	}
	return out
}

// OpenURIs returns the URIs of all open documents.
func (s *Store) OpenURIs() []protocol.DocumentURI {
	s.mu.RLock()
	defer s.mu.RUnlock()
	uris := make([]protocol.DocumentURI, 0, len(s.docs))
	for uri := range s.docs {
		uris = append(uris, uri)
	}
	return uris
}
