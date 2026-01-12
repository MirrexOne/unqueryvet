package messages

import (
	"strings"
	"testing"
)

func TestMessageTypes(t *testing.T) {
	// All message types should be defined
	types := []MessageType{
		BasicSelectStar,
		AliasedWildcard,
		SQLBuilder,
		EmptySelect,
		Subquery,
		Concatenation,
		FormatString,
		StringBuilder,
	}

	for _, mt := range types {
		t.Run(string(mt), func(t *testing.T) {
			msg := getMessage(mt)
			if msg.Title == "" {
				t.Errorf("Message type %s should have a title", mt)
			}
		})
	}
}

func TestGetEnhancedMessageSimple(t *testing.T) {
	tests := []struct {
		msgType  MessageType
		contains string
	}{
		{BasicSelectStar, "avoid SELECT *"},
		{AliasedWildcard, "avoid SELECT alias.*"},
		{SQLBuilder, "avoid SELECT * in SQL builder"},
		{EmptySelect, "Select() without columns"},
		{Subquery, "avoid SELECT * in subquery"},
		{Concatenation, "avoid SELECT * in concatenated"},
		{FormatString, "avoid SELECT * in format string"},
		{StringBuilder, "avoid SELECT * when building"},
	}

	for _, tt := range tests {
		t.Run(string(tt.msgType), func(t *testing.T) {
			msg := GetEnhancedMessage(tt.msgType, false)
			if !strings.Contains(msg, tt.contains) {
				t.Errorf("GetEnhancedMessage(%s, false) = %q, should contain %q", tt.msgType, msg, tt.contains)
			}
		})
	}
}

func TestGetEnhancedMessageVerbose(t *testing.T) {
	msg := GetEnhancedMessage(BasicSelectStar, true)

	// Verbose message should contain more details
	if !strings.Contains(msg, "avoid SELECT *") {
		t.Error("Verbose message should contain title")
	}

	if !strings.Contains(msg, "Example fix:") {
		t.Error("Verbose message should contain example")
	}

	if !strings.Contains(msg, "Impact:") {
		t.Error("Verbose message should contain impact")
	}

	if !strings.Contains(msg, "Learn more:") {
		t.Error("Verbose message should contain learn more link")
	}

	if !strings.Contains(msg, DocsBaseURL) {
		t.Error("Verbose message should contain docs URL")
	}
}

func TestGetEnhancedMessageUnknownType(t *testing.T) {
	msg := GetEnhancedMessage(MessageType("unknown"), false)
	if !strings.Contains(msg, "avoid SELECT *") {
		t.Error("Unknown message type should return fallback message")
	}
}

func TestGetMessage(t *testing.T) {
	msg := getMessage(BasicSelectStar)

	if msg.Title == "" {
		t.Error("Message should have title")
	}

	if msg.Description == "" {
		t.Error("Message should have description")
	}

	if msg.Example == "" {
		t.Error("Message should have example")
	}

	if msg.Suggestion == "" {
		t.Error("Message should have suggestion")
	}

	if msg.Impact == "" {
		t.Error("Message should have impact")
	}

	if msg.LearnMore == "" {
		t.Error("Message should have learn more link")
	}
}

func TestFormatDiagnostic(t *testing.T) {
	tests := []struct {
		file     string
		line     int
		col      int
		verbose  bool
		contains []string
	}{
		{
			file:     "main.go",
			line:     10,
			col:      5,
			verbose:  false,
			contains: []string{"main.go:10:5:", "avoid SELECT *"},
		},
		{
			file:     "pkg/db/query.go",
			line:     42,
			col:      15,
			verbose:  false,
			contains: []string{"pkg/db/query.go:42:15:", "avoid SELECT *"},
		},
		{
			file:     "test.go",
			line:     1,
			col:      1,
			verbose:  true,
			contains: []string{"test.go:1:1:", "Example fix:", "Impact:"},
		},
	}

	for _, tt := range tests {
		name := tt.file
		t.Run(name, func(t *testing.T) {
			result := FormatDiagnostic(tt.file, tt.line, tt.col, BasicSelectStar, tt.verbose)
			for _, c := range tt.contains {
				if !strings.Contains(result, c) {
					t.Errorf("FormatDiagnostic() = %q, should contain %q", result, c)
				}
			}
		})
	}
}

func TestGetQuickFix(t *testing.T) {
	tests := []struct {
		name     string
		msgType  MessageType
		original string
		contains string
	}{
		{
			name:     "basic select star",
			msgType:  BasicSelectStar,
			original: "SELECT * FROM users",
			contains: "SELECT id",
		},
		{
			name:     "aliased wildcard",
			msgType:  AliasedWildcard,
			original: "SELECT t.* FROM users t",
			contains: "TODO",
		},
		{
			name:     "sql builder",
			msgType:  SQLBuilder,
			original: `Select("*")`,
			contains: `"id"`,
		},
		{
			name:     "empty select",
			msgType:  EmptySelect,
			original: "Select()",
			contains: `Select("id"`,
		},
		{
			name:     "unknown type",
			msgType:  MessageType("unknown"),
			original: "SELECT * FROM users",
			contains: "SELECT * FROM users",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetQuickFix(tt.msgType, tt.original)
			if !strings.Contains(result, tt.contains) {
				t.Errorf("GetQuickFix(%s, %q) = %q, should contain %q",
					tt.msgType, tt.original, result, tt.contains)
			}
		})
	}
}

func TestDiagnosticMessageStruct(t *testing.T) {
	msg := DiagnosticMessage{
		Title:       "Test Title",
		Description: "Test Description",
		Example:     "Test Example",
		Suggestion:  "Test Suggestion",
		Impact:      "Test Impact",
		LearnMore:   "https://example.com",
	}

	if msg.Title != "Test Title" {
		t.Error("Title not set correctly")
	}
	if msg.Description != "Test Description" {
		t.Error("Description not set correctly")
	}
	if msg.Example != "Test Example" {
		t.Error("Example not set correctly")
	}
	if msg.Suggestion != "Test Suggestion" {
		t.Error("Suggestion not set correctly")
	}
	if msg.Impact != "Test Impact" {
		t.Error("Impact not set correctly")
	}
	if msg.LearnMore != "https://example.com" {
		t.Error("LearnMore not set correctly")
	}
}

func TestDocsBaseURL(t *testing.T) {
	if DocsBaseURL == "" {
		t.Error("DocsBaseURL should not be empty")
	}

	if !strings.HasPrefix(DocsBaseURL, "https://") {
		t.Error("DocsBaseURL should be an HTTPS URL")
	}
}

func TestAllMessageTypesHaveCompleteInfo(t *testing.T) {
	types := []MessageType{
		BasicSelectStar,
		AliasedWildcard,
		SQLBuilder,
		EmptySelect,
		Subquery,
		Concatenation,
		FormatString,
		StringBuilder,
	}

	for _, mt := range types {
		t.Run(string(mt), func(t *testing.T) {
			msg := getMessage(mt)

			if msg.Title == "" {
				t.Errorf("Message type %s missing Title", mt)
			}
			if msg.Description == "" {
				t.Errorf("Message type %s missing Description", mt)
			}
			if msg.Example == "" {
				t.Errorf("Message type %s missing Example", mt)
			}
			if msg.Suggestion == "" {
				t.Errorf("Message type %s missing Suggestion", mt)
			}
			if msg.Impact == "" {
				t.Errorf("Message type %s missing Impact", mt)
			}
			if msg.LearnMore == "" {
				t.Errorf("Message type %s missing LearnMore", mt)
			}
		})
	}
}
