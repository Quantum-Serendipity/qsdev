package cmdutil

import (
	"slices"
	"sync/atomic"
	"testing"

	"github.com/spf13/cobra"
)

func TestCompleteFrom(t *testing.T) {
	t.Parallel()
	var calls atomic.Int32
	list := func() []string {
		calls.Add(1)
		return []string{"postgres", "redis", "rabbitmq"}
	}
	complete := CompleteFrom(list)
	if n := calls.Load(); n != 0 {
		t.Fatalf("list resolved %d times while building the completion func; want it deferred", n)
	}

	tests := []struct {
		name       string
		args       []string
		toComplete string
		want       []cobra.Completion
	}{
		{"empty prefix", nil, "", []cobra.Completion{"postgres", "redis", "rabbitmq"}},
		{"prefix filters", nil, "r", []cobra.Completion{"redis", "rabbitmq"}},
		{"no match", nil, "x", nil},
		{"only first argument", []string{"redis"}, "", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, directive := complete(&cobra.Command{}, tt.args, tt.toComplete)
			if !slices.Equal(got, tt.want) {
				t.Errorf("completions = %v, want %v", got, tt.want)
			}
			if directive != cobra.ShellCompDirectiveNoFileComp {
				t.Errorf("directive = %v, want ShellCompDirectiveNoFileComp", directive)
			}
		})
	}
}

func TestCompleteFrom_NilList(t *testing.T) {
	t.Parallel()
	if CompleteFrom(nil) != nil {
		t.Error("CompleteFrom(nil) should return nil to keep cobra's default completion")
	}
}
