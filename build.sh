set -eu
go build .
mv gh-monorepo-dep-doctor "../../${GH_REPO}"