package main

import (
	"bytes"
	"context"
	"os/exec"
)

type processResult struct {
	Stdout, Stderr []byte
	Err            error
	ExitCode       int
}
type executor interface {
	Execute(context.Context, ...string) processResult
}
type ghExecutor struct{ path string }

func locateGH() (executor, error) {
	p, e := exec.LookPath("gh")
	if e != nil {
		return nil, e
	}
	return ghExecutor{p}, nil
}
func (g ghExecutor) Execute(ctx context.Context, args ...string) processResult {
	if err := ctx.Err(); err != nil {
		return processResult{Err: err, ExitCode: -1}
	}
	argv := append([]string{"api", "graphql", "--hostname", "github.com"}, args...)
	cmd := exec.CommandContext(ctx, g.path, argv...)
	var out, errOut bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errOut
	err := cmd.Run()
	code := 0
	if err != nil {
		code = -1
		if cmd.ProcessState != nil {
			code = cmd.ProcessState.ExitCode()
		}
	}
	if ctx.Err() != nil {
		err = ctx.Err()
	}
	return processResult{Stdout: out.Bytes(), Stderr: errOut.Bytes(), Err: err, ExitCode: code}
}
