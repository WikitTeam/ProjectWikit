package db

import (
	"reflect"
	"testing"
)

func TestParseTagGroups(t *testing.T) {
	got := ParseTagGroups("SCP 故事 其他\n\n  原创  搬运 \nuv至上 dv至上\n")
	want := [][]string{{"scp", "故事", "其他"}, {"原创", "搬运"}, {"uv至上", "dv至上"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ParseTagGroups() = %v, want %v", got, want)
	}
}

func TestMissingTagGroups(t *testing.T) {
	groups := [][]string{{"scp", "故事"}, {"原创", "搬运"}, {"uv至上", "dv至上"}}
	got := MissingTagGroups(groups, []string{"SCP", "原创", "精品"})
	want := [][]string{{"uv至上", "dv至上"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("MissingTagGroups() = %v, want %v", got, want)
	}
	if got := MissingTagGroups(groups, []string{"故事", "搬运", "dv至上"}); got != nil {
		t.Errorf("MissingTagGroups(all present) = %v, want nil", got)
	}
}
