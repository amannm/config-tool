package cmd

import (
	"bytes"
	"io/ioutil"
	"testing"
)

func Test_ExecuteVersionCommand(t *testing.T) {
	cmd := NewRootCommand()
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetArgs([]string{"version"})
	err := cmd.Execute()
	if err != nil {
		t.Fatal(err)
	}
	out, err := ioutil.ReadAll(b)
	if err != nil {
		t.Fatal(err)
	}
	if string(out) == "0.0.0-dev" {
		t.Fatalf("expected default version string output")
	}
}
