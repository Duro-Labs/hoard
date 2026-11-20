package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Duro-Labs/hoard"
	"github.com/Duro-Labs/hoard/internal/agent"
	"github.com/Duro-Labs/hoard/internal/assets"
	"github.com/Duro-Labs/hoard/internal/config"
	"github.com/Duro-Labs/hoard/internal/env"
)

var version = "dev"

const usage = `usage:
  hoard install --agent claude|codex|hermes [--vault <path>]
  hoard uninstall --agent claude|codex|hermes
  hoard doctor [--agent claude|codex|hermes]
  hoard version
`

func main() {
	e, err := env.FromOS()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(run(os.Args[1:], e, os.Stdout, os.Stderr, time.Now))
}

func run(args []string, e env.Env, stdout, stderr io.Writer, now func() time.Time) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, usage)
		return 2
	}
	fl := flag.NewFlagSet(args[0], flag.ContinueOnError)
	fl.SetOutput(stderr)
	name := fl.String("agent", "", "agent")
	vault := fl.String("vault", "", "vault root")
	if err := fl.Parse(args[1:]); err != nil {
		return 2
	}
	stamp := now().UTC().Format("20060102T150405Z")
	var err error
	switch args[0] {
	case "install":
		err = install(e, *name, *vault, stamp, stdout)
	case "uninstall":
		err = withAgent(*name, func(a agent.Agent) error { return a.Uninstall(e, stamp) })
		if err == nil {
			fmt.Fprintf(stdout, "hoard removed from %s\n", *name)
		}
	case "doctor":
		return doctor(e, *name, stdout)
	case "version":
		fmt.Fprintln(stdout, version)
		return 0
	default:
		fmt.Fprint(stderr, usage)
		return 2
	}
	if err != nil {
		fmt.Fprintln(stderr, "hoard:", err)
		return 1
	}
	return 0
}

func withAgent(name string, fn func(agent.Agent) error) error {
	a, err := agent.ByName(name)
	if err != nil {
		return err
	}
	return fn(a)
}

func install(e env.Env, name, vault, stamp string, stdout io.Writer) error {
	a, err := agent.ByName(name)
	if err != nil {
		return err
	}
	c, err := config.Read(e.ConfigFile)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	if vault != "" {
		abs, err := filepath.Abs(vault)
		if err != nil {
			return err
		}
		if st, err := os.Stat(abs); err != nil || !st.IsDir() {
			return fmt.Errorf("vault %s is not a directory", abs)
		}
		c.VaultRoot = abs
	}
	if c.VaultRoot == "" {
		return fmt.Errorf("no vault configured: pass --vault <path> on first install")
	}
	if c.DefaultScope == "" {
		c.DefaultScope = "personal"
	}
	if err := config.Write(e.ConfigFile, c); err != nil {
		return err
	}
	if err := assets.Extract(hoard.Assets, e.DataDir); err != nil {
		return err
	}
	if err := a.Install(e, stamp); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "hoard installed for %s (vault %s). Run `hoard doctor --agent %s` to check.\n", name, c.VaultRoot, name)
	return nil
}

func doctor(e env.Env, name string, stdout io.Writer) int {
	var fs []agent.Finding
	for _, bin := range []string{"jq", "rg", "git"} {
		if _, err := exec.LookPath(bin); err != nil {
			fs = append(fs, agent.Finding{Level: "fail", Msg: bin + " not found in PATH"})
		}
	}
	if out, err := exec.Command("bash", "-c", "echo ${BASH_VERSINFO[0]}").Output(); err != nil {
		fs = append(fs, agent.Finding{Level: "fail", Msg: "bash not found"})
	} else if v, _ := strconv.Atoi(strings.TrimSpace(string(out))); v < 4 {
		fs = append(fs, agent.Finding{Level: "fail", Msg: "bash in PATH is older than 4"})
	}
	c, err := config.Read(e.ConfigFile)
	switch {
	case err != nil:
		fs = append(fs, agent.Finding{Level: "fail", Msg: "config " + e.ConfigFile + ": " + err.Error()})
	case c.VaultRoot == "":
		fs = append(fs, agent.Finding{Level: "fail", Msg: "config: HOARD_VAULT_ROOT is empty"})
	default:
		if st, err := os.Stat(c.VaultRoot); err != nil || !st.IsDir() {
			fs = append(fs, agent.Finding{Level: "fail", Msg: "config: vault " + c.VaultRoot + " is not a directory"})
		} else {
			fs = append(fs, agent.Finding{Level: "ok", Msg: "config: vault " + c.VaultRoot})
		}
	}
	agents := agent.All()
	if name != "" {
		a, err := agent.ByName(name)
		if err != nil {
			fmt.Fprintln(stdout, "fail", err)
			return 1
		}
		agents = []agent.Agent{a}
	}
	for _, a := range agents {
		fs = append(fs, a.Check(e)...)
	}
	code := 0
	for _, f := range fs {
		fmt.Fprintf(stdout, "%-4s %s\n", f.Level, f.Msg)
		if f.Level == "fail" {
			code = 1
		}
	}
	return code
}
