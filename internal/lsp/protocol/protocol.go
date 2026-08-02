// Package protocol defines the subset of Language Server Protocol 3.17 types
// that BatchWeaver's language server and gopls proxy use. The types are written
// by hand from the public LSP specification; no code is imported or copied from
// gopls (see docs/adr/0072-standalone-lsp-server.md). Positions use the LSP
// convention of zero-based lines and UTF-16 code-unit character offsets unless a
// different position encoding is negotiated.
package protocol

import "encoding/json"

// DocumentURI is an LSP document URI (for example "file:///path/to/x.go").
type DocumentURI = string

// Position is a zero-based line and character offset.
type Position struct {
	Line      uint32 `json:"line"`
	Character uint32 `json:"character"`
}

// Range is an inclusive-start, exclusive-end span.
type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

// Location is a range within a document.
type Location struct {
	URI   DocumentURI `json:"uri"`
	Range Range       `json:"range"`
}

// TextEdit replaces Range with NewText.
type TextEdit struct {
	Range   Range  `json:"range"`
	NewText string `json:"newText"`
}

// DiagnosticSeverity levels.
type DiagnosticSeverity int

// Diagnostic severity levels defined by LSP.
const (
	SeverityError       DiagnosticSeverity = 1
	SeverityWarning     DiagnosticSeverity = 2
	SeverityInformation DiagnosticSeverity = 3
	SeverityHint        DiagnosticSeverity = 4
)

// DiagnosticTag values.
type DiagnosticTag int

// Diagnostic tags defined by LSP.
const (
	TagUnnecessary DiagnosticTag = 1
	TagDeprecated  DiagnosticTag = 2
)

// CodeDescription links a diagnostic code to documentation.
type CodeDescription struct {
	Href string `json:"href"`
}

// DiagnosticRelatedInformation points at a related location.
type DiagnosticRelatedInformation struct {
	Location Location `json:"location"`
	Message  string   `json:"message"`
}

// Diagnostic is a single diagnostic.
type Diagnostic struct {
	Range              Range                          `json:"range"`
	Severity           DiagnosticSeverity             `json:"severity,omitempty"`
	Code               string                         `json:"code,omitempty"`
	CodeDescription    *CodeDescription               `json:"codeDescription,omitempty"`
	Source             string                         `json:"source,omitempty"`
	Message            string                         `json:"message"`
	Tags               []DiagnosticTag                `json:"tags,omitempty"`
	RelatedInformation []DiagnosticRelatedInformation `json:"relatedInformation,omitempty"`
	Data               json.RawMessage                `json:"data,omitempty"`
}

// PublishDiagnosticsParams is the textDocument/publishDiagnostics payload.
type PublishDiagnosticsParams struct {
	URI         DocumentURI  `json:"uri"`
	Version     int32        `json:"version,omitempty"`
	Diagnostics []Diagnostic `json:"diagnostics"`
}

// ClientInfo identifies the client.
type ClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
}

// ServerInfo identifies the server.
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
}

// WorkspaceFolder is one root of a multi-root workspace.
type WorkspaceFolder struct {
	URI  DocumentURI `json:"uri"`
	Name string      `json:"name"`
}

// InitializeParams is a subset of the initialize request payload. Capabilities
// and InitializationOptions are kept raw so the server tolerates unknown fields.
type InitializeParams struct {
	ProcessID             int               `json:"processId,omitempty"`
	ClientInfo            *ClientInfo       `json:"clientInfo,omitempty"`
	RootURI               DocumentURI       `json:"rootUri,omitempty"`
	WorkspaceFolders      []WorkspaceFolder `json:"workspaceFolders,omitempty"`
	Capabilities          json.RawMessage   `json:"capabilities,omitempty"`
	InitializationOptions json.RawMessage   `json:"initializationOptions,omitempty"`
	Trace                 string            `json:"trace,omitempty"`
}

// InitializeResult is the initialize response.
type InitializeResult struct {
	Capabilities ServerCapabilities `json:"capabilities"`
	ServerInfo   *ServerInfo        `json:"serverInfo,omitempty"`
}

// TextDocumentSyncKind values.
type TextDocumentSyncKind int

// Text document synchronization kinds.
const (
	SyncNone        TextDocumentSyncKind = 0
	SyncFull        TextDocumentSyncKind = 1
	SyncIncremental TextDocumentSyncKind = 2
)

// TextDocumentSyncOptions configures text synchronization.
type TextDocumentSyncOptions struct {
	OpenClose bool                 `json:"openClose"`
	Change    TextDocumentSyncKind `json:"change"`
	Save      bool                 `json:"save,omitempty"`
}

// CodeActionOptions advertises code-action support.
type CodeActionOptions struct {
	CodeActionKinds []string `json:"codeActionKinds,omitempty"`
	ResolveProvider bool     `json:"resolveProvider,omitempty"`
}

// CodeLensOptions advertises code-lens support.
type CodeLensOptions struct {
	ResolveProvider bool `json:"resolveProvider,omitempty"`
}

// ExecuteCommandOptions advertises the supported commands.
type ExecuteCommandOptions struct {
	Commands []string `json:"commands"`
}

// WorkspaceFoldersServerCapabilities advertises multi-root support.
type WorkspaceFoldersServerCapabilities struct {
	Supported           bool `json:"supported"`
	ChangeNotifications bool `json:"changeNotifications,omitempty"`
}

// WorkspaceServerCapabilities groups workspace-level capabilities.
type WorkspaceServerCapabilities struct {
	WorkspaceFolders *WorkspaceFoldersServerCapabilities `json:"workspaceFolders,omitempty"`
}

// ServerCapabilities advertises exactly the features the server implements.
type ServerCapabilities struct {
	PositionEncoding       string                       `json:"positionEncoding,omitempty"`
	TextDocumentSync       *TextDocumentSyncOptions     `json:"textDocumentSync,omitempty"`
	HoverProvider          bool                         `json:"hoverProvider,omitempty"`
	CodeActionProvider     *CodeActionOptions           `json:"codeActionProvider,omitempty"`
	CodeLensProvider       *CodeLensOptions             `json:"codeLensProvider,omitempty"`
	ExecuteCommandProvider *ExecuteCommandOptions       `json:"executeCommandProvider,omitempty"`
	DefinitionProvider     bool                         `json:"definitionProvider,omitempty"`
	ReferencesProvider     bool                         `json:"referencesProvider,omitempty"`
	DocumentSymbolProvider bool                         `json:"documentSymbolProvider,omitempty"`
	InlayHintProvider      bool                         `json:"inlayHintProvider,omitempty"`
	Workspace              *WorkspaceServerCapabilities `json:"workspace,omitempty"`
}

// TextDocumentItem is an opened document.
type TextDocumentItem struct {
	URI        DocumentURI `json:"uri"`
	LanguageID string      `json:"languageId"`
	Version    int32       `json:"version"`
	Text       string      `json:"text"`
}

// TextDocumentIdentifier names a document.
type TextDocumentIdentifier struct {
	URI DocumentURI `json:"uri"`
}

// VersionedTextDocumentIdentifier names a document with its version.
type VersionedTextDocumentIdentifier struct {
	URI     DocumentURI `json:"uri"`
	Version int32       `json:"version"`
}

// DidOpenTextDocumentParams is textDocument/didOpen.
type DidOpenTextDocumentParams struct {
	TextDocument TextDocumentItem `json:"textDocument"`
}

// TextDocumentContentChangeEvent is one change. An absent Range means a full
// replacement.
type TextDocumentContentChangeEvent struct {
	Range *Range `json:"range,omitempty"`
	Text  string `json:"text"`
}

// DidChangeTextDocumentParams is textDocument/didChange.
type DidChangeTextDocumentParams struct {
	TextDocument   VersionedTextDocumentIdentifier  `json:"textDocument"`
	ContentChanges []TextDocumentContentChangeEvent `json:"contentChanges"`
}

// DidCloseTextDocumentParams is textDocument/didClose.
type DidCloseTextDocumentParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

// DidSaveTextDocumentParams is textDocument/didSave.
type DidSaveTextDocumentParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Text         string                 `json:"text,omitempty"`
}

// TextDocumentPositionParams is the common position request payload.
type TextDocumentPositionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
}

// MarkupContent is Markdown or plaintext content.
type MarkupContent struct {
	Kind  string `json:"kind"`
	Value string `json:"value"`
}

// Hover is a hover response.
type Hover struct {
	Contents MarkupContent `json:"contents"`
	Range    *Range        `json:"range,omitempty"`
}

// Command is an executable command reference.
type Command struct {
	Title     string            `json:"title"`
	Command   string            `json:"command"`
	Arguments []json.RawMessage `json:"arguments,omitempty"`
}

// CodeActionKinds used by BatchWeaver.
const (
	CodeActionQuickFix              = "quickfix"
	CodeActionRefactorRewrite       = "refactor.rewrite"
	CodeActionSourceBatchWeaver     = "source.batchweaver"
	CodeActionSourceFixAllBWSafe    = "source.fixAll.batchweaver-safe"
	CodeActionRefactorRewriteBWName = "refactor.rewrite.batchweaver"
)

// CodeActionContext accompanies a code-action request.
type CodeActionContext struct {
	Diagnostics []Diagnostic `json:"diagnostics"`
	Only        []string     `json:"only,omitempty"`
}

// CodeActionParams is textDocument/codeAction.
type CodeActionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Range        Range                  `json:"range"`
	Context      CodeActionContext      `json:"context"`
}

// WorkspaceEdit is a set of document edits.
type WorkspaceEdit struct {
	Changes map[DocumentURI][]TextEdit `json:"changes,omitempty"`
}

// CodeAction is an offered action.
type CodeAction struct {
	Title       string          `json:"title"`
	Kind        string          `json:"kind,omitempty"`
	Diagnostics []Diagnostic    `json:"diagnostics,omitempty"`
	IsPreferred bool            `json:"isPreferred,omitempty"`
	Edit        *WorkspaceEdit  `json:"edit,omitempty"`
	Command     *Command        `json:"command,omitempty"`
	Data        json.RawMessage `json:"data,omitempty"`
}

// CodeLensParams is textDocument/codeLens.
type CodeLensParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

// CodeLens is a single lens.
type CodeLens struct {
	Range   Range           `json:"range"`
	Command *Command        `json:"command,omitempty"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// ExecuteCommandParams is workspace/executeCommand.
type ExecuteCommandParams struct {
	Command   string            `json:"command"`
	Arguments []json.RawMessage `json:"arguments,omitempty"`
}

// InlayHintParams is textDocument/inlayHint.
type InlayHintParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Range        Range                  `json:"range"`
}

// InlayHintKind values.
type InlayHintKind int

// Inlay hint kinds.
const (
	InlayHintType      InlayHintKind = 1
	InlayHintParameter InlayHintKind = 2
)

// InlayHint is a single inlay hint.
type InlayHint struct {
	Position     Position      `json:"position"`
	Label        string        `json:"label"`
	Kind         InlayHintKind `json:"kind,omitempty"`
	PaddingLeft  bool          `json:"paddingLeft,omitempty"`
	PaddingRight bool          `json:"paddingRight,omitempty"`
}

// ReferenceContext accompanies a references request.
type ReferenceContext struct {
	IncludeDeclaration bool `json:"includeDeclaration"`
}

// ReferenceParams is textDocument/references.
type ReferenceParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
	Context      ReferenceContext       `json:"context"`
}

// DocumentSymbolParams is textDocument/documentSymbol.
type DocumentSymbolParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

// SymbolKind values used by BatchWeaver symbols.
type SymbolKind int

// Symbol kinds used by BatchWeaver.
const (
	SymbolFunction SymbolKind = 12
	SymbolObject   SymbolKind = 19
)

// DocumentSymbol is a hierarchical symbol.
type DocumentSymbol struct {
	Name           string           `json:"name"`
	Detail         string           `json:"detail,omitempty"`
	Kind           SymbolKind       `json:"kind"`
	Range          Range            `json:"range"`
	SelectionRange Range            `json:"selectionRange"`
	Children       []DocumentSymbol `json:"children,omitempty"`
}

// MessageType values for window/showMessage.
type MessageType int

// Message types for window/showMessage.
const (
	MessageError   MessageType = 1
	MessageWarning MessageType = 2
	MessageInfo    MessageType = 3
	MessageLog     MessageType = 4
)

// ShowMessageParams is window/showMessage.
type ShowMessageParams struct {
	Type    MessageType `json:"type"`
	Message string      `json:"message"`
}

// ApplyWorkspaceEditParams is workspace/applyEdit (server to client).
type ApplyWorkspaceEditParams struct {
	Label string        `json:"label,omitempty"`
	Edit  WorkspaceEdit `json:"edit"`
}

// ApplyWorkspaceEditResult is the workspace/applyEdit response.
type ApplyWorkspaceEditResult struct {
	Applied       bool   `json:"applied"`
	FailureReason string `json:"failureReason,omitempty"`
}
