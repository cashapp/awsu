package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"
)

// renewCredentials gets a new set of credentials from AWS
// rendering it into the credentials tempfile
func renewCredentials() error {
	sess, err := session.NewSession(&aws.Config{})
	if err != nil {
		return fmt.Errorf("failed to retrieve a session: %v", err)
	}

	svc := sts.New(sess)
	req := &sts.AssumeRoleInput{
		RoleArn:         &roleArn,
		RoleSessionName: sessionName,
		DurationSeconds: duration,
	}

	if *externalID != "" {
		req.ExternalId = externalID
	}

	assumedRole, err := svc.AssumeRole(req)
	if err != nil {
		return fmt.Errorf("failed to assume the role: %v", err)
	}

	err = renderCredentials(tempDir, assumedRole.Credentials)
	if err != nil {
		return err
	}

	credentialsExpiry := *assumedRole.Credentials.Expiration

	// Set the renewal time to 80% of the credential lifetime
	credentialsRenew = credentialsExpiry.Add(-1 * (time.Duration(time.Until(credentialsExpiry).Seconds()*20) * time.Second / 100))

	return nil
}

// renderCredentials creates a new temporary credentials file,
// and uses os.Rename to replace the existing file
func renderCredentials(dir string, creds *sts.Credentials) error {
	cf, err := os.OpenFile(filepath.Join(dir, "credentials.tmp"), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return err
	}
	defer cf.Close()

	cf.WriteString("[default]\n")
	cf.WriteString(fmt.Sprintf("aws_access_key_id=%s\n", *creds.AccessKeyId))
	cf.WriteString(fmt.Sprintf("aws_secret_access_key=%s\n", *creds.SecretAccessKey))
	cf.WriteString(fmt.Sprintf("aws_session_token=%s\n", *creds.SessionToken))

	return os.Rename(cf.Name(), strings.TrimSuffix(cf.Name(), ".tmp"))
}
