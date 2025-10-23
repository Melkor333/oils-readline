package main

import (
	"bufio"
	"bytes"
	"context"
	_ "embed"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path"
	"strconv"
	"syscall"
)

//go:generate bash ./static-oils.sh
//go:embed assets/oils-for-unix-static.stripped
var embeddedOils []byte

var (
	fanosShellPath = flag.String("oil_path", "", "Path to Oil shell interpreter")
	fifo           = flag.Bool("fifo", false, "Use named fifo instead of anonymous pipe")
)

type FANOSShell struct {
	cmd    *exec.Cmd
	cancel context.CancelFunc
	socket *os.File

	in, out, err *os.File
}

func (s *FANOSShell) Cancel() {
	s.cancel()
	s.in.Close()
	s.out.Close()
	s.err.Close()
	s.cmd.Wait()
}

func NewFANOSShell() (*FANOSShell, error) {
	shell := &FANOSShell{}
	var ctx context.Context
	ctx, shell.cancel = context.WithCancel(context.Background())
	if *fanosShellPath == "" {
		// Use the mmap and syscall execution method described in the blog post
		tempDir := os.TempDir()
		filePath := path.Join(tempDir, "ysh")
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			// Write the embedded binary to a temporary file
			if err := os.WriteFile(filePath, embeddedOils, 0700); err != nil {
				return nil, fmt.Errorf("failed to write embedded binary: %w", err)
			}
			defer os.Remove(filePath)
			// Set permissions to make it executable
			syscall.Chmod(filePath, 0700)
		}
		shell.cmd = exec.CommandContext(ctx, filePath, "--headless")
	} else {
		shell.cmd = exec.CommandContext(ctx, *fanosShellPath, "--headless")
	}

	fds, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
	if err != nil {
		return nil, fmt.Errorf("can't create socketpair: %w", err)
	}
	shell.socket = os.NewFile(uintptr(fds[0]), "fanos_client")
	server := os.NewFile(uintptr(fds[1]), "fanos_server")
	shell.cmd.Stdin = server
	shell.cmd.Stdout = server

	shell.cmd.Stderr = io.Discard
	//shell.cmd.Stderr = os.Stderr

	o := shell.cmd.Start()
	// TODO: Graceful exit
	go func() {
		shell.cmd.Wait()
		os.Exit(0)
	}()
	return shell, o
}

//func (s *FANOSShell) StdIO(in, out, err *os.File) error {
//	// Save these for the next Run
//	s.in, s.out, s.err = in, out, err
//	if s.in == nil {
//		s.in, _ = os.Open(os.DevNull)
//	}
//	if s.out == nil {
//		s.out, _ = os.Open(os.DevNull)
//	}
//	if s.err == nil {
//		s.err, _ = os.Open(os.DevNull)
//	}
//
//	return nil
//}

// Run calls the FANOS EVAL method
func (s *FANOSShell) Run(command string, stdin, stdout, stderr *os.File) error {
	// ------------------
	// Setup File Descriptors, read them into `command.stdXXX`
	// ------------------

	// TODO: should be set via an API?
	// TODO: should createCommand be big?
	//var ptmx, tty *os.File
	var err error
	defer func() {
		stdin.Close()
		stdout.Close()
		stderr.Close()
	}()

	// ------------------
	// Send command and FDs via FANOS
	// ------------------
	rights := syscall.UnixRights(int(stdin.Fd()), int(stdout.Fd()), int(stderr.Fd()))
	var buf bytes.Buffer
	buf.WriteString("EVAL ")
	buf.WriteString(command)
	// Send command per Netstring
	_, err = s.socket.Write([]byte(strconv.Itoa(buf.Len()) + ":"))
	if err != nil {
		log.Println(err)
		return err
	}
	err = syscall.Sendmsg(int(s.socket.Fd()), buf.Bytes(), rights, nil, 0)
	if err != nil {
		log.Println(err)
		return err
	}
	_, err = s.socket.Write([]byte(","))
	if err != nil {
		return err
	}

	// TODO: Actually read netstring instead of reading until ','
	// Wait for FANOS Answer
	//log.Println("Running command")
	sockReader := bufio.NewReader(s.socket)
	_, err = sockReader.ReadString(',')
	if err != nil {
		return err
	}
	return nil
}

func (s *FANOSShell) Dir() string {
	return ""
}

func (s *FANOSShell) Complete(ctx context.Context, r CompletionReq) (*CompletionResult, error) {
	comps := CompletionResult{}
	comps.To = len(r.Text)
	return &comps, nil
}
