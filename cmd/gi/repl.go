package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/gijit/gi/pkg/compiler"
	"path"
)

var ProgramName string = path.Base(os.Args[0])

// Makefile tells the linker for main to set these,
// propagate to compiler package here.
func setCompilerVersion() {
	compiler.LastGitCommitHash = LastGitCommitHash
	compiler.BuildTimeStamp = BuildTimeStamp
	compiler.NearestGitTag = NearestGitTag
	compiler.GitBranch = GitBranch
	compiler.GoVersion = GoVersion
	compiler.LuajitVersion = LuajitVersion
}

func main() {
	setCompilerVersion()
	myflags := flag.NewFlagSet("gi", flag.ExitOnError)
	cfg := compiler.NewGIConfig()
	cfg.DefineFlags(myflags)

	err := myflags.Parse(os.Args[1:])
	err = cfg.ValidateConfig()
	if err != nil {
		log.Fatalf("%s command line flag error: '%s'", ProgramName, err)
	}
	if !cfg.Quiet {
		fmt.Printf(
			`====================
gijit: a go interpreter, just-in-time.
====================
https://github.com/gijit/gi
Copyright (c) 2018, Jason E. Aten. All rights reserved.
License: 3-clause BSD. See the LICENSE file at
https://github.com/gijit/gi/blob/master/LICENSE
====================
  [ gijit/gi is an interactive Golang environment,
    also known as a REPL or Read-Eval-Print-Loop.]
  [ at the gi> prompt, type ctrl-d to exit.]
  [ at the gi> :?   or :help for help.]
  [ $ gi -h for flag help, when first launching gijit.]
  [ $ gi -q to start quietly, without this banner.]
====================
%s
==================
`, compiler.Version())
	}

	cfg.LuajitMain()
}
