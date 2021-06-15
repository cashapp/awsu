# awsu - Switch-user for AWS

## Introduction

Sometimes you find the need to switch roles on AWS from the command-line.
While this is possible using the aws cli, it's quite cumbersome:

```bash
ASSUME=$(aws sts assume-role --role-arn "${TARGET_ACCOUNT_ROLE}" --role-session-name="awsu-bash")
export AWS_ACCESS_KEY_ID=$(echo "$ASSUME" | jq '.Credentials.AccessKeyId')
export AWS_SECRET_ACCESS_KEY=$(echo "$ASSUME" | jq '.Credentials.SecretAccessKey')
export AWS_SESSION_TOKEN=$(echo "$ASSUME" | jq '.Credentials.SessionToken')
```

Furthermore, depending on the duration of your tokens, they may expire before you're finished using them.

AWSU is designed to handle this for you, including performing token renewals at 80% of the token's lifetime.

## Quickstart

```bash
$ awsu arn:aws:iam::468901978831:role/superman <command>
2021/06/08 16:10:07 Running bash with assumedRole arn:aws:iam::468901978831:role/superman, renewal in 47m
$ 
```

## Usage

<pre>

Usage: awsu [options] &lt;role-arn&gt; -- &lt;command&gt; [arguments]

Options: 
  -d int
        Duration of the session (default 3600)
  -e string
        ExternalID to authenticate the request
  -s string
        Session name of the role to assume (default "awsu")
  -v    Verbose error logging
</pre>


## License

<pre>
Copyright 2021 Square Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
</pre>
