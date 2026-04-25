package helpers

import "testing"

func TestResourceName(t *testing.T) {
	tests := []struct {
		parts []string
		want  string
	}{
		{[]string{"github_team", "My Org", "dev-team"}, "github_team.my_org.dev-team"},
		{[]string{"github_repository", "oddbit.com"}, "github_repository.oddbit_com"},
		{[]string{"github_user_ssh_key", "larsks", "0"}, "github_user_ssh_key.larsks.0"},
	}
	for _, tt := range tests {
		got := ResourceName(tt.parts...)
		if got != tt.want {
			t.Errorf("ResourceName(%v) = %q, want %q", tt.parts, got, tt.want)
		}
	}
}
