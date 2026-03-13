package fanos

import (
	"context"
	_ "embed"
	"os"
	"sync"
	"testing"

	"github.com/creack/pty"
)

//func TestShell_Command(t *testing.T) {
//	type fields struct {
//		cmd    *exec.Cmd
//		cancel context.CancelFunc
//		socket *os.File
//		in     *os.File
//		out    *os.File
//		err    *os.File
//	}
//	type args struct {
//		commandLine string
//		size        *pty.Winsize
//	}
//	tests := []struct {
//		name    string
//		fields  fields
//		args    args
//		want    *Command
//		wantErr bool
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			shell := &Shell{
//				cmd:    tt.fields.cmd,
//				cancel: tt.fields.cancel,
//				socket: tt.fields.socket,
//				in:     tt.fields.in,
//				out:    tt.fields.out,
//				err:    tt.fields.err,
//			}
//			got, err := shell.Command(tt.args.commandLine, tt.args.size)
//			if (err != nil) != tt.wantErr {
//				t.Fatalf("Shell.Command() error = %v, wantErr %v", err, tt.wantErr)
//			}
//			if tt.wantErr {
//				return
//			}
//			if !reflect.DeepEqual(got, tt.want) {
//				t.Errorf("Shell.Command() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}

func TestCommand_Wait(t *testing.T) {
	type fields struct {
		shell       *Shell
		CommandLine string
		err         error
		ctx         context.Context
		Cancel      context.CancelFunc
		Stdin       *os.File
		Stdout      *os.File
		Stderr      *os.File
		tty         *os.File
		stderrIn    *os.File
		wg          *sync.WaitGroup
		lock        *sync.Mutex
	}
	tests := []struct {
		name   string
		fields fields
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Command{
				shell:       tt.fields.shell,
				commandline: tt.fields.CommandLine,
				err:         tt.fields.err,
				ctx:         tt.fields.ctx,
				Cancel:      tt.fields.Cancel,
				stdin:       tt.fields.Stdin,
				stdout:      tt.fields.Stdout,
				stderr:      tt.fields.Stderr,
				tty:         tt.fields.tty,
				stderrIn:    tt.fields.stderrIn,
				wg:          tt.fields.wg,
				lock:        tt.fields.lock,
			}
			c.Wait()
		})
	}
}

func TestCommand_Run(t *testing.T) {
	for _, tt := range command_tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := New()
			if err != nil {
				t.Fatal(err)
			}
			command, err := s.Command(tt.args.command, &pty.Winsize{1, 100, 100, 100})
			// We don't care about the output. we could make sure it's the same command, though
			command.Run()
			command.Wait()

			if command.Stdout() != tt.stdout {
				t.Errorf("stdout doesn't match! got '%q', wanted '%q'", command.Stdout(), tt.stdout)
			}
			if command.Stdout() != tt.stdout {
				t.Errorf("stdout doesn't match on second read: got '%q', wanted '%q'", command.Stdout(), tt.stdout)
			}
			if command.Stderr() != tt.stderr {
				t.Errorf(`stderr doesn't match! got: '%q' wanted '%q'`, command.Stderr(), tt.stderr)
			}
			if command.Stderr() != tt.stderr {
				t.Errorf(`stderr doesn't match on second read: '%q' wanted '%q'`, command.Stderr(), tt.stderr)
			}
		})
	}
}

func TestCommand_Error(t *testing.T) {
	type fields struct {
		shell       *Shell
		CommandLine string
		err         error
		ctx         context.Context
		Cancel      context.CancelFunc
		Stdin       *os.File
		Stdout      *os.File
		Stderr      *os.File
		tty         *os.File
		stderrIn    *os.File
		wg          *sync.WaitGroup
		lock        *sync.Mutex
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Command{
				shell:       tt.fields.shell,
				commandline: tt.fields.CommandLine,
				err:         tt.fields.err,
				ctx:         tt.fields.ctx,
				Cancel:      tt.fields.Cancel,
				stdin:       tt.fields.Stdin,
				stdout:      tt.fields.Stdout,
				stderr:      tt.fields.Stderr,
				tty:         tt.fields.tty,
				stderrIn:    tt.fields.stderrIn,
				wg:          tt.fields.wg,
				lock:        tt.fields.lock,
			}
			if got := c.Error(); got != tt.want {
				t.Errorf("Command.Error() = %v, want %v", got, tt.want)
			}
		})
	}
}
