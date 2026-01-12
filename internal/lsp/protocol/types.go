// Package protocol defines the types and constants for the Language Server Protocol.
package protocol

import "encoding/json"

// JSON-RPC error codes
const (
	ParseError           = -32700
	InvalidRequest       = -32600
	MethodNotFound       = -32601
	InvalidParams        = -32602
	InternalError        = -32603
	ServerNotInitialized = -32002
	RequestCancelled     = -32800
	ContentModified      = -32801
)

// TextDocumentSyncKind defines how the client and server synchronize text documents.
type TextDocumentSyncKind int

const (
	TextDocumentSyncKindNone        TextDocumentSyncKind = 0
	TextDocumentSyncKindFull        TextDocumentSyncKind = 1
	TextDocumentSyncKindIncremental TextDocumentSyncKind = 2
)

// DiagnosticSeverity represents the severity of a diagnostic.
type DiagnosticSeverity int

const (
	DiagnosticSeverityError       DiagnosticSeverity = 1
	DiagnosticSeverityWarning     DiagnosticSeverity = 2
	DiagnosticSeverityInformation DiagnosticSeverity = 3
	DiagnosticSeverityHint        DiagnosticSeverity = 4
)

// DiagnosticTag represents diagnostic tags.
type DiagnosticTag int

const (
	DiagnosticTagUnnecessary DiagnosticTag = 1
	DiagnosticTagDeprecated  DiagnosticTag = 2
)

// CodeActionKind represents the kind of code action.
type CodeActionKind string

const (
	CodeActionKindQuickFix              CodeActionKind = "quickfix"
	CodeActionKindRefactor              CodeActionKind = "refactor"
	CodeActionKindRefactorExtract       CodeActionKind = "refactor.extract"
	CodeActionKindRefactorInline        CodeActionKind = "refactor.inline"
	CodeActionKindRefactorRewrite       CodeActionKind = "refactor.rewrite"
	CodeActionKindSource                CodeActionKind = "source"
	CodeActionKindSourceOrganizeImports CodeActionKind = "source.organizeImports"
	CodeActionKindSourceFixAll          CodeActionKind = "source.fixAll"
)

// CompletionItemKind represents the kind of completion item.
type CompletionItemKind int

const (
	CompletionItemKindText          CompletionItemKind = 1
	CompletionItemKindMethod        CompletionItemKind = 2
	CompletionItemKindFunction      CompletionItemKind = 3
	CompletionItemKindConstructor   CompletionItemKind = 4
	CompletionItemKindField         CompletionItemKind = 5
	CompletionItemKindVariable      CompletionItemKind = 6
	CompletionItemKindClass         CompletionItemKind = 7
	CompletionItemKindInterface     CompletionItemKind = 8
	CompletionItemKindModule        CompletionItemKind = 9
	CompletionItemKindProperty      CompletionItemKind = 10
	CompletionItemKindUnit          CompletionItemKind = 11
	CompletionItemKindValue         CompletionItemKind = 12
	CompletionItemKindEnum          CompletionItemKind = 13
	CompletionItemKindKeyword       CompletionItemKind = 14
	CompletionItemKindSnippet       CompletionItemKind = 15
	CompletionItemKindColor         CompletionItemKind = 16
	CompletionItemKindFile          CompletionItemKind = 17
	CompletionItemKindReference     CompletionItemKind = 18
	CompletionItemKindFolder        CompletionItemKind = 19
	CompletionItemKindEnumMember    CompletionItemKind = 20
	CompletionItemKindConstant      CompletionItemKind = 21
	CompletionItemKindStruct        CompletionItemKind = 22
	CompletionItemKindEvent         CompletionItemKind = 23
	CompletionItemKindOperator      CompletionItemKind = 24
	CompletionItemKindTypeParameter CompletionItemKind = 25
)

// MarkupKind describes the content type.
type MarkupKind string

const (
	MarkupKindPlainText MarkupKind = "plaintext"
	MarkupKindMarkdown  MarkupKind = "markdown"
)

// InsertTextFormat defines how the insert text should be interpreted.
type InsertTextFormat int

const (
	// InsertTextFormatPlainText is plain text format.
	InsertTextFormatPlainText InsertTextFormat = 1
	// InsertTextFormatSnippet is snippet format (supports placeholders like $1, ${1:default}).
	InsertTextFormatSnippet InsertTextFormat = 2
)

// Message is a JSON-RPC message.
type Message struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// Response is a JSON-RPC response.
type Response struct {
	JSONRPC string         `json:"jsonrpc"`
	ID      interface{}    `json:"id"`
	Result  interface{}    `json:"result,omitempty"`
	Error   *ResponseError `json:"error,omitempty"`
}

// ResponseError is a JSON-RPC error.
type ResponseError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Position in a text document expressed as zero-based line and character offset.
type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

// Range in a text document expressed as start and end positions.
type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

// Location represents a location inside a resource.
type Location struct {
	URI   string `json:"uri"`
	Range Range  `json:"range"`
}

// TextDocumentIdentifier identifies a text document.
type TextDocumentIdentifier struct {
	URI string `json:"uri"`
}

// VersionedTextDocumentIdentifier identifies a specific version of a text document.
type VersionedTextDocumentIdentifier struct {
	TextDocumentIdentifier
	Version int `json:"version"`
}

// TextDocumentItem represents a text document.
type TextDocumentItem struct {
	URI        string `json:"uri"`
	LanguageID string `json:"languageId"`
	Version    int    `json:"version"`
	Text       string `json:"text"`
}

// TextDocumentPositionParams is a parameter literal used in requests to pass a text document and a position.
type TextDocumentPositionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
}

// TextEdit represents a textual edit applicable to a text document.
type TextEdit struct {
	Range   Range  `json:"range"`
	NewText string `json:"newText"`
}

// WorkspaceEdit represents changes to many resources.
type WorkspaceEdit struct {
	Changes         map[string][]TextEdit `json:"changes,omitempty"`
	DocumentChanges []TextDocumentEdit    `json:"documentChanges,omitempty"`
}

// TextDocumentEdit represents edits to a text document.
type TextDocumentEdit struct {
	TextDocument VersionedTextDocumentIdentifier `json:"textDocument"`
	Edits        []TextEdit                      `json:"edits"`
}

// TextDocumentContentChangeEvent describes content changes to a text document.
type TextDocumentContentChangeEvent struct {
	Range       *Range `json:"range,omitempty"`
	RangeLength int    `json:"rangeLength,omitempty"`
	Text        string `json:"text"`
}

// Diagnostic represents a diagnostic, such as a compiler error.
type Diagnostic struct {
	Range              Range                          `json:"range"`
	Severity           DiagnosticSeverity             `json:"severity,omitempty"`
	Code               interface{}                    `json:"code,omitempty"`
	CodeDescription    *CodeDescription               `json:"codeDescription,omitempty"`
	Source             string                         `json:"source,omitempty"`
	Message            string                         `json:"message"`
	Tags               []DiagnosticTag                `json:"tags,omitempty"`
	RelatedInformation []DiagnosticRelatedInformation `json:"relatedInformation,omitempty"`
	Data               interface{}                    `json:"data,omitempty"`
}

// CodeDescription represents a description for a diagnostic code.
type CodeDescription struct {
	Href string `json:"href"`
}

// DiagnosticRelatedInformation represents a related message and source code location.
type DiagnosticRelatedInformation struct {
	Location Location `json:"location"`
	Message  string   `json:"message"`
}

// CompletionItem represents a completion item.
type CompletionItem struct {
	Label               string             `json:"label"`
	Kind                CompletionItemKind `json:"kind,omitempty"`
	Tags                []int              `json:"tags,omitempty"`
	Detail              string             `json:"detail,omitempty"`
	Documentation       interface{}        `json:"documentation,omitempty"`
	Deprecated          bool               `json:"deprecated,omitempty"`
	Preselect           bool               `json:"preselect,omitempty"`
	SortText            string             `json:"sortText,omitempty"`
	FilterText          string             `json:"filterText,omitempty"`
	InsertText          string             `json:"insertText,omitempty"`
	InsertTextFormat    InsertTextFormat   `json:"insertTextFormat,omitempty"`
	InsertTextMode      int                `json:"insertTextMode,omitempty"`
	TextEdit            *TextEdit          `json:"textEdit,omitempty"`
	AdditionalTextEdits []TextEdit         `json:"additionalTextEdits,omitempty"`
	CommitCharacters    []string           `json:"commitCharacters,omitempty"`
	Command             *Command           `json:"command,omitempty"`
	Data                interface{}        `json:"data,omitempty"`
}

// Command represents a reference to a command.
type Command struct {
	Title     string        `json:"title"`
	Command   string        `json:"command"`
	Arguments []interface{} `json:"arguments,omitempty"`
}

// CodeAction represents a code action.
type CodeAction struct {
	Title       string         `json:"title"`
	Kind        CodeActionKind `json:"kind,omitempty"`
	Diagnostics []Diagnostic   `json:"diagnostics,omitempty"`
	IsPreferred bool           `json:"isPreferred,omitempty"`
	Disabled    *struct {
		Reason string `json:"reason"`
	} `json:"disabled,omitempty"`
	Edit    *WorkspaceEdit `json:"edit,omitempty"`
	Command *Command       `json:"command,omitempty"`
	Data    interface{}    `json:"data,omitempty"`
}

// Hover represents hover information.
type Hover struct {
	Contents MarkupContent `json:"contents"`
	Range    *Range        `json:"range,omitempty"`
}

// MarkupContent represents content with markup.
type MarkupContent struct {
	Kind  MarkupKind `json:"kind"`
	Value string     `json:"value"`
}

// ClientInfo information about the client.
type ClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
}

// ServerInfo information about the server.
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
}

// ClientCapabilities defines capabilities provided by the client.
type ClientCapabilities struct {
	Workspace    *WorkspaceClientCapabilities    `json:"workspace,omitempty"`
	TextDocument *TextDocumentClientCapabilities `json:"textDocument,omitempty"`
	Window       *WindowClientCapabilities       `json:"window,omitempty"`
	General      *GeneralClientCapabilities      `json:"general,omitempty"`
	Experimental interface{}                     `json:"experimental,omitempty"`
}

// WorkspaceClientCapabilities defines workspace client capabilities.
type WorkspaceClientCapabilities struct {
	ApplyEdit              bool        `json:"applyEdit,omitempty"`
	WorkspaceEdit          interface{} `json:"workspaceEdit,omitempty"`
	DidChangeConfiguration interface{} `json:"didChangeConfiguration,omitempty"`
	DidChangeWatchedFiles  interface{} `json:"didChangeWatchedFiles,omitempty"`
	Symbol                 interface{} `json:"symbol,omitempty"`
	ExecuteCommand         interface{} `json:"executeCommand,omitempty"`
	WorkspaceFolders       bool        `json:"workspaceFolders,omitempty"`
	Configuration          bool        `json:"configuration,omitempty"`
}

// TextDocumentClientCapabilities defines text document client capabilities.
type TextDocumentClientCapabilities struct {
	Synchronization    interface{} `json:"synchronization,omitempty"`
	Completion         interface{} `json:"completion,omitempty"`
	Hover              interface{} `json:"hover,omitempty"`
	SignatureHelp      interface{} `json:"signatureHelp,omitempty"`
	Declaration        interface{} `json:"declaration,omitempty"`
	Definition         interface{} `json:"definition,omitempty"`
	TypeDefinition     interface{} `json:"typeDefinition,omitempty"`
	Implementation     interface{} `json:"implementation,omitempty"`
	References         interface{} `json:"references,omitempty"`
	DocumentHighlight  interface{} `json:"documentHighlight,omitempty"`
	DocumentSymbol     interface{} `json:"documentSymbol,omitempty"`
	CodeAction         interface{} `json:"codeAction,omitempty"`
	CodeLens           interface{} `json:"codeLens,omitempty"`
	DocumentLink       interface{} `json:"documentLink,omitempty"`
	ColorProvider      interface{} `json:"colorProvider,omitempty"`
	Formatting         interface{} `json:"formatting,omitempty"`
	RangeFormatting    interface{} `json:"rangeFormatting,omitempty"`
	OnTypeFormatting   interface{} `json:"onTypeFormatting,omitempty"`
	Rename             interface{} `json:"rename,omitempty"`
	PublishDiagnostics interface{} `json:"publishDiagnostics,omitempty"`
	FoldingRange       interface{} `json:"foldingRange,omitempty"`
	SelectionRange     interface{} `json:"selectionRange,omitempty"`
}

// WindowClientCapabilities defines window client capabilities.
type WindowClientCapabilities struct {
	WorkDoneProgress bool        `json:"workDoneProgress,omitempty"`
	ShowMessage      interface{} `json:"showMessage,omitempty"`
	ShowDocument     interface{} `json:"showDocument,omitempty"`
}

// GeneralClientCapabilities defines general client capabilities.
type GeneralClientCapabilities struct {
	StaleRequestSupport interface{} `json:"staleRequestSupport,omitempty"`
	RegularExpressions  interface{} `json:"regularExpressions,omitempty"`
	Markdown            interface{} `json:"markdown,omitempty"`
}

// ServerCapabilities defines capabilities provided by the server.
type ServerCapabilities struct {
	TextDocumentSync                 *TextDocumentSyncOptions `json:"textDocumentSync,omitempty"`
	CompletionProvider               *CompletionOptions       `json:"completionProvider,omitempty"`
	HoverProvider                    interface{}              `json:"hoverProvider,omitempty"`
	SignatureHelpProvider            interface{}              `json:"signatureHelpProvider,omitempty"`
	DeclarationProvider              interface{}              `json:"declarationProvider,omitempty"`
	DefinitionProvider               interface{}              `json:"definitionProvider,omitempty"`
	TypeDefinitionProvider           interface{}              `json:"typeDefinitionProvider,omitempty"`
	ImplementationProvider           interface{}              `json:"implementationProvider,omitempty"`
	ReferencesProvider               interface{}              `json:"referencesProvider,omitempty"`
	DocumentHighlightProvider        interface{}              `json:"documentHighlightProvider,omitempty"`
	DocumentSymbolProvider           interface{}              `json:"documentSymbolProvider,omitempty"`
	CodeActionProvider               *CodeActionOptions       `json:"codeActionProvider,omitempty"`
	CodeLensProvider                 interface{}              `json:"codeLensProvider,omitempty"`
	DocumentLinkProvider             interface{}              `json:"documentLinkProvider,omitempty"`
	ColorProvider                    interface{}              `json:"colorProvider,omitempty"`
	DocumentFormattingProvider       interface{}              `json:"documentFormattingProvider,omitempty"`
	DocumentRangeFormattingProvider  interface{}              `json:"documentRangeFormattingProvider,omitempty"`
	DocumentOnTypeFormattingProvider interface{}              `json:"documentOnTypeFormattingProvider,omitempty"`
	RenameProvider                   interface{}              `json:"renameProvider,omitempty"`
	FoldingRangeProvider             interface{}              `json:"foldingRangeProvider,omitempty"`
	ExecuteCommandProvider           interface{}              `json:"executeCommandProvider,omitempty"`
	SelectionRangeProvider           interface{}              `json:"selectionRangeProvider,omitempty"`
	WorkspaceSymbolProvider          interface{}              `json:"workspaceSymbolProvider,omitempty"`
	Workspace                        interface{}              `json:"workspace,omitempty"`
	DiagnosticProvider               *DiagnosticOptions       `json:"diagnosticProvider,omitempty"`
	Experimental                     interface{}              `json:"experimental,omitempty"`
}

// TextDocumentSyncOptions defines options for text document sync.
type TextDocumentSyncOptions struct {
	OpenClose         bool                 `json:"openClose,omitempty"`
	Change            TextDocumentSyncKind `json:"change,omitempty"`
	WillSave          bool                 `json:"willSave,omitempty"`
	WillSaveWaitUntil bool                 `json:"willSaveWaitUntil,omitempty"`
	Save              *SaveOptions         `json:"save,omitempty"`
}

// SaveOptions defines options for save operations.
type SaveOptions struct {
	IncludeText bool `json:"includeText,omitempty"`
}

// CompletionOptions defines options for completion.
type CompletionOptions struct {
	TriggerCharacters   []string `json:"triggerCharacters,omitempty"`
	AllCommitCharacters []string `json:"allCommitCharacters,omitempty"`
	ResolveProvider     bool     `json:"resolveProvider,omitempty"`
}

// CodeActionOptions defines options for code actions.
type CodeActionOptions struct {
	CodeActionKinds []CodeActionKind `json:"codeActionKinds,omitempty"`
	ResolveProvider bool             `json:"resolveProvider,omitempty"`
}

// DiagnosticOptions defines options for diagnostics.
type DiagnosticOptions struct {
	Identifier            string `json:"identifier,omitempty"`
	InterFileDependencies bool   `json:"interFileDependencies,omitempty"`
	WorkspaceDiagnostics  bool   `json:"workspaceDiagnostics,omitempty"`
}

// InitializeParams are parameters for the initialize request.
type InitializeParams struct {
	ProcessID             *int               `json:"processId"`
	ClientInfo            ClientInfo         `json:"clientInfo,omitempty"`
	Locale                string             `json:"locale,omitempty"`
	RootPath              string             `json:"rootPath,omitempty"`
	RootURI               string             `json:"rootUri,omitempty"`
	InitializationOptions json.RawMessage    `json:"initializationOptions,omitempty"`
	Capabilities          ClientCapabilities `json:"capabilities"`
	Trace                 string             `json:"trace,omitempty"`
	WorkspaceFolders      []WorkspaceFolder  `json:"workspaceFolders,omitempty"`
}

// WorkspaceFolder represents a workspace folder.
type WorkspaceFolder struct {
	URI  string `json:"uri"`
	Name string `json:"name"`
}

// InitializeResult is the result of the initialize request.
type InitializeResult struct {
	Capabilities ServerCapabilities `json:"capabilities"`
	ServerInfo   *ServerInfo        `json:"serverInfo,omitempty"`
}

// DidOpenTextDocumentParams are parameters for textDocument/didOpen.
type DidOpenTextDocumentParams struct {
	TextDocument TextDocumentItem `json:"textDocument"`
}

// DidChangeTextDocumentParams are parameters for textDocument/didChange.
type DidChangeTextDocumentParams struct {
	TextDocument   VersionedTextDocumentIdentifier  `json:"textDocument"`
	ContentChanges []TextDocumentContentChangeEvent `json:"contentChanges"`
}

// DidCloseTextDocumentParams are parameters for textDocument/didClose.
type DidCloseTextDocumentParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

// DidSaveTextDocumentParams are parameters for textDocument/didSave.
type DidSaveTextDocumentParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Text         string                 `json:"text,omitempty"`
}

// HoverParams are parameters for textDocument/hover.
type HoverParams struct {
	TextDocumentPositionParams
}

// CompletionParams are parameters for textDocument/completion.
type CompletionParams struct {
	TextDocumentPositionParams
	Context *CompletionContext `json:"context,omitempty"`
}

// CompletionContext contains additional information about the context of completion.
type CompletionContext struct {
	TriggerKind      int    `json:"triggerKind"`
	TriggerCharacter string `json:"triggerCharacter,omitempty"`
}

// CodeActionParams are parameters for textDocument/codeAction.
type CodeActionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Range        Range                  `json:"range"`
	Context      CodeActionContext      `json:"context"`
}

// CodeActionContext contains additional diagnostic information.
type CodeActionContext struct {
	Diagnostics []Diagnostic     `json:"diagnostics"`
	Only        []CodeActionKind `json:"only,omitempty"`
}

// PublishDiagnosticsParams are parameters for textDocument/publishDiagnostics.
type PublishDiagnosticsParams struct {
	URI         string       `json:"uri"`
	Version     int          `json:"version,omitempty"`
	Diagnostics []Diagnostic `json:"diagnostics"`
}
