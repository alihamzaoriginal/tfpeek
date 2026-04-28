package main

import (
	"bytes"
	"io"
	"testing"
)

func TestPlanStdoutWriter(t *testing.T) {
	var buf bytes.Buffer
	t.Setenv("TFPEEK_VERBOSE_PLAN", "")
	if planStdoutWriter(&buf) != io.Discard {
		t.Fatal("expected io.Discard when TFPEEK_VERBOSE_PLAN unset")
	}
	t.Setenv("TFPEEK_VERBOSE_PLAN", "1")
	if planStdoutWriter(&buf) != &buf {
		t.Fatal("expected stdout writer when TFPEEK_VERBOSE_PLAN=1")
	}
}

func TestResolveCLI_invalidEnv(t *testing.T) {
	t.Setenv("TFPEEK_CLI", "ansible")
	_, err := resolveCLI()
	if err == nil {
		t.Fatal("expected error for invalid TFPEEK_CLI")
	}
}
