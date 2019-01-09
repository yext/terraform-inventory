package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/adammck/venv"
	"github.com/blang/vfs"
)

var version = flag.Bool("version", false, "print version information and exit")
var list = flag.Bool("list", false, "list mode")
var host = flag.String("host", "", "host mode")
var inventory = flag.Bool("inventory", false, "inventory mode")

func appendState(a state, b state) state {
	return state{
		Modules: append(a.Modules, b.Modules...),
	}
}

func getState(directory string) state {
	os.Chdir(directory)

	cmd := exec.Command("terragrunt", "state", "pull")
	cmd.Dir = directory
	var out bytes.Buffer
	cmd.Stdout = &out

	var s state

	err := cmd.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error running `terragrunt state pull` in directory %s, %s, attempting to recurse\n", directory, err)
		files, err := ioutil.ReadDir(directory)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading directory %s, %s, exiting\n", directory, err)
			os.Exit(1)
		}
		for _, f := range files {
			if f.IsDir() {
				nextdir := fmt.Sprintf("%s/%s", directory, f.Name())
				s = appendState(s, getState(nextdir))
			}
		}
		return s
	}

	// Since state.read is addititve, it should parse this correctly
	err = s.read(&out)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading `terragrunt state pull` in directory %s, output: %s\n", directory, err)
		return s
	}

	return s

}

func main() {
	flag.Parse()
	file := flag.Arg(0)

	if *version == true {
		fmt.Printf("%s version %s\n", os.Args[0], versionInfo())
		return
	}

	fs := vfs.OS()
	if file == "" {

		env := venv.OS()
		file = GetInputPath(fs, env)
	}

	if !*list && *host == "" && !*inventory {
		fmt.Fprint(os.Stderr, "Either --host or --list must be specified")
		os.Exit(1)
	}

	path, err := filepath.Abs(file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid file: %s\n", err)
		os.Exit(1)
	}

	f, err := fs.Stat(path)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid file: %s\n", err)
		os.Exit(1)
	}

	var s state

	if !f.IsDir() {
		stateFile, err := os.Open(path)
		defer stateFile.Close()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening tfstate file: %s\n", err)
			os.Exit(1)
		}

		err = s.read(stateFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading tfstate file: %s\n", err)
			os.Exit(1)
		}
	}

	if f.IsDir() {
		// Getstate will simply exit on error
		cwd, _ := os.Getwd()
		s = getState(cwd)
	}

	if s.Modules == nil {
		fmt.Printf("Usage: %s [options] path\npath: this is either a path to a state file or a folder from which `terraform commands` are valid\n", os.Args[0])
		os.Exit(1)
	}

	if *list {
		os.Exit(cmdList(os.Stdout, os.Stderr, &s))

	} else if *inventory {
		os.Exit(cmdInventory(os.Stdout, os.Stderr, &s))

	} else if *host != "" {
		os.Exit(cmdHost(os.Stdout, os.Stderr, &s, *host))

	}
}
