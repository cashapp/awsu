package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/alecthomas/kong"
)

var tempDir string
var credentialsRenew time.Time

type CLI struct {
	SessionName    string            `name:"session-name" short:"s" help:"Session name of the role to assume" default:"awsu" env:"USER"`
	ExternalID     string            `name:"external-id" short:"e" help:"ExternalID to authenticate the request"`
	Duration       int64             `name:"duration" short:"d" help:"Duration of the session" default:"3600"`
	Verbose        bool              `name:"verbose" short:"v" help:"Verbose error logging"`
	SessionTags    map[string]string `name:"session-tags" short:"t" help:"Session tags to apply to the role-assumption (eg: -t tag1=batman)"`
	TransitiveTags []string          `name:"transitive-tags" short:"x" help:"Keys for session tags which are transitive (eg: -x tag1)"`
	SourceIdentity string            `name:"source-identity" short:"i" help:"Source identity to set for this session"`
	RoleArn        string            `arg:"" help:"The role name or ARN you want to assume"`
	Command        []string          `arg:"" passthrough:""`
}

func main() {
	ctx := kong.Parse(
		&CLI{},
		kong.Name("awsu"),
		kong.Description("Switch-user for AWS"),
	)

	err := ctx.Run()
	ctx.FatalIfErrorf(err)
}

func (c *CLI) Run(ctx *kong.Context) error {
	var err error
	tempDir, err = ioutil.TempDir("", "awsu")
	if err != nil {
		log.Fatalf("Failed to create a tempdir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	err = c.renewCredentials()
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		for {
			time.Sleep(time.Until(credentialsRenew))

			if c.Verbose {
				log.Print("awsu: Renewing credentials")
			}

			if err := c.renewCredentials(); err != nil {
				// We don't exit here - let the sub-command die it's own way
				log.Printf("awsu: Failed to renew credentials: %v", err)
				// Renew in a minute
				credentialsRenew = time.Now().Add(time.Minute)
			}

			if c.Verbose {
				log.Printf("awsu: Credentials renewed, next renewal in  %s", humanDur(time.Until(credentialsRenew)))
			}
		}
	}()

	cmd := exec.Command(c.Command[0], c.Command[1:]...)
	cmd.Env = []string{fmt.Sprintf("AWS_SHARED_CREDENTIALS_FILE=%s", filepath.Join(tempDir, "credentials"))}

	// We strip any AWS_ vars (except region vars, and profile), to ensure we have precedence over credentials
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, "AWS_REGION=") ||
			strings.HasPrefix(e, "AWS_DEFAULT_REGION=") ||
			strings.HasPrefix(e, "AWS_PROFILE=") ||
			!strings.HasPrefix(e, "AWS_") {
			cmd.Env = append(cmd.Env, e)
		}
	}

	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin

	log.Printf("Running %s with assumedRole %s, renewal in %s", c.Command, c.RoleArn, humanDur(time.Until(credentialsRenew)))
	err = cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		return fmt.Errorf("an error occurred waiting for cmd; %w", err)
	}
	return nil
}

func humanDur(d time.Duration) string {
	if seconds := int(d.Seconds()); seconds < 60 {
		return fmt.Sprintf("%ds", seconds)
	}
	return fmt.Sprintf("%dm", int(d.Minutes()))
}
