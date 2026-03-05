package fanos

import (
	_ "embed"
	"io"
	"os"
	"strings"
	"testing"
)

type TestArgs struct {
	command string
	stdin   string
}

var (
	command_tests = []struct {
		name    string
		args    TestArgs
		stdout  string
		stderr  string
		wantErr bool
	}{
		{
			"Run write hello",
			TestArgs{"write hello", ""},
			"hello\n", "",
			false,
		},
		{
			"Stderr",
			TestArgs{"write hello >&2", ""},
			"", "hello\n",
			false,
		},
		{
			"multiple commands",
			TestArgs{`write hello >&2
				write hello
				write hello >&2`, ""},
			"hello\n", "hello\nhello\n",
			false,
		},
		{
			"weird characters",
			TestArgs{string('\t'), ""},
			"", "",
			false,
		},
	}
)

func TestShell_Run(t *testing.T) {
	// Currently fails
	//{
	//	"Error",
	//	args{"return 2", ""},
	//	"", "",
	//	true,
	//},
	for _, tt := range command_tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := New()
			if err != nil {
				t.Fatal(err)
			}
			stdinReader, stdinWriter, err := os.Pipe()
			if err != nil {
				t.Fatal(err)
			}
			go func() {
				stdinWriter.WriteString(tt.args.stdin)
				stdinWriter.Close()
			}()

			stdoutReader, stdoutWriter, err := os.Pipe()
			if err != nil {
				t.Fatal(err)
			}
			stderrReader, stderrWriter, err := os.Pipe()
			if err != nil {
				t.Fatal(err)
			}

			if err := s.Run(tt.args.command, stdinReader, stdoutWriter, stderrWriter); (err != nil) != tt.wantErr {
				t.Errorf("Shell.Run() error = %v, wantErr %v", err, tt.wantErr)
			}
			stdinReader.Close()
			stderrWriter.Close()
			stdoutWriter.Close()

			var stdout, stderr strings.Builder
			io.Copy(&stdout, stdoutReader)
			if stdout.String() != tt.stdout {
				t.Errorf("stdout doesn't match! got %v, wanted %v", stdout.String(), tt.stdout)
			}
			io.Copy(&stderr, stderrReader)
			if stderr.String() != tt.stderr {
				t.Errorf("stderr doesn't match! got %v, wanted %v", stdout.String(), tt.stdout)
			}

		})
	}
}
