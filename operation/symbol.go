package operation

import (
	"errors"
	"fmt"
	"strings"
)

// Symbol is a stable, source-independent reference to a Go function or method,
// used in configuration to name the scalar and batch implementations of an
// operation.
//
// Supported forms:
//
//	github.com/example/project/users.GetUser              // function
//	github.com/example/project/users.(*Repository).GetUser // pointer method
//	github.com/example/project/users.(Repository).GetUser  // value method
//
// Generic instantiation is not encoded in version-1 configuration. A Symbol is
// parsed and validated for shape only; it is not resolved against loaded Go
// packages in this release. The zero value is invalid.
type Symbol struct {
	importPath string
	receiver   string // receiver type name without '*'; empty for functions
	name       string
	pointer    bool
}

// ErrInvalidSymbol is returned when a Symbol fails to parse or validate.
var ErrInvalidSymbol = errors.New("invalid symbol")

// ParseSymbol parses a Go symbol reference. It preserves case because Go
// identifiers are case-sensitive, and it rejects empty input, relative import
// paths, filesystem paths, spaces, control characters, and malformed
// parentheses.
func ParseSymbol(value string) (Symbol, error) {
	if value == "" {
		return Symbol{}, fmt.Errorf("%w: symbol is empty", ErrInvalidSymbol)
	}
	if strings.ContainsAny(value, " \t\r\n") {
		return Symbol{}, fmt.Errorf("%w: symbol contains whitespace", ErrInvalidSymbol)
	}
	for _, r := range value {
		if r < 0x20 || r == 0x7f {
			return Symbol{}, fmt.Errorf("%w: symbol contains control characters", ErrInvalidSymbol)
		}
	}

	if strings.Contains(value, ".(") {
		return parseMethodSymbol(value)
	}
	return parseFunctionSymbol(value)
}

// parseFunctionSymbol parses "<importPath>.<Name>".
func parseFunctionSymbol(value string) (Symbol, error) {
	if strings.ContainsAny(value, "()") {
		return Symbol{}, fmt.Errorf("%w: malformed parentheses in %q", ErrInvalidSymbol, value)
	}
	dot := strings.LastIndex(value, ".")
	if dot <= 0 || dot == len(value)-1 {
		return Symbol{}, fmt.Errorf("%w: function symbol must be <import-path>.<Name>", ErrInvalidSymbol)
	}
	importPath, name := value[:dot], value[dot+1:]
	if err := validateImportPath(importPath); err != nil {
		return Symbol{}, err
	}
	if err := validateGoIdentifier(name, "function name"); err != nil {
		return Symbol{}, err
	}
	return Symbol{importPath: importPath, name: name}, nil
}

// parseMethodSymbol parses "<importPath>.(*Recv).Name" or "<importPath>.(Recv).Name".
func parseMethodSymbol(value string) (Symbol, error) {
	open := strings.Index(value, ".(")
	if open < 0 {
		return Symbol{}, fmt.Errorf("%w: malformed method symbol %q", ErrInvalidSymbol, value)
	}
	importPath := value[:open]
	if err := validateImportPath(importPath); err != nil {
		return Symbol{}, err
	}
	rest := value[open+2:] // after ".("
	close := strings.Index(rest, ").")
	if close < 0 {
		return Symbol{}, fmt.Errorf("%w: malformed method receiver in %q", ErrInvalidSymbol, value)
	}
	recvPart := rest[:close]
	name := rest[close+2:]
	if strings.ContainsAny(name, "().") {
		return Symbol{}, fmt.Errorf("%w: malformed method name in %q", ErrInvalidSymbol, value)
	}
	pointer := false
	if strings.HasPrefix(recvPart, "*") {
		pointer = true
		recvPart = recvPart[1:]
	}
	if err := validateGoIdentifier(recvPart, "receiver type"); err != nil {
		return Symbol{}, err
	}
	if err := validateGoIdentifier(name, "method name"); err != nil {
		return Symbol{}, err
	}
	return Symbol{importPath: importPath, receiver: recvPart, name: name, pointer: pointer}, nil
}

// validateImportPath checks that path is a plausible Go import path and not a
// relative or filesystem path.
func validateImportPath(path string) error {
	if path == "" {
		return fmt.Errorf("%w: import path is empty", ErrInvalidSymbol)
	}
	if strings.HasPrefix(path, "/") {
		return fmt.Errorf("%w: import path must not be absolute", ErrInvalidSymbol)
	}
	if strings.HasPrefix(path, ".") {
		return fmt.Errorf("%w: import path must not be relative", ErrInvalidSymbol)
	}
	if strings.ContainsAny(path, `\:`) {
		return fmt.Errorf("%w: import path looks like a filesystem path", ErrInvalidSymbol)
	}
	for _, seg := range strings.Split(path, "/") {
		if seg == "" || seg == "." || seg == ".." {
			return fmt.Errorf("%w: import path has an invalid segment", ErrInvalidSymbol)
		}
		for _, r := range seg {
			if !isImportPathRune(r) {
				return fmt.Errorf("%w: import path contains invalid character %q", ErrInvalidSymbol, r)
			}
		}
	}
	return nil
}

// isImportPathRune reports whether r is allowed in an import path segment.
func isImportPathRune(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
		(r >= '0' && r <= '9') || r == '.' || r == '_' || r == '-' || r == '~'
}

// validateGoIdentifier checks that s is a valid Go identifier.
func validateGoIdentifier(s, what string) error {
	if s == "" {
		return fmt.Errorf("%w: %s is empty", ErrInvalidSymbol, what)
	}
	for i, r := range s {
		switch {
		case r == '_':
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z':
		case i > 0 && r >= '0' && r <= '9':
		default:
			return fmt.Errorf("%w: %s %q is not a valid Go identifier", ErrInvalidSymbol, what, s)
		}
	}
	return nil
}

// MustParseSymbol is like ParseSymbol but panics on error, for constants.
func MustParseSymbol(value string) Symbol {
	s, err := ParseSymbol(value)
	if err != nil {
		panic(fmt.Sprintf("operation.MustParseSymbol(%q): %v", value, err))
	}
	return s
}

// ImportPath returns the package import path.
func (s Symbol) ImportPath() string { return s.importPath }

// Receiver returns the receiver type name without a leading '*', or an empty
// string for a function symbol.
func (s Symbol) Receiver() string { return s.receiver }

// Name returns the function or method name.
func (s Symbol) Name() string { return s.name }

// PointerReceiver reports whether the method has a pointer receiver. It is
// always false for function symbols.
func (s Symbol) PointerReceiver() bool { return s.pointer }

// IsMethod reports whether the symbol references a method rather than a function.
func (s Symbol) IsMethod() bool { return s.receiver != "" }

// IsZero reports whether the symbol is the zero value.
func (s Symbol) IsZero() bool { return s == Symbol{} }

// Validate reports whether the symbol is well-formed.
func (s Symbol) Validate() error {
	if s.IsZero() {
		return fmt.Errorf("%w: symbol is empty", ErrInvalidSymbol)
	}
	_, err := ParseSymbol(s.String())
	return err
}

// String reconstructs the canonical textual form of the symbol.
func (s Symbol) String() string {
	if s.receiver == "" {
		return s.importPath + "." + s.name
	}
	if s.pointer {
		return s.importPath + ".(*" + s.receiver + ")." + s.name
	}
	return s.importPath + ".(" + s.receiver + ")." + s.name
}

// MarshalText implements encoding.TextMarshaler.
func (s Symbol) MarshalText() ([]byte, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}
	return []byte(s.String()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler with strict parsing.
func (s *Symbol) UnmarshalText(data []byte) error {
	parsed, err := ParseSymbol(string(data))
	if err != nil {
		return err
	}
	*s = parsed
	return nil
}
