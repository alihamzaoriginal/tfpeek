package summary

import (
	"strings"
	"testing"
)

func TestParse_empty(t *testing.T) {
	const j = `{"format_version":"1.2","resource_changes":[]}`
	b, err := Parse(strings.NewReader(j))
	if err != nil {
		t.Fatal(err)
	}
	out := Format(b, "apply", "terraform")
	if !strings.Contains(out, "dry-run") || !strings.Contains(out, "terraform apply was not executed") {
		t.Fatalf("missing dry-run line: %q", out)
	}
	if !strings.Contains(out, "tfpeek summary") {
		t.Fatalf("unexpected output: %q", out)
	}
	if !strings.Contains(out, "0 tracked changes") {
		t.Fatalf("unexpected output: %q", out)
	}
}

func TestParse_buckets(t *testing.T) {
	const j = `{
  "resource_changes": [
    {"address":"test_instance.a","change":{"actions":["create"]}},
    {"address":"test_instance.b","change":{"actions":["update"]}},
    {"address":"test_instance.c","change":{"actions":["delete"]}},
    {"address":"test_instance.d","change":{"actions":["delete","create"]}},
    {"address":"test_instance.e","change":{"actions":["no-op"]}}
  ]
}`
	b, err := Parse(strings.NewReader(j))
	if err != nil {
		t.Fatal(err)
	}
	if len(b.Create) != 1 || len(b.Update) != 1 || len(b.Destroy) != 1 || len(b.Replace) != 1 || len(b.NoOp) != 1 {
		t.Fatalf("unexpected buckets: %+v", b)
	}
	out := Format(b, "destroy", "tofu")
	for _, want := range []string{
		"dry-run",
		"tofu destroy was not executed",
		"CREATE (1)",
		"UPDATE (1)",
		"REPLACE (1)",
		"DESTROY (1)",
		"  + test_instance.a — will be created",
		"  ~ test_instance.b — will be updated",
		"  ± test_instance.d — will be replaced",
		"  - test_instance.c — will be destroyed",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
	if strings.Contains(out, "test_instance.e") {
		t.Fatal("no-op should not appear in formatted output")
	}
}

func TestClassify_replace(t *testing.T) {
	b, err := Parse(strings.NewReader(`{"resource_changes":[{"address":"x","change":{"actions":["delete","create"]}}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(b.Replace) != 1 {
		t.Fatalf("got %+v", b)
	}
}
