package main

import (
	"os"
	"runtime"
	"syscall"
    "golang.org/x/term"
)

type Runner string

const (
	RunnerGitHub Runner = "github"
	RunnerGitLab Runner = "gitlab"
	RunnerBitbucket Runner = "bitbucket"
	RunnerUnknown Runner = "unknown"
	RunnerLocal Runner = "local"
)

type Platform struct{
	System string  `json:"system"`
	Architecture string  `json:"architecture"`
	Runner  `json:"runner"`
}

type Terminal struct{
	Width int `json:"width"`
	Height int `json:"height"`
	Teletype bool `json:"teletype"`
}

type Environment struct {
	Platform  `json:"platform"`
	Terminal `json:"terminal"`
}

func GetEnvironment() (*Environment, error) {
	e := new(Environment)

	e.Platform.System = runtime.GOOS
	e.Platform.Architecture = runtime.GOARCH

    width, height, err := term.GetSize(syscall.Stdin)
    if err != nil {
        return nil, err
    }
	
	e.Terminal.Width = width
	e.Terminal.Height = height
	e.Terminal.Teletype = term.IsTerminal(syscall.Stdin)

	switch _, ok := os.LookupEnv("CI"); {
	case os.Getenv("GITHUB_ACTIONS") != "":
		e.Platform.Runner = RunnerGitHub
	case os.Getenv("GITLAB_CI") != "":
		e.Platform.Runner = RunnerGitLab
	case os.Getenv("BITBUCKET_BUILD_NUMBER") != "":
		e.Platform.Runner = RunnerBitbucket
	case ok:
		e.Platform.Runner = RunnerUnknown
	default:
		e.Platform.Runner = RunnerLocal
	}

	return e, nil
}