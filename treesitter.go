package main

import (
	"context"
	_ "embed"
	// TODO: use ysh, not bash
	tree_sitter_ysh "github.com/danyspin97/tree-sitter-ysh/bindings/go"
	tree_sitter "github.com/tree-sitter/go-tree-sitter"
	//tree_sitter_ysh "github.com/tree-sitter/tree-sitter-go/bindings/go"
	highlight "go.gopad.dev/go-tree-sitter-highlight"
	"log"
	"strings"
)

// We need at least 2 additional `.scm` files:

// https://github.com/nvim-treesitter/nvim-treesitter/blob/42fc28ba918343ebfd5565147a42a26580579482/queries/ecma/highlights.scm
//
//go:embed assets/highlights.scm
var highlights []byte

// https://github.com/nvim-treesitter/nvim-treesitter/blob/42fc28ba918343ebfd5565147a42a26580579482/queries/ecma/locals.scm
//
// /go:embed assets/locals.scm
var locals = []byte{}

// No injections for now!
// Would be: https://github.com/nvim-treesitter/nvim-treesitter/blob/42fc28ba918343ebfd5565147a42a26580579482/queries/ecma/injections.scm
// But this means we'd maintain a language stack with
var injections = []byte{}

// the colors for each type
// better approach would be reading a theme:
// https://tree-sitter.github.io/tree-sitter/cli/init-config.html#theme
var colorMap = map[string]string{
	"black":                 "\033[0m",
	"attribute":             "\033[0;31m",
	"comment":               "\033[0;32m",
	"constant":              "\033[0;33m",
	"constant.builtin":      "\033[0;33m",
	"constructor":           "\033[0;34m",
	"function":              "\033[0;37m",
	"function.builtin":      "\033[0;37m",
	"keyword":               "\033[0;35m",
	"module":                "\033[0;34m",
	"number":                "\033[0;33m",
	"operator":              "\033[0;36m",
	"property":              "\033[0;31m",
	"property.builtin":      "\033[0;31m",
	"punctuation":           "\033[0;36m",
	"punctuation.bracket":   "\033[0;36m",
	"punctuation.delimiter": "\033[0;36m",
	"punctuation.special":   "\033[0;36m",
	"string":                "\033[0;34m",
	"string.special":        "\033[0;35m",
	"tag":                   "\033[0;31m",
	"type":                  "\033[0;31m",
	"type.builtin":          "\033[0;31m",
	"variable":              "\033[0;33m",
	"variable.builtin":      "\033[0;33m",
	"variable.parameter":    "\033[0;34m",
}

type Highlighter struct {
	parser           *tree_sitter.Parser
	language         *tree_sitter.Language
	query            *tree_sitter.Query
	queryCursor      *tree_sitter.QueryCursor
	existingCaptures []string
	tree             *tree_sitter.Tree
	events           *[]highlight.Event
	Highlighter      *highlight.Highlighter
	cfg              *highlight.Configuration
}

func (h *Highlighter) Close() {
	h.parser.Close()
	h.query.Close()
	h.queryCursor.Close()
	h.tree.Close()
}

func NewHighlighter() *Highlighter {
	var h Highlighter

	// The types we want to highlight
	// Order is relevant
	captureNames := []string{
		"attribute",
		"comment",
		"constant",
		"constructor",
		"function",
		"keyword",
		"module",
		"number",
		"operator",
		"property",
		"property.builtin",
		"punctuation",
		"string",
		"tag",
		"type",
		"variable",
	}

	// set up the language and parser
	h.language = tree_sitter.NewLanguage(tree_sitter_ysh.Language())
	h.parser = tree_sitter.NewParser()
	h.parser.SetLanguage(h.language)

	// Parse and create the query
	h.query, _ = tree_sitter.NewQuery(h.language, string(highlights))
	h.queryCursor = tree_sitter.NewQueryCursor()

	// Get a list of all captures
	// More efficient would be to make the colormap based on int -> color instead of string -> color
	h.existingCaptures = h.query.CaptureNames()

	// Set up the highlighter and its config
	var err error
	h.cfg, err = highlight.NewConfiguration(h.language, "ysh", highlights, injections, locals)
	if err != nil {
		log.Fatal(err)
	}
	// What objects we care about
	h.cfg.Configure(captureNames)
	// Create the highlighter and
	h.Highlighter = highlight.New()
	// Start with an empty tree
	h.tree = h.parser.Parse([]byte("echo hello world"), nil)
	return &h
}

func (h *Highlighter) Highlight(_code string) string {
	if len(_code) < 3 {
		return _code
	}
	code := []byte(_code)
	events := h.Highlighter.Highlight(context.Background(), *h.cfg, code, func(name string) *highlight.Configuration {
		return nil
	})
	// The final string containing all highlights
	var s strings.Builder
	// Current highlighting type
	var t string
	for event, err := range events {
		if err != nil {
			log.Fatal(err)
		}

		switch e := event.(type) {

		//case highlight.EventLayerStart:
		// Only required if language injections exists
		// In which case we need to add the new language to the stack
		// and make sure to use the same language down below

		//case highlight.EventLayerEnd:
		// Only required if language injections exists
		// In which case we need to remove the language from the stack

		case highlight.EventCaptureStart:
			//log.Printf("New highlight capture %v, %v", e.Highlight, h.existingCaptures[e.Highlight])
			t = h.existingCaptures[e.Highlight]
		case highlight.EventCaptureEnd:
			//log.Printf("Capture end")
		case highlight.EventSource:
			//log.Printf("Highlight range %d-%d", e.StartByte, e.EndByte)
			if t != "" {
				s.WriteString(colorMap[t])
			}
			s.Write(code[e.StartByte:e.EndByte])
			if t != "" {
				s.WriteString(colorMap["black"])
			}
			t = ""
		}
	}
	return s.String()
}
