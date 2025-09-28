package main

import (
	"fmt"
	"bufio"
	"bytes"
	"context"
	"flag"
	"io"
	"log"
	"os"
	"os/exec"
	"path"
	"strconv"
	"syscall"
	_ "embed"

	"github.com/creack/pty"
	//"gopkg.in/alessio/shellescape.v1"
	"github.com/amenzhinsky/go-memexec"
)

//go:embed assets/oils-for-unix-static.stripped
var embeddedOils []byte

var (
	fanosShellPath = flag.String("oil_path", "", "Path to Oil shell interpreter")
	fifo           = flag.Bool("fifo", false, "Use named fifo instead of anonymous pipe")
)

type FANOSShell struct {
	cmd    *exec.Cmd
	socket *os.File

	in, out, err *os.File
}

func NewFANOSShell() (*FANOSShell, error) {
	shell := &FANOSShell{}
	if *fanosShellPath == "" {
		// Use the mmap and syscall execution method described in the blog post
		exe, err := memexec.New(embeddedOils)
		//defer exe.Close()

		if err != nil {
			return nil, fmt.Errorf("Embedded oils: %w", err)
		}
		// TODO: Allow selecting ysh or osh!
		shell.cmd = exe.Command("ysh", "--headless")
	} else {
		shell.cmd = exec.Command(*fanosShellPath, "--headless")
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
func (s *FANOSShell) Run(command *Command) error {
	// To be added before invocation!
	// TODO: assert there is 1?
	//defer command.wg.Done()

	command.ctx, runCancel = context.WithCancel(context.Background())
	// TODO: Cancel!
	//defer runCancel()

	// ------------------
	// Setup File Descriptors, read them into `command.stdXXX`
	// ------------------

	// TODO: should be set via an API?
	// TODO: should createCommand be big?

	ptmx, tty, err := pty.Open()
	if err != nil {
		log.Println(err)
		// TODO: update the command.status to "failed" and don't return an error
		// TODO: Should be done with all returns here
		return err
	}
	defer func() {
		ptmx.Close()
		tty.Close()
	}()

	var _stderr, rdPipe *os.File
	// Open a fifo for stderr
	if *fifo {
		dir := os.TempDir()
		pipeName := path.Join(dir, "errpipe")
		syscall.Mkfifo(pipeName, 0600)
		// If you open only the read side, then you need to open with O_NONBLOCK
		// and clear that flag after opening.
		//	pipe, err := os.OpenFile(pipeName, os.O_RDONLY|syscall.O_NONBLOCK, 0600)
		_stderr, err = os.OpenFile(pipeName, os.O_RDWR, 0600)
		// read/write are the same for FIFOs
		rdPipe = _stderr
		//log.Println(int(_stderr.Fd()))
		if err != nil {
			log.Println(err)
			return err
		}
		defer func() {
			_stderr.Close()
			os.Remove(pipeName)
			os.Remove(dir)
		}()
	} else {
		rdPipe, _stderr, err = os.Pipe()
		//log.Println(int(_stderr.Fd()))
		if err != nil {
			log.Println(err)
			return err
		}
		defer func() {
			rdPipe.Close()
			_stderr.Close()
		}()
	}

	// ------------------
	// Send command and FDs via FANOS
	// ------------------
	rights := syscall.UnixRights(int(tty.Fd()), int(tty.Fd()), int(_stderr.Fd()))
	command.StdIO(
		ptmx,
		ptmx,
		rdPipe)
	var buf bytes.Buffer
	buf.WriteString("EVAL ")
	buf.WriteString(command.CommandLine)
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
	//log.Println(msg)
	//log.Println("Command is done")
	//log.Println(command.Id)
	command.wg.Done()

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
