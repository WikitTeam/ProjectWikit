package renderer

import "context"

type Mode string

const (
	ModeArticle           Mode = "article"
	ModeMessage           Mode = "message"
	ModeInline            Mode = "inline"
	ModeSystem            Mode = "system"
	ModeSystemWithModules Mode = "system-with-modules"
)

func (m Mode) Valid() bool {
	switch m {
	case ModeArticle, ModeMessage, ModeInline, ModeSystem, ModeSystemWithModules:
		return true
	}
	return false
}

type PageInfo struct {
	Page        string
	Category    string
	Site        string
	Title       string
	Domain      string
	MediaDomain string
	Rating      float64
	Tags        []string
	Language    string
}

type IncludeRef struct {
	FullName  string
	Variables map[string]string
}

type FetchedPage struct {
	FullName string
	Content  *string
}

type PartialPageInfo struct {
	FullName string
	Title    *string
	Exists   bool
}

type CodeBlock struct {
	Language string
	Source   string
}

type Result struct {
	Body          string
	IncludedPages []string
	LinkedPages   []string
	Code          []CodeBlock
	HTML          []string
}

type Parts struct {
	Code []CodeBlock
	HTML []string
}

type ExpressionKind uint8

const (
	ExprNone ExpressionKind = iota
	ExprString
	ExprBool
	ExprFloat
	ExprInt
)

type ExpressionResult struct {
	Kind  ExpressionKind
	Str   string
	Bool  bool
	Float float64
	Int   int64
}

func StringExpr(v string) ExpressionResult { return ExpressionResult{Kind: ExprString, Str: v} }
func BoolExpr(v bool) ExpressionResult     { return ExpressionResult{Kind: ExprBool, Bool: v} }
func FloatExpr(v float64) ExpressionResult { return ExpressionResult{Kind: ExprFloat, Float: v} }
func IntExpr(v int64) ExpressionResult     { return ExpressionResult{Kind: ExprInt, Int: v} }

type PageCallbacks interface {
	ModuleHasBody(name string) (bool, error)
	RenderModule(name string, params map[string]string, body string) (string, error)
	RenderUser(user string, avatar bool) (string, error)
	GetI18nMessage(id string) (string, error)
	GetHTMLInjectedCode(id string) (string, error)
	GetPageInfo(refs []string) ([]PartialPageInfo, error)
	EvaluateExpression(expr string) (ExpressionResult, error)
	NormalizePageName(fullName string) (string, error)
}

type Includer interface {
	IncludePages(refs []IncludeRef) ([]FetchedPage, error)
	NoSuchInclude(fullName string) (string, error)
}

type Callbacks interface {
	PageCallbacks
	Includer
	NextIncludeLevel() (bool, error)
}

type Renderer interface {
	RenderHTML(ctx context.Context, source string, info PageInfo, cb Callbacks, mode Mode) (Result, error)
	RenderText(ctx context.Context, source string, info PageInfo, cb Callbacks, mode Mode) (Result, error)
	CollectBacklinks(ctx context.Context, source string, info PageInfo, cb Callbacks, mode Mode) (Result, error)
	CollectCodeAndHTML(ctx context.Context, source string, info PageInfo, cb Callbacks, mode Mode) (Parts, error)
}
