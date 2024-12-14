# gh-monorepo-dep-doctor [![go](https://github.com/chaspy/gh-monorepo-dep-doctor/actions/workflows/test.yml/badge.svg)](https://github.com/chaspy/gh-monorepo-dep-doctor/actions/workflows/test.yml) [![Go Report Card](https://goreportcard.com/badge/github.com/chaspy/gh-monorepo-dep-doctor)](https://goreportcard.com/report/github.com/chaspy/gh-monorepo-dep-doctor)

gh extension to execute [dep-doctor](https://github.com/kyoshidajp/dep-doctor) in monorepo for direct dipendencies.

## Motivation

1. I want to run dep-doctor on **monorepo** for all target services at once.

2. I want to detect **only direct dependencies**. dep-doctor's feature is that it detects the maintenance state of dependencies of dependencies, but I noticed that the detected ones are not controllable for me.

3. I want to notify Slack of the output results of dep-doctor. For this purpose, output in a format that can be handled programmatically is preferred. ([dep-doctor is output in text.](https://github.com/kyoshidajp/dep-doctor/blob/main/cmd/report.go)) In this tool, we implemented it in csv.

4. [Ruby] I wanted to check only official gems, and exclude forked gems and in-house gems from the diagnoses.

## Requirement

- [dep-doctor](https://github.com/kyoshidajp/dep-doctor) v1.2.1 or later

## Support Language

- Ruby, bundle

## Installation

```bash
gh extension install chaspy/gh-monorepo-dep-doctor
```

To upgrade,

```bash
gh extension upgrade chaspy/gh-monorepo-dep-doctor
```

## Usage

```bash
gh monorepo-dep-doctor
```

Output is like below.

```
api/Gemfile,grape-cache_control,not-maintained,https://github.com/karlfreeman/grape-cache_control
api/Gemfile,http_accept_language,not-maintained,https://github.com/iain/http_accept_language
back-office/Gemfile,sass,archived,https://github.com/sass/ruby-sass
```

A CSV file of the form `depenedency file, library, status, url` will be output

> [!NOTE]
> As it is executed asynchronously, the order of output is not guaranteed

## Ignore File Format

The `ignore.txt` file allows you to specify which library to ignore for each application in your monorepo.
The format is as follows:

```bash
# Ignore specific library for specific application
app1,library1
app2,library2

# Use wildcard (*) to ignore library for all applications
*,library3
```

### Format Description

- Each line follows the format: `application_name,library_name`
- Lines starting with `#` are treated as comments
- Wildcard `*` can be used:
  - `*,library1`: Ignore library1 for all applications
  - `app1,*`: Ignore all libraries for app1
- Empty lines are ignored

## Notification to Slack

If you want to notify the result of gh-monorepo-dep-doctor to Slack, use [Incoming Webhook](https://api.slack.com/messaging/webhooks).

```bash
gh monorepo-dep-doctor >> result.csv
```

```bash
SLACK_WEBHOOK_URL="please-add-webhook-url"

while IFS=, read -r file_path package_name maintenance_status url
do
  app=$(echo $file_path | cut -d'/' -f1)
  group_handle="group-handle-to-notify"
  group_id="group-id-to-notify"
  message=$(cat <<EOF
<!subteam^${group_id}|${group_handle}> The package *${package_name}* used by ${app} is in *${maintenance_status}*. Details: ${url}
EOF
)
echo $message
curl -X POST -H 'Content-type: application/json' \
  --data "{\"text\": \"$message\"}" \
  $SLACK_WEBHOOK_URL
done < result.csv

```

## Environment Variables

| Name              | Description                                         |
| ----------------- | --------------------------------------------------- |
| `MAX_CONCURRENCY` | The maximum number of concurrentcy. Defaults to 10. |
