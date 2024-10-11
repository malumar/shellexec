package shellexec

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os/exec"
)

// return true if you want to continue streaming
type OnOutput func(output []byte) bool

func NewDefault() *ShellCmd {
	return New(nil)
}

// New using tracer you can use this example
// ...
// import "github.com/malumar/tracer"
// ...
//
//	cmd := tracer.New(tracer.NewSimple(tracer.All, NewSimple(Debug|Trace, func(bytes []byte) {
//			sb.Write(bytes)
//	}))
//
// ...
func New(tracer io.Writer) *ShellCmd {
	return &ShellCmd{tracer: tracer}
}

type ShellCmd struct {
	tracer      io.Writer
	err         error
	haveResult  bool
	cmd         *exec.Cmd
	output      []byte
	env         []string
	errorOutput []byte
	stderr      []byte
	stdIn       io.Writer
	stdOut      io.ReadCloser
	stdErr      io.ReadCloser

	outputBuf bytes.Buffer
	stderrBuf bytes.Buffer
}

func (t *ShellCmd) SetEnv(env Environment) *ShellCmd {
	t.env = EnvironmentToSliceOfStr(env)
	return t
}

func (t *ShellCmd) Run(command string, args ...interface{}) *ShellCmd {
	cmds := ""
	argString := make([]string, 0)
	if args != nil && len(args) > 0 {
		for _, a := range args {
			if cmds != "" {
				cmds += " "
			}
			as := fmt.Sprintf("%v", a)
			argString = append(argString, as)
			cmds += as
		}
	}
	t.Tracef("exec: %s %v\n", command, cmds)

	t.cmd = exec.Command(command, argString...)

	t.stdOut, t.err = t.cmd.StdoutPipe()
	if t.err == nil {
		t.stdErr, t.err = t.cmd.StderrPipe()
	}

	return t
}

func (t *ShellCmd) WriteToPipeIn(b []byte) *ShellCmd {
	if t.err == nil {
		var err error
		if t.stdIn == nil {
			t.stdIn, err = t.cmd.StdinPipe()
			if err != nil {
				t.err = err
				t.stdIn = nil
			}

		}
		if t.err == nil {
			if _, err = t.stdIn.Write(b); err != nil {
				t.err = err
			}
		}

	}

	return t
}

func (t *ShellCmd) Error() error {
	return t.err
}
func (t *ShellCmd) RunNow(command string, args ...interface{}) (err error) {
	t.Run(command, args...)
	return t.Go()
}
func (t *ShellCmd) RunNowAndCleanup(command string, args ...interface{}) (err error) {
	t.Run(command, args...)
	e := t.Go()
	t.Cleanup()
	return e
}
func (t *ShellCmd) Go() error {
	return t.Start().OnLine(nil).Wait()
}
func (t *ShellCmd) GoAndCleanup() error {
	t.Start().OnLine(nil).Wait()
	return t.Cleanup()
}
func (t *ShellCmd) Start() *ShellCmd {
	if t.err == nil {
		if t.env != nil {
			t.cmd.Env = t.env

		}
		t.err = t.cmd.Start()

	}

	return t
}
func (t *ShellCmd) OnLine(cb OnOutput) *ShellCmd {
	if t.err != nil {
		return t
	}
	t.errorOutput, _ = io.ReadAll(t.stdErr)
	if cb == nil {
		t.output, t.err = io.ReadAll(t.stdOut)
		t.haveResult = true
		return t
	}
	scanner := bufio.NewScanner(t.stdOut)
	for scanner.Scan() {
		b := scanner.Bytes()
		// if false, we stop streaming
		if !cb(b) {
			return t
		}
	}

	return t
}
func (t *ShellCmd) Wait() error {
	if t.err != nil {
		return t.err
	}

	t.err = t.cmd.Wait()
	if t.err != nil {
		t.Tracef("result: %s [%s]\n", NeedNoErr(t.err), t.OutputErr())
	} else {
		if t.IsHaveOutput() {

			t.Tracef("result: %s: %s", NeedNoErr(t.err), t.Output())
		}
	}
	return t.err
}

func (t *ShellCmd) StartAndWait() error {
	if t.err != nil {
		return t.err
	}

	if t.err = t.cmd.Start(); t.err != nil {
		return t.err
	}

	scanner := bufio.NewScanner(t.stdOut)
	for scanner.Scan() {
		m := scanner.Text()
		fmt.Println(m)
	}

	t.err = t.cmd.Wait()
	if t.err != nil && t.err.Error() == "exec: Wait was already called" {
		t.err = nil
	}

	return t.err
}
func (t *ShellCmd) IsHaveOutput() bool {
	return t.haveResult && len(t.output) > 0
}
func (t *ShellCmd) Output() string {
	return string(t.output)
}

func (t *ShellCmd) OutputErr() string {
	return string(t.errorOutput)
}

func (t *ShellCmd) Cleanup() error {
	t.cmd = nil
	t.stdIn = nil
	t.errorOutput = nil
	t.output = nil
	t.stderr = nil
	t.outputBuf.Reset()
	t.stderrBuf.Reset()
	t.tracer = nil
	t.stdOut = nil
	t.stdErr = nil
	return t.err
}

func (t *ShellCmd) Tracef(format string, args ...interface{}) {
	if t.tracer == nil {
		return
	}

	t.tracer.Write([]byte(fmt.Sprintf(format, args...)))
}

func NeedNoErr(err error) string {
	if err == nil {
		return "[OK]"
	} else {
		return "[Error: " + err.Error() + "]"
	}
}
