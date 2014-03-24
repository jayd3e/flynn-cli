package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"syscall"

	"github.com/BurntSushi/toml"
)

var (
	apiURL         = "http://localhost:1200"
	stdin          = bufio.NewReader(os.Stdin)
	configFilePath = os.Getenv("HOME") + "/.flynnrc"
)

type Config struct {
	Servers []*server
}

type server struct {
	GitHost   string
	ApiUrl    string
	ApiKey    string
	ApiTlsPin string
}

type Command struct {
	// args does not include the command name
	Run  func(cmd *Command, args []string)
	Flag flag.FlagSet

	Usage string // first word is the command name
	Short string // `hk help` output
	Long  string // `hk help cmd` output
}

func (c *Command) printUsage() {
	if c.Runnable() {
		fmt.Printf("Usage: flynn %s\n\n", c.Usage)
	}
	fmt.Println(strings.Trim(c.Long, "\n"))
}

func (c *Command) Name() string {
	name := c.Usage
	i := strings.Index(name, " ")
	if i >= 0 {
		name = name[:i]
	}
	return name
}

func (c *Command) Runnable() bool {
	return c.Run != nil
}

const extra = " (extra)"

func (c *Command) List() bool {
	return c.Short != "" && !strings.HasSuffix(c.Short, extra)
}

func (c *Command) ListAsExtra() bool {
	return c.Short != "" && strings.HasSuffix(c.Short, extra)
}

func (c *Command) ShortExtra() string {
	return c.Short[:len(c.Short)-len(extra)]
}

// Running `flynn help` will list commands in this order.
var commands = []*Command{
	cmdHelp,
	cmdLogin,
	cmdCreate,
	cmdRun,
	cmdPs,
	cmdLogs,
	cmdScale,
	cmdDomain,
}

var (
	flagApp  string
	flagLong bool
	config   Config
)

func main() {
	log.SetFlags(0)

	// Load our stored config.  Located at ~/.flynnrc
	err := loadConfig()
	if err != nil {
		fmt.Errorf("failed to load config file", err)
		os.Exit(2)
	}

	args := os.Args[1:]

	// Determine the correct value of the app name specified by the flag
	if len(args) >= 2 && "-a" == args[0] {
		flagApp = args[1]
		args = args[2:]

		// If a remote was specified, resolve the app from the remote name
		if gitRemoteApp, err := appFromGitRemote(flagApp); err == nil {
			flagApp = gitRemoteApp
		}
	}

	// Determine the url of the API
	if s := os.Getenv("FLYNN_API_URL"); s != "" {
		apiURL = strings.TrimRight(s, "/")
	} else {
		for _, serv := range config.Servers {
			urlFromRemote, _ := urlFromGitRemote(serv.GitHost)
			if urlFromRemote == serv.GitHost {
				apiURL = serv.ApiUrl
			}
		}
	}

	// Display usage information if a command isn't specified
	if len(args) < 1 {
		usage()
	}

	// Run the command
	for _, cmd := range commands {
		if cmd.Name() == args[0] && cmd.Run != nil {
			cmd.Flag.Usage = func() {
				cmd.printUsage()
			}
			if err := cmd.Flag.Parse(args[1:]); err != nil {
				os.Exit(2)
			}
			cmd.Run(cmd, cmd.Flag.Args())
			return
		}
	}

	fmt.Fprintf(os.Stderr, "Unknown command: %s\n", args[0])
	usage()
}

func loadConfig() error {
	if _, err := toml.DecodeFile(configFilePath, &config); err != nil {
		return err
	}
	return nil
}

func writeConfig() error {
	w, _ := os.OpenFile(configFilePath, os.O_RDWR|os.O_CREATE, 0666)
	encoder := toml.NewEncoder(w)
	if err := encoder.Encode(config); err != nil {
		return err
	}
	return nil
}

func app() (string, error) {
	if flagApp != "" {
		return flagApp, nil
	}

	if app := os.Getenv("FLYNN_APP"); app != "" {
		return app, nil
	}

	gitRemoteApp, err := appFromGitRemote("flynn")
	if err != nil {
		return "", err
	}

	return gitRemoteApp, nil
}

func appFromGitRemote(remote string) (string, error) {
	url, _ := urlFromGitRemote(remote)
	app, _ := appFromGitRemoteUrl(url)
	return app, nil
}

func urlFromGitRemote(remote string) (string, error) {
	b, err := exec.Command("git", "config", "remote."+remote+".url").Output()
	if err != nil {
		if isNotFound(err) {
			wdir, _ := os.Getwd()
			return "", fmt.Errorf("could not find git remote "+remote+" in %s", wdir)
		}
		return "", err
	}

	return string(b), nil
}

func gitRemoteUrlFromApp(name string) (string, error) {
	return "git@" + config.Servers[0].GitHost + ":" + name, nil
}

var appFromRemoteUrlRegex, _ = regexp.Compile(`(?:ssh://)?(?:\w+)@(?:localhost(?::\d+)?/|\d{1,3}.\d{1,3}.\d{1,3}.\d{1,3}(?::\d+)?/|.+:)(.+)`)

func appFromGitRemoteUrl(url string) (string, error) {
	url = strings.Trim(url, "\r\n ")

	match := appFromRemoteUrlRegex.FindStringSubmatch(url)
	if match == nil {
		return "", fmt.Errorf("could not find app name in " + url + " git remote")
	}

	return match[1], nil
}

func isNotFound(err error) bool {
	if ee, ok := err.(*exec.ExitError); ok {
		if ws, ok := ee.ProcessState.Sys().(syscall.WaitStatus); ok {
			return ws.ExitStatus() == 1
		}
	}
	return false
}

func mustApp() string {
	name, err := app()
	if err != nil {
		log.Fatal(err)
	}
	return name
}

func must(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func listRec(w io.Writer, a ...interface{}) {
	for i, x := range a {
		fmt.Fprint(w, x)
		if i+1 < len(a) {
			w.Write([]byte{'\t'})
		} else {
			w.Write([]byte{'\n'})
		}
	}
}
