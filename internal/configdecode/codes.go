package configdecode

import "github.com/Voskan/BatchWeaver/diagnostics"

// Configuration diagnostic codes (BWCFG range). These are shared across the
// configuration pipeline (decode, load, normalize, validate) and documented in
// docs/reference/diagnostic-codes.md. Once committed, a code keeps its meaning.
const (
	// CodeSyntax reports a YAML or JSON syntax error.
	CodeSyntax diagnostics.Code = "BWCFG001"
	// CodeDuplicateKey reports a duplicate mapping key.
	CodeDuplicateKey diagnostics.Code = "BWCFG002"
	// CodeMultipleDocuments reports more than one document in a source.
	CodeMultipleDocuments diagnostics.Code = "BWCFG003"
	// CodeTrailingContent reports trailing content after a JSON document.
	CodeTrailingContent diagnostics.Code = "BWCFG004"
	// CodeTypeMismatch reports a value of an unexpected kind.
	CodeTypeMismatch diagnostics.Code = "BWCFG005"
	// CodeInvalidEnum reports an unknown enumeration value.
	CodeInvalidEnum diagnostics.Code = "BWCFG006"
	// CodeInvalidDuration reports an invalid duration value.
	CodeInvalidDuration diagnostics.Code = "BWCFG007"
	// CodeInvalidByteSize reports an invalid byte-size value.
	CodeInvalidByteSize diagnostics.Code = "BWCFG008"
	// CodeUnsupportedConstruct reports a disallowed YAML construct.
	CodeUnsupportedConstruct diagnostics.Code = "BWCFG009"
	// CodeLimitExceeded reports that a size or count limit was exceeded.
	CodeLimitExceeded diagnostics.Code = "BWCFG010"
	// CodeIncludeError reports a failure to read or resolve an include.
	CodeIncludeError diagnostics.Code = "BWCFG011"
	// CodeIncludeCycle reports an include cycle.
	CodeIncludeCycle diagnostics.Code = "BWCFG012"
	// CodeAbsoluteInclude reports a disallowed absolute include path.
	CodeAbsoluteInclude diagnostics.Code = "BWCFG013"
	// CodeRemoteInclude reports a forbidden remote include.
	CodeRemoteInclude diagnostics.Code = "BWCFG014"
	// CodeMissingField reports a missing required field.
	CodeMissingField diagnostics.Code = "BWCFG015"
	// CodeUnsupportedVersion reports an unsupported schema version.
	CodeUnsupportedVersion diagnostics.Code = "BWCFG016"
	// CodeNotFound reports that no configuration file was found.
	CodeNotFound diagnostics.Code = "BWCFG017"
	// CodeAmbiguous reports multiple candidate configuration files.
	CodeAmbiguous diagnostics.Code = "BWCFG018"
	// CodeDuplicateOperation reports a conflicting operation definition.
	CodeDuplicateOperation diagnostics.Code = "BWCFG019"
	// CodeSecurity reports a configuration security violation.
	CodeSecurity diagnostics.Code = "BWCFG020"
	// CodeUnknownField reports an unknown field.
	CodeUnknownField diagnostics.Code = "BWCFG021"
	// CodeInvalidValue reports a value that is invalid for its field.
	CodeInvalidValue diagnostics.Code = "BWCFG022"
	// CodeSemantic reports a semantic validation failure.
	CodeSemantic diagnostics.Code = "BWCFG023"
)

const configSource = "config"

// diag builds an error-severity config diagnostic at pos.
func diag(code diagnostics.Code, pos diagnostics.Position, msg string) diagnostics.Diagnostic {
	return diagnostics.Diagnostic{
		Code:     code,
		Severity: diagnostics.SeverityError,
		Message:  msg,
		Source:   configSource,
		Range:    diagnostics.AtPosition(pos),
	}
}
