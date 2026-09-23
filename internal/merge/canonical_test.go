package merge

import "testing"

func TestCanonicalJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		in      string
		want    string
		wantErr bool
	}{
		{"sorts keys at every level", `{"b":1,"a":{"z":true,"y":null}}`, "{\n  \"a\": {\n    \"y\": null,\n    \"z\": true\n  },\n  \"b\": 1\n}\n", false},
		{"keeps array order and empty arrays", `{"x":["b","a"],"e":[]}`, "{\n  \"e\": [],\n  \"x\": [\n    \"b\",\n    \"a\"\n  ]\n}\n", false},
		{"preserves number text", `{"n":10000000000000000000001,"f":1.50}`, "{\n  \"f\": 1.50,\n  \"n\": 10000000000000000000001\n}\n", false},
		{"is idempotent on its own output", "{\n  \"a\": 1\n}\n", "{\n  \"a\": 1\n}\n", false},
		{"rejects trailing data", `{"a":1} {"b":2}`, "", true},
		{"rejects invalid JSON", `{"a":`, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := CanonicalJSON([]byte(tt.in))
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if string(got) != tt.want {
				t.Errorf("got\n%s\nwant\n%s", got, tt.want)
			}
		})
	}
}
