// Package lsp implements a Language Server Protocol server for unqueryvet.
// It provides real-time SQL analysis with diagnostics, code actions, and hover information.
package lsp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/MirrexOne/unqueryvet/internal/configloader"
	"github.com/MirrexOne/unqueryvet/internal/lsp/protocol"
)

// Server implements the Language Server Protocol for unqueryvet.
type Server struct {
	// reader reads JSON-RPC messages from the client (with Content-Length)
	reader *BaseReader
	// writer writes JSON-RPC messages to the client (with Content-Length)
	writer *BaseWriter
	// writerMu protects concurrent writes
	writerMu sync.Mutex

	// documents stores open documents by URI
	documents map[string]*Document
	// documentsMu protects the documents map
	documentsMu sync.RWMutex

	// analyzer performs SQL analysis
	analyzer *Analyzer

	// initialized indicates whether the server has been initialized
	initialized bool

	// shutdown indicates whether shutdown has been requested
	shutdown bool

	// logger for debug output
	logger io.Writer

	// capabilities reported by the client
	clientCapabilities protocol.ClientCapabilities
}

// Document represents an open text document.
type Document struct {
	URI        string
	LanguageID string
	Version    int
	Content    string
}

// NewServer creates a new LSP server.
func NewServer(in io.Reader, out, logger io.Writer) *Server {
	// Load configuration from file or use defaults
	cfg, err := configloader.LoadOrDefault("")
	if err != nil {
		// If config loading fails, use defaults
		cfg, _ = configloader.LoadOrDefault("")
	}

	return &Server{
		reader:    NewBaseReader(in),
		writer:    NewBaseWriter(out),
		documents: make(map[string]*Document),
		analyzer:  NewAnalyzerWithConfig(*cfg),
		logger:    logger,
	}
}

// NewStdioServer creates a new LSP server using stdin/stdout.
func NewStdioServer() *Server {
	return NewServer(os.Stdin, os.Stdout, os.Stderr)
}

// Run starts the main message loop.
func (s *Server) Run(ctx context.Context) error {
	s.log("unqueryvet LSP server started")

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err := s.handleMessage(ctx); err != nil {
				if err == io.EOF {
					return nil
				}
				s.log("Error handling message: %v", err)
			}
		}
	}
}

// handleMessage reads and processes a single JSON-RPC message.
func (s *Server) handleMessage(ctx context.Context) error {
	// Read message with Content-Length header
	data, err := s.reader.Read()
	if err != nil {
		return err
	}

	var msg protocol.Message
	if err := json.Unmarshal(data, &msg); err != nil {
		return fmt.Errorf("failed to unmarshal message: %w", err)
	}

	s.log("Received: method=%s, id=%v", msg.Method, msg.ID)

	// Handle the message based on method
	switch msg.Method {
	case "initialize":
		return s.handleInitialize(ctx, &msg)
	case "initialized":
		return s.handleInitialized(ctx, &msg)
	case "shutdown":
		return s.handleShutdown(ctx, &msg)
	case "exit":
		return s.handleExit(ctx, &msg)
	case "textDocument/didOpen":
		return s.handleTextDocumentDidOpen(ctx, &msg)
	case "textDocument/didChange":
		return s.handleTextDocumentDidChange(ctx, &msg)
	case "textDocument/didClose":
		return s.handleTextDocumentDidClose(ctx, &msg)
	case "textDocument/didSave":
		return s.handleTextDocumentDidSave(ctx, &msg)
	case "textDocument/hover":
		return s.handleTextDocumentHover(ctx, &msg)
	case "textDocument/codeAction":
		return s.handleTextDocumentCodeAction(ctx, &msg)
	case "textDocument/completion":
		return s.handleTextDocumentCompletion(ctx, &msg)
	case "textDocument/diagnostic":
		return s.handleTextDocumentDiagnostic(ctx, &msg)
	default:
		// Unknown method - send error for requests, ignore notifications
		if msg.ID != nil {
			return s.sendError(msg.ID, protocol.MethodNotFound, "Method not found: "+msg.Method)
		}
		return nil
	}
}

// InitializationOptions contains options sent by the client during initialization.
type InitializationOptions struct {
	CheckN1Queries    bool `json:"checkN1Queries"`
	CheckSQLInjection bool `json:"checkSQLInjection"`
	CheckSelectStar   bool `json:"checkSelectStar"`
}

// handleInitialize handles the initialize request.
func (s *Server) handleInitialize(ctx context.Context, msg *protocol.Message) error {
	var params protocol.InitializeParams
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		return s.sendError(msg.ID, protocol.ParseError, "Failed to parse initialize params")
	}

	s.clientCapabilities = params.Capabilities
	s.log("Client: %s %s", params.ClientInfo.Name, params.ClientInfo.Version)

	// Parse initialization options from client
	if params.InitializationOptions != nil {
		var opts InitializationOptions
		if err := json.Unmarshal(params.InitializationOptions, &opts); err == nil {
			s.log("Initialization options: N+1=%v, SQLI=%v, SELECT*=%v",
				opts.CheckN1Queries, opts.CheckSQLInjection, opts.CheckSelectStar)

			// Configure analyzer based on client options
			s.analyzer.SetN1Detection(opts.CheckN1Queries)
			s.analyzer.SetSQLInjectionDetection(opts.CheckSQLInjection)
		}
	}

	result := protocol.InitializeResult{
		Capabilities: protocol.ServerCapabilities{
			TextDocumentSync: &protocol.TextDocumentSyncOptions{
				OpenClose: true,
				Change:    protocol.TextDocumentSyncKindFull,
				Save: &protocol.SaveOptions{
					IncludeText: true,
				},
			},
			HoverProvider: true,
			CodeActionProvider: &protocol.CodeActionOptions{
				CodeActionKinds: []protocol.CodeActionKind{
					protocol.CodeActionKindQuickFix,
					protocol.CodeActionKindSourceFixAll,
				},
			},
			CompletionProvider: &protocol.CompletionOptions{
				TriggerCharacters: []string{"\"", "'", "`", "*"},
				ResolveProvider:   false,
			},
			DiagnosticProvider: &protocol.DiagnosticOptions{
				InterFileDependencies: false,
				WorkspaceDiagnostics:  false,
			},
		},
		ServerInfo: &protocol.ServerInfo{
			Name:    "unqueryvet-lsp",
			Version: "1.0.0",
		},
	}

	s.initialized = true
	return s.sendResult(msg.ID, result)
}

// handleInitialized handles the initialized notification.
func (s *Server) handleInitialized(ctx context.Context, msg *protocol.Message) error {
	s.log("Server initialized")
	return nil
}

// handleShutdown handles the shutdown request.
func (s *Server) handleShutdown(ctx context.Context, msg *protocol.Message) error {
	s.shutdown = true
	return s.sendResult(msg.ID, nil)
}

// handleExit handles the exit notification.
func (s *Server) handleExit(ctx context.Context, msg *protocol.Message) error {
	if s.shutdown {
		os.Exit(0)
	}
	os.Exit(1)
	return nil
}

// handleTextDocumentDidOpen handles textDocument/didOpen notification.
func (s *Server) handleTextDocumentDidOpen(ctx context.Context, msg *protocol.Message) error {
	var params protocol.DidOpenTextDocumentParams
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		return err
	}

	doc := &Document{
		URI:        params.TextDocument.URI,
		LanguageID: params.TextDocument.LanguageID,
		Version:    params.TextDocument.Version,
		Content:    params.TextDocument.Text,
	}

	s.documentsMu.Lock()
	s.documents[params.TextDocument.URI] = doc
	s.documentsMu.Unlock()

	s.log("Opened document: %s", params.TextDocument.URI)

	// Analyze and publish diagnostics
	return s.analyzeAndPublishDiagnostics(ctx, doc)
}

// handleTextDocumentDidChange handles textDocument/didChange notification.
func (s *Server) handleTextDocumentDidChange(ctx context.Context, msg *protocol.Message) error {
	var params protocol.DidChangeTextDocumentParams
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		return err
	}

	s.documentsMu.Lock()
	doc, ok := s.documents[params.TextDocument.URI]
	if ok && len(params.ContentChanges) > 0 {
		// For full sync, just replace the content
		doc.Content = params.ContentChanges[0].Text
		doc.Version = params.TextDocument.Version
	}
	s.documentsMu.Unlock()

	if ok {
		return s.analyzeAndPublishDiagnostics(ctx, doc)
	}
	return nil
}

// handleTextDocumentDidClose handles textDocument/didClose notification.
func (s *Server) handleTextDocumentDidClose(ctx context.Context, msg *protocol.Message) error {
	var params protocol.DidCloseTextDocumentParams
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		return err
	}

	s.documentsMu.Lock()
	delete(s.documents, params.TextDocument.URI)
	s.documentsMu.Unlock()

	// Clear diagnostics for the closed document
	return s.publishDiagnostics(params.TextDocument.URI, []protocol.Diagnostic{})
}

// handleTextDocumentDidSave handles textDocument/didSave notification.
func (s *Server) handleTextDocumentDidSave(ctx context.Context, msg *protocol.Message) error {
	var params protocol.DidSaveTextDocumentParams
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		return err
	}

	s.documentsMu.RLock()
	doc, ok := s.documents[params.TextDocument.URI]
	s.documentsMu.RUnlock()

	if ok {
		if params.Text != "" {
			doc.Content = params.Text
		}
		return s.analyzeAndPublishDiagnostics(ctx, doc)
	}
	return nil
}

// handleTextDocumentHover handles textDocument/hover request.
func (s *Server) handleTextDocumentHover(ctx context.Context, msg *protocol.Message) error {
	var params protocol.HoverParams
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		return s.sendError(msg.ID, protocol.ParseError, "Failed to parse hover params")
	}

	s.documentsMu.RLock()
	doc, ok := s.documents[params.TextDocument.URI]
	s.documentsMu.RUnlock()

	if !ok {
		return s.sendResult(msg.ID, nil)
	}

	hover := s.analyzer.GetHover(doc, params.Position)
	return s.sendResult(msg.ID, hover)
}

// handleTextDocumentCodeAction handles textDocument/codeAction request.
func (s *Server) handleTextDocumentCodeAction(ctx context.Context, msg *protocol.Message) error {
	var params protocol.CodeActionParams
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		return s.sendError(msg.ID, protocol.ParseError, "Failed to parse codeAction params")
	}

	s.documentsMu.RLock()
	doc, ok := s.documents[params.TextDocument.URI]
	s.documentsMu.RUnlock()

	if !ok {
		return s.sendResult(msg.ID, []protocol.CodeAction{})
	}

	actions := s.analyzer.GetCodeActions(doc, params)
	return s.sendResult(msg.ID, actions)
}

// handleTextDocumentCompletion handles textDocument/completion request.
func (s *Server) handleTextDocumentCompletion(ctx context.Context, msg *protocol.Message) error {
	var params protocol.CompletionParams
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		return s.sendError(msg.ID, protocol.ParseError, "Failed to parse completion params")
	}

	s.documentsMu.RLock()
	doc, ok := s.documents[params.TextDocument.URI]
	s.documentsMu.RUnlock()

	if !ok {
		return s.sendResult(msg.ID, []protocol.CompletionItem{})
	}

	items := s.analyzer.GetCompletions(doc, params.Position)
	return s.sendResult(msg.ID, items)
}

// handleTextDocumentDiagnostic handles textDocument/diagnostic request (LSP 3.17 pull diagnostics).
func (s *Server) handleTextDocumentDiagnostic(ctx context.Context, msg *protocol.Message) error {
	var params struct {
		TextDocument protocol.TextDocumentIdentifier `json:"textDocument"`
	}
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		return err
	}

	s.documentsMu.RLock()
	doc, ok := s.documents[params.TextDocument.URI]
	s.documentsMu.RUnlock()

	if !ok {
		// Document not found - return empty diagnostics
		return s.sendResult(msg.ID, map[string]interface{}{
			"kind":  "full",
			"items": []protocol.Diagnostic{},
		})
	}

	diagnostics := s.analyzer.Analyze(doc)
	return s.sendResult(msg.ID, map[string]interface{}{
		"kind":  "full",
		"items": diagnostics,
	})
}

// analyzeAndPublishDiagnostics analyzes a document and publishes diagnostics.
func (s *Server) analyzeAndPublishDiagnostics(ctx context.Context, doc *Document) error {
	diagnostics := s.analyzer.Analyze(doc)
	return s.publishDiagnostics(doc.URI, diagnostics)
}

// publishDiagnostics sends diagnostics to the client.
func (s *Server) publishDiagnostics(uri string, diagnostics []protocol.Diagnostic) error {
	params := protocol.PublishDiagnosticsParams{
		URI:         uri,
		Diagnostics: diagnostics,
	}
	return s.sendNotification("textDocument/publishDiagnostics", params)
}

// sendResult sends a successful response.
func (s *Server) sendResult(id, result interface{}) error {
	resp := protocol.Response{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
	return s.send(resp)
}

// sendError sends an error response.
func (s *Server) sendError(id interface{}, code int, message string) error {
	resp := protocol.Response{
		JSONRPC: "2.0",
		ID:      id,
		Error: &protocol.ResponseError{
			Code:    code,
			Message: message,
		},
	}
	return s.send(resp)
}

// sendNotification sends a notification to the client.
func (s *Server) sendNotification(method string, params interface{}) error {
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return err
	}
	notification := protocol.Message{
		JSONRPC: "2.0",
		Method:  method,
		Params:  paramsJSON,
	}
	return s.send(notification)
}

// send writes a message to the client.
func (s *Server) send(msg interface{}) error {
	s.writerMu.Lock()
	defer s.writerMu.Unlock()
	return s.writer.WriteJSON(msg)
}

// log writes a debug message to the logger.
func (s *Server) log(format string, args ...interface{}) {
	if s.logger != nil {
		fmt.Fprintf(s.logger, "[unqueryvet-lsp] "+format+"\n", args...)
	}
}
