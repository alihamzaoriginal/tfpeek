package summary

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
)

// PlanJSON matches the subset of `tofu show -json` output we need.
type PlanJSON struct {
	ResourceChanges []ResourceChange `json:"resource_changes"`
}

type ResourceChange struct {
	Address string `json:"address"`
	Change  struct {
		Actions []string `json:"actions"`
	} `json:"change"`
}

type Bucket struct {
	Create   []string
	Update   []string
	Replace  []string
	Destroy  []string
	NoOp     []string
}

func Parse(r io.Reader) (*Bucket, error) {
	var p PlanJSON
	dec := json.NewDecoder(r)
	if err := dec.Decode(&p); err != nil {
		return nil, err
	}
	b := &Bucket{}
	for _, rc := range p.ResourceChanges {
		addr := strings.TrimSpace(rc.Address)
		if addr == "" {
			continue
		}
		switch classify(rc.Change.Actions) {
		case "create":
			b.Create = append(b.Create, addr)
		case "update":
			b.Update = append(b.Update, addr)
		case "replace":
			b.Replace = append(b.Replace, addr)
		case "destroy":
			b.Destroy = append(b.Destroy, addr)
		case "noop":
			b.NoOp = append(b.NoOp, addr)
		default:
			b.Update = append(b.Update, addr)
		}
	}
	sort.Strings(b.Create)
	sort.Strings(b.Update)
	sort.Strings(b.Replace)
	sort.Strings(b.Destroy)
	sort.Strings(b.NoOp)
	return b, nil
}

func classify(actions []string) string {
	if len(actions) == 0 {
		return "noop"
	}
	// Normalized order from OpenTofu/Terraform JSON.
	hasDelete := false
	hasCreate := false
	hasUpdate := false
	for _, a := range actions {
		switch a {
		case "delete":
			hasDelete = true
		case "create":
			hasCreate = true
		case "update":
			hasUpdate = true
		case "no-op":
			return "noop"
		}
	}
	if hasDelete && hasCreate {
		return "replace"
	}
	if hasDelete {
		return "destroy"
	}
	if hasCreate {
		return "create"
	}
	if hasUpdate {
		return "update"
	}
	return "noop"
}

// Format renders a human-readable summary. dryRunMode is "apply" or "destroy"
// and adds a line clarifying that apply/destroy was not executed.
// cliExe is the executable basename used for messaging ("terraform" or "tofu").
func Format(b *Bucket, dryRunMode, cliExe string) string {
	var buf bytes.Buffer
	n := len(b.Create) + len(b.Update) + len(b.Replace) + len(b.Destroy)
	plural := "changes"
	if n == 1 {
		plural = "change"
	}
	fmt.Fprintf(&buf, "tfpeek summary — %d tracked %s\n", n, plural)
	switch dryRunMode {
	case "apply":
		fmt.Fprintf(&buf, "dry-run: %s apply was not executed — preview below\n", cliExe)
	case "destroy":
		fmt.Fprintf(&buf, "dry-run: %s destroy was not executed — preview below\n", cliExe)
	}
	writeSection(&buf, "create", b.Create)
	writeSection(&buf, "update", b.Update)
	writeSection(&buf, "replace", b.Replace)
	writeSection(&buf, "destroy", b.Destroy)
	return strings.TrimRight(buf.String(), "\n")
}

var outcomePhrase = map[string]string{
	"create":  "will be created",
	"update":  "will be updated",
	"replace": "will be replaced",
	"destroy": "will be destroyed",
}

var sectionMarker = map[string]string{
	"create":  "+",
	"update":  "~",
	"replace": "±",
	"destroy": "-",
}

func writeSection(w io.Writer, title string, addrs []string) {
	if len(addrs) == 0 {
		return
	}
	mark := sectionMarker[title]
	if mark == "" {
		mark = "•"
	}
	phrase := outcomePhrase[title]
	if phrase == "" {
		phrase = "will change"
	}
	fmt.Fprintf(w, "\n%s (%d):\n", strings.ToUpper(title), len(addrs))
	for _, a := range addrs {
		fmt.Fprintf(w, "  %s %s — %s\n", mark, a, phrase)
	}
}
