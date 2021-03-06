package main

import (
	"fmt"
	"log"
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
func (c *CLI) renewCredentials() error {
	sess, err := session.NewSession(&aws.Config{})
	if err != nil {
		return fmt.Errorf("failed to retrieve a session: %v", err)
	}

	svc := sts.New(sess)

	// Convert role name to role ARN
	if !strings.HasPrefix(c.RoleArn, "arn:") {
		callerIdentity, err := svc.GetCallerIdentity(&sts.GetCallerIdentityInput{})
		if err != nil {
			return fmt.Errorf("failed to discover parent account caller identity: %w", err)
		}
		c.RoleArn = fmt.Sprintf("arn:aws:iam::%s:role/%s", *callerIdentity.Account, c.RoleArn)
	}

	req := &sts.AssumeRoleInput{
		RoleArn:         aws.String(c.RoleArn),
		RoleSessionName: aws.String(c.SessionName),
		DurationSeconds: aws.Int64(c.Duration),
	}

	for key := range c.SessionTags {
		log.Printf("Tagging Session: %q=%q", key, c.SessionTags[key])
		req.Tags = append(req.Tags, &sts.Tag{
			Key:   aws.String(key),
			Value: aws.String(c.SessionTags[key]),
		})
	}

	if len(c.TransitiveTags) > 0 {
		req.TransitiveTagKeys = aws.StringSlice(c.TransitiveTags)
	}

	if c.ExternalID != "" {
		req.ExternalId = &c.ExternalID
	}

	if c.SourceIdentity != "" {
		req.SourceIdentity = &c.SourceIdentity
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

	awsProfile := os.Getenv("AWS_PROFILE")
	if awsProfile == "" {
		awsProfile = "default"
	}

	cf.WriteString(fmt.Sprintf("[%s]\n", awsProfile))
	cf.WriteString(fmt.Sprintf("aws_access_key_id=%s\n", *creds.AccessKeyId))
	cf.WriteString(fmt.Sprintf("aws_secret_access_key=%s\n", *creds.SecretAccessKey))
	cf.WriteString(fmt.Sprintf("aws_session_token=%s\n", *creds.SessionToken))

	return os.Rename(cf.Name(), strings.TrimSuffix(cf.Name(), ".tmp"))
}
