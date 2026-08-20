package renderer

import (
	"context"
	"errors"
	"testing"
)

func TestModeValid(t *testing.T) {
	fromRust := []Mode{"article", "message", "inline", "system", "system-with-modules"}
	for _, m := range fromRust {
		if !m.Valid() {
			t.Errorf("Valid(%q) = false, want true", m)
		}
	}
	for _, m := range []Mode{"", "page", "Article", "system_with_modules"} {
		if m.Valid() {
			t.Errorf("Valid(%q) = true, want false", m)
		}
	}
}

func TestFakeRecordsCalls(t *testing.T) {
	f := &Fake{Result: Result{Body: "<p>hi</p>"}}
	ctx := context.Background()
	info := PageInfo{Page: "173", Category: "scp"}

	if _, err := f.RenderHTML(ctx, "src", info, NopCallbacks{}, ModeArticle); err != nil {
		t.Fatal(err)
	}
	if _, err := f.RenderText(ctx, "src", info, NopCallbacks{}, ModeSystem); err != nil {
		t.Fatal(err)
	}

	if len(f.Calls) != 2 {
		t.Fatalf("len(Calls) = %d, want 2", len(f.Calls))
	}
	if f.Calls[0].Op != "RenderHTML" || f.Calls[0].Mode != ModeArticle {
		t.Errorf("Calls[0] = {Op:%s Mode:%s}, want {Op:RenderHTML Mode:article}", f.Calls[0].Op, f.Calls[0].Mode)
	}
	if f.Calls[1].Info.Category != "scp" {
		t.Errorf("Calls[1].Info.Category = %q, want %q", f.Calls[1].Info.Category, "scp")
	}
}

func TestFakeContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	f := &Fake{Result: Result{Body: "x"}}
	if _, err := f.RenderHTML(ctx, "src", PageInfo{}, NopCallbacks{}, ModeArticle); !errors.Is(err, context.Canceled) {
		t.Errorf("RenderHTML() err = %v, want context.Canceled", err)
	}
}

func TestFakeRejectsUnknownMode(t *testing.T) {
	f := &Fake{}
	if _, err := f.RenderHTML(context.Background(), "src", PageInfo{}, NopCallbacks{}, "nonsense"); err == nil {
		t.Error("RenderHTML(mode=\"nonsense\") err = nil, want non-nil")
	}
}

func TestNopCallbacksNormalizePageName(t *testing.T) {
	got, err := NopCallbacks{}.NormalizePageName("SCP:173")
	if err != nil {
		t.Fatal(err)
	}
	if got != "SCP:173" {
		t.Errorf("NormalizePageName(%q) = %q, want %q", "SCP:173", got, "SCP:173")
	}
}

func TestNopCallbacksNextIncludeLevel(t *testing.T) {
	ok, err := NopCallbacks{}.NextIncludeLevel()
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Error("NextIncludeLevel() = false, want true")
	}
}

func TestCallbacksSatisfiesSubInterfaces(t *testing.T) {
	var cb Callbacks = NopCallbacks{}

	if _, ok := cb.(PageCallbacks); !ok {
		t.Error("Callbacks does not satisfy PageCallbacks")
	}
	if _, ok := cb.(Includer); !ok {
		t.Error("Callbacks does not satisfy Includer")
	}
}

func TestExpressionResultZeroValue(t *testing.T) {
	if (ExpressionResult{}).Kind != ExprNone {
		t.Errorf("ExpressionResult{}.Kind = %v, want %v", (ExpressionResult{}).Kind, ExprNone)
	}
}

func TestExpressionConstructors(t *testing.T) {
	tests := []struct {
		got  ExpressionResult
		kind ExpressionKind
	}{
		{StringExpr("x"), ExprString},
		{BoolExpr(true), ExprBool},
		{FloatExpr(1.5), ExprFloat},
		{IntExpr(-3), ExprInt},
	}
	for _, tt := range tests {
		if tt.got.Kind != tt.kind {
			t.Errorf("Kind = %v, want %v", tt.got.Kind, tt.kind)
		}
	}

	if got := StringExpr("x").Str; got != "x" {
		t.Errorf("StringExpr(\"x\").Str = %q, want %q", got, "x")
	}
	if got := BoolExpr(true).Bool; got != true {
		t.Errorf("BoolExpr(true).Bool = %v, want true", got)
	}
	if got := FloatExpr(1.5).Float; got != 1.5 {
		t.Errorf("FloatExpr(1.5).Float = %v, want 1.5", got)
	}
	if got := IntExpr(-3).Int; got != -3 {
		t.Errorf("IntExpr(-3).Int = %d, want -3", got)
	}
}
