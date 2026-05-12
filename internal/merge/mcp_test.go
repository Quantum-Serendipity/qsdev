package merge

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestMergeMcpJson_AllUnmodified(t *testing.T) {
	base := []byte(`{
  "mcpServers": {
    "github": {
      "command": "gh",
      "args": ["mcp"]
    }
  }
}`)
	theirs := []byte(`{
  "mcpServers": {
    "github": {
      "command": "gh",
      "args": ["mcp"]
    }
  }
}`)
	ours := []byte(`{
  "mcpServers": {
    "github": {
      "command": "gh",
      "args": ["mcp", "--verbose"]
    }
  }
}`)

	got, err := MergeMcpJson(base, theirs, ours)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed mcpJSON
	if err := json.Unmarshal(got, &parsed); err != nil {
		t.Fatalf("result is not valid JSON: %v", err)
	}

	gh, ok := parsed.MCPServers["github"]
	if !ok {
		t.Fatal("expected github server in result")
	}
	if gh.Args[len(gh.Args)-1] != "--verbose" {
		t.Errorf("expected updated args from ours, got %v", gh.Args)
	}
}

func TestMergeMcpJson_UserAddedServer(t *testing.T) {
	base := []byte(`{
  "mcpServers": {
    "github": {
      "command": "gh",
      "args": ["mcp"]
    }
  }
}`)
	theirs := []byte(`{
  "mcpServers": {
    "github": {
      "command": "gh",
      "args": ["mcp"]
    },
    "my-custom": {
      "command": "/usr/local/bin/my-mcp",
      "args": ["serve"]
    }
  }
}`)
	ours := []byte(`{
  "mcpServers": {
    "github": {
      "command": "gh",
      "args": ["mcp"]
    }
  }
}`)

	got, err := MergeMcpJson(base, theirs, ours)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed mcpJSON
	if err := json.Unmarshal(got, &parsed); err != nil {
		t.Fatalf("result is not valid JSON: %v", err)
	}

	if _, ok := parsed.MCPServers["my-custom"]; !ok {
		t.Error("expected user-added 'my-custom' server to survive merge")
	}
	if _, ok := parsed.MCPServers["github"]; !ok {
		t.Error("expected 'github' server to remain")
	}
}

func TestMergeMcpJson_GeneratedServerUpdated(t *testing.T) {
	base := []byte(`{
  "mcpServers": {
    "github": {
      "command": "gh",
      "args": ["mcp"]
    }
  }
}`)
	theirs := []byte(`{
  "mcpServers": {
    "github": {
      "command": "gh",
      "args": ["mcp"]
    }
  }
}`)
	ours := []byte(`{
  "mcpServers": {
    "github": {
      "command": "gh",
      "args": ["mcp", "--profile", "work"]
    }
  }
}`)

	got, err := MergeMcpJson(base, theirs, ours)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed mcpJSON
	if err := json.Unmarshal(got, &parsed); err != nil {
		t.Fatalf("result is not valid JSON: %v", err)
	}

	gh := parsed.MCPServers["github"]
	wantArgs := []string{"mcp", "--profile", "work"}
	if len(gh.Args) != len(wantArgs) {
		t.Fatalf("expected args %v, got %v", wantArgs, gh.Args)
	}
	for i, a := range gh.Args {
		if a != wantArgs[i] {
			t.Errorf("args[%d]: got %q, want %q", i, a, wantArgs[i])
		}
	}
}

func TestMergeMcpJson_GeneratedServerRemoved(t *testing.T) {
	base := []byte(`{
  "mcpServers": {
    "github": {
      "command": "gh",
      "args": ["mcp"]
    },
    "deprecated-tool": {
      "command": "dep-tool",
      "args": ["serve"]
    }
  }
}`)
	theirs := []byte(`{
  "mcpServers": {
    "github": {
      "command": "gh",
      "args": ["mcp"]
    },
    "deprecated-tool": {
      "command": "dep-tool",
      "args": ["serve"]
    }
  }
}`)
	// ours removed "deprecated-tool"
	ours := []byte(`{
  "mcpServers": {
    "github": {
      "command": "gh",
      "args": ["mcp"]
    }
  }
}`)

	got, err := MergeMcpJson(base, theirs, ours)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed mcpJSON
	if err := json.Unmarshal(got, &parsed); err != nil {
		t.Fatalf("result is not valid JSON: %v", err)
	}

	if _, ok := parsed.MCPServers["deprecated-tool"]; ok {
		t.Error("expected deprecated-tool to be removed")
	}
	if _, ok := parsed.MCPServers["github"]; !ok {
		t.Error("expected github to remain")
	}
}

func TestMergeMcpJson_UserDeletedGeneratedServer(t *testing.T) {
	base := []byte(`{
  "mcpServers": {
    "github": {
      "command": "gh",
      "args": ["mcp"]
    },
    "sentry": {
      "command": "sentry-mcp",
      "args": ["serve"]
    }
  }
}`)
	// User deleted "sentry"
	theirs := []byte(`{
  "mcpServers": {
    "github": {
      "command": "gh",
      "args": ["mcp"]
    }
  }
}`)
	ours := []byte(`{
  "mcpServers": {
    "github": {
      "command": "gh",
      "args": ["mcp"]
    },
    "sentry": {
      "command": "sentry-mcp",
      "args": ["serve", "--verbose"]
    }
  }
}`)

	got, err := MergeMcpJson(base, theirs, ours)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed mcpJSON
	if err := json.Unmarshal(got, &parsed); err != nil {
		t.Fatalf("result is not valid JSON: %v", err)
	}

	// User deleted sentry — should stay deleted.
	if _, ok := parsed.MCPServers["sentry"]; ok {
		t.Error("expected sentry to remain deleted (user deleted it)")
	}
}

func TestMergeMcpJson_UserModifiedGeneratedServer(t *testing.T) {
	base := []byte(`{
  "mcpServers": {
    "github": {
      "command": "gh",
      "args": ["mcp"],
      "env": {"GITHUB_TOKEN": "base-token"}
    }
  }
}`)
	// User changed the env var.
	theirs := []byte(`{
  "mcpServers": {
    "github": {
      "command": "gh",
      "args": ["mcp"],
      "env": {"GITHUB_TOKEN": "user-custom-token"}
    }
  }
}`)
	// ours also updates it.
	ours := []byte(`{
  "mcpServers": {
    "github": {
      "command": "gh",
      "args": ["mcp", "--profile", "work"],
      "env": {"GITHUB_TOKEN": "new-generated-token"}
    }
  }
}`)

	got, err := MergeMcpJson(base, theirs, ours)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed mcpJSON
	if err := json.Unmarshal(got, &parsed); err != nil {
		t.Fatalf("result is not valid JSON: %v", err)
	}

	gh := parsed.MCPServers["github"]
	// User modified → keep theirs version.
	if gh.Env["GITHUB_TOKEN"] != "user-custom-token" {
		t.Errorf("expected user's token, got %q", gh.Env["GITHUB_TOKEN"])
	}
	// Theirs version of args (not ours) since user modified the entry.
	if len(gh.Args) != 1 || gh.Args[0] != "mcp" {
		t.Errorf("expected theirs args [mcp], got %v", gh.Args)
	}
}

func TestMergeMcpJson_NewGeneratedServer(t *testing.T) {
	base := []byte(`{
  "mcpServers": {
    "github": {
      "command": "gh",
      "args": ["mcp"]
    }
  }
}`)
	theirs := []byte(`{
  "mcpServers": {
    "github": {
      "command": "gh",
      "args": ["mcp"]
    }
  }
}`)
	// ours adds a new server.
	ours := []byte(`{
  "mcpServers": {
    "github": {
      "command": "gh",
      "args": ["mcp"]
    },
    "sentry": {
      "command": "sentry-mcp",
      "args": ["serve"]
    }
  }
}`)

	got, err := MergeMcpJson(base, theirs, ours)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed mcpJSON
	if err := json.Unmarshal(got, &parsed); err != nil {
		t.Fatalf("result is not valid JSON: %v", err)
	}

	if _, ok := parsed.MCPServers["sentry"]; !ok {
		t.Error("expected newly generated 'sentry' server")
	}
	if _, ok := parsed.MCPServers["github"]; !ok {
		t.Error("expected 'github' to remain")
	}
}

func TestMergeMcpJson_EmptyBase(t *testing.T) {
	// nil base — all theirs servers treated as user-added.
	theirs := []byte(`{
  "mcpServers": {
    "github": {
      "command": "gh",
      "args": ["mcp"]
    },
    "my-custom": {
      "command": "/usr/local/bin/my-mcp",
      "args": ["serve"]
    }
  }
}`)
	ours := []byte(`{
  "mcpServers": {
    "github": {
      "command": "gh",
      "args": ["mcp", "--verbose"]
    }
  }
}`)

	got, err := MergeMcpJson(nil, theirs, ours)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed mcpJSON
	if err := json.Unmarshal(got, &parsed); err != nil {
		t.Fatalf("result is not valid JSON: %v", err)
	}

	// github: not in base → newly generated from ours → use ours version.
	gh := parsed.MCPServers["github"]
	if len(gh.Args) != 2 || gh.Args[1] != "--verbose" {
		t.Errorf("expected ours github args, got %v", gh.Args)
	}

	// my-custom: not in ours, not in base → user-added → preserved.
	if _, ok := parsed.MCPServers["my-custom"]; !ok {
		t.Error("expected user-added 'my-custom' to be preserved")
	}
}

func TestMergeMcpJson_InvalidJSON(t *testing.T) {
	base := []byte(`{"mcpServers":{}}`)
	theirs := []byte(`{not valid json}`)
	ours := []byte(`{"mcpServers":{}}`)

	_, err := MergeMcpJson(base, theirs, ours)
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
	if !strings.Contains(err.Error(), "theirs") {
		t.Errorf("expected error to mention 'theirs', got: %v", err)
	}
}

func TestMergeMcpJson_EmptyServers(t *testing.T) {
	base := []byte(`{"mcpServers": {}}`)
	theirs := []byte(`{"mcpServers": {}}`)
	ours := []byte(`{"mcpServers": {}}`)

	got, err := MergeMcpJson(base, theirs, ours)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed mcpJSON
	if err := json.Unmarshal(got, &parsed); err != nil {
		t.Fatalf("result is not valid JSON: %v", err)
	}

	if len(parsed.MCPServers) != 0 {
		t.Errorf("expected empty mcpServers, got %v", parsed.MCPServers)
	}
}

func TestMergeMcpJson_OutputIsValidJSON(t *testing.T) {
	base := []byte(`{
  "mcpServers": {
    "github": {
      "command": "gh",
      "args": ["mcp"],
      "env": {"TOKEN": "abc"}
    }
  }
}`)
	theirs := []byte(`{
  "mcpServers": {
    "github": {
      "command": "gh",
      "args": ["mcp"],
      "env": {"TOKEN": "abc"}
    },
    "custom": {
      "command": "my-tool",
      "args": []
    }
  }
}`)
	ours := []byte(`{
  "mcpServers": {
    "github": {
      "command": "gh",
      "args": ["mcp", "--v2"],
      "env": {"TOKEN": "def"}
    },
    "new-server": {
      "command": "new-tool",
      "args": ["run"]
    }
  }
}`)

	got, err := MergeMcpJson(base, theirs, ours)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var raw json.RawMessage
	if err := json.Unmarshal(got, &raw); err != nil {
		t.Fatalf("result is not valid JSON: %v\nresult:\n%s", err, got)
	}

	// Verify trailing newline.
	if got[len(got)-1] != '\n' {
		t.Error("expected trailing newline")
	}
}
