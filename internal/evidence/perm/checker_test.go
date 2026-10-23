package perm_test

import (
	"testing"

	"github.com/kas-runtime/kas/internal/domain"
	"github.com/kas-runtime/kas/internal/evidence/perm"
	"github.com/kas-runtime/kas/internal/testutil"
)

func TestMain(m *testing.M) { testutil.VerifyTestMain(m) }

// TestTagCheckerDeclarative 声明式映射（O-7 默认模式）。
func TestTagCheckerDeclarative(t *testing.T) {
	c := perm.NewTagChecker(map[string][]string{
		"fp-vip":   {"tier:premium", "kb:internal"},
		"fp-basic": {"kb:public"},
	})
	vip := domain.PrincipalScope{ScopeFingerprint: "fp-vip"}
	basic := domain.PrincipalScope{ScopeFingerprint: "fp-basic"}
	unknown := domain.PrincipalScope{ScopeFingerprint: "fp-other"}

	tests := []struct {
		name  string
		ev    domain.Evidence
		scope domain.PrincipalScope
		want  bool
	}{
		{"public_no_tags", domain.Evidence{ID: "e"}, basic, true},
		{"vip_full_set", domain.Evidence{ID: "e", PermissionTags: []string{"tier:premium", "kb:internal"}}, vip, true},
		{"vip_subset", domain.Evidence{ID: "e", PermissionTags: []string{"kb:public"}}, vip, false}, // kb:public 不在 vip 集内
		{"basic_allowed", domain.Evidence{ID: "e", PermissionTags: []string{"kb:public"}}, basic, true},
		{"basic_denied", domain.Evidence{ID: "e", PermissionTags: []string{"tier:premium"}}, basic, false},
		{"unknown_scope_fail_closed", domain.Evidence{ID: "e", PermissionTags: []string{"kb:public"}}, unknown, false},
		{"multi_tag_partial_deny", domain.Evidence{ID: "e", PermissionTags: []string{"kb:public", "tier:premium"}}, basic, false},
	}
	for _, tc := range tests {
		if got := c.Allowed(tc.ev, tc.scope); got != tc.want {
			t.Errorf("%s: Allowed = %v, want %v", tc.name, got, tc.want)
		}
	}
}

// TestFuncCheckerInjected 宿主注入函数模式（O-7 第二模式）。
func TestFuncCheckerInjected(t *testing.T) {
	var called int
	c := perm.FuncChecker(func(ev domain.Evidence, scope domain.PrincipalScope) bool {
		called++
		return scope.AuthorizationEpoch == "fresh"
	})
	if !c.Allowed(domain.Evidence{ID: "e"}, domain.PrincipalScope{AuthorizationEpoch: "fresh"}) {
		t.Fatal("fresh epoch 应允许")
	}
	if c.Allowed(domain.Evidence{ID: "e"}, domain.PrincipalScope{AuthorizationEpoch: "stale"}) {
		t.Fatal("stale epoch 应拒绝")
	}
	if called != 2 {
		t.Fatalf("注入函数调用计数 = %d", called)
	}
}
