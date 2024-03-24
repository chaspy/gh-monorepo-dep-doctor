# gh-monorepo-dep-doctor

gh extension to execute [dep-doctor](https://github.com/kyoshidajp/dep-doctor) in monorepo for direct dipendencies.

## Motivation

1. I want to run dep-doctor on **monorepo** for all target services at once.

2. I want to detect **only direct dependencies**. dep-doctor's feature is that it detects the maintenance state of dependencies of dependencies, but I noticed that the detected ones are not controllable for me.

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

## Ignore file

To ignore a specific library, you can create a `.gh-monorepo-dep-doctor-ignore` file in the root of the repository.

The format of the file is as follows.

```bash
# This file allow a comment with "#"
# Lines beginning with # and blank lines are ignored
# Also, files beginning with "#" are ignored.
library-name
another-library-name # You can leave a comment here
```

## Environment Variables

|Name|Description|
|---|---|
| `MAX_CONCURRENCY` | The maximum number of concurrentcy. Defaults to 10.|
