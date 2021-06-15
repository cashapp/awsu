package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"
)

var tempDir string
var credentialsRenew time.Time

// CLI Flags
var (
	sessionName = flag.String("s", "awsu", "Session name of the role to assume")
	externalID  = flag.String("e", "", "ExternalID to authenticate the request")
	duration    = flag.Int64("d", 3600, "Duration of the session")
	verbose     = flag.Bool("v", false, "Verbose error logging")
	roleArn     string
	command     string
)

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s [options] <role-arn> -- <command> [arguments]\n\nOptions: \n", path.Base(os.Args[0]))
	flag.PrintDefaults()
	fmt.Println()
}

func main() {
	flag.Usage = usage
	flag.Parse()

	args := flag.Args()
	if len(args) < 2 {
		usage()
		os.Exit(1)
	}
	roleArn = args[0]
	command = args[1]
	args = args[2:]

	var err error
	tempDir, err = ioutil.TempDir("", "awsu")
	if err != nil {
		log.Fatalf("Failed to create a tempdir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	err = renewCredentials()
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		for {
			time.Sleep(time.Until(credentialsRenew))

			if *verbose {
				log.Print("awsu: Renewing credentials")
			}

			if err := renewCredentials(); err != nil {
				// We don't exit here - let the sub-command die it's own way
				log.Print("awsu: Failed to renew credentials")
				// Renew in a minute
				credentialsRenew = time.Now().Add(time.Minute)
			}

			if *verbose {
				log.Printf("awsu: Credentials renewed, next renewal in  %s", humanDur(time.Until(credentialsRenew)))
			}
		}
	}()

	cmd := exec.Command(command, args...)
	cmd.Env = []string{fmt.Sprintf("AWS_SHARED_CREDENTIALS_FILE=%s", filepath.Join(tempDir, "credentials"))}

	// We strip any AWS_ vars (except region vars), to ensure we have precedence over credentials
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, "AWS_REGION") ||
			strings.HasPrefix(e, "AWS_DEFAULT_REGION") ||
			!strings.HasPrefix(e, "AWS_") {
			cmd.Env = append(cmd.Env, e)
		}
	}

	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin

	log.Printf("Running %s with assumedRole %s, renewal in %s", command, roleArn, humanDur(time.Until(credentialsRenew)))
	err = cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		log.Fatalf("An error occurred waiting for cmd: %v", err)
	}
}

func humanDur(d time.Duration) string {
	if seconds := int(d.Seconds()); seconds < 60 {
		return fmt.Sprintf("%ds", seconds)
	}
	return fmt.Sprintf("%dm", int(d.Minutes()))
}
