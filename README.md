# wr
experimental reimplementation of wr using TDD

## Developers
To develop this code base, you should use TDD. To aid this, the test suite is
written using GoConvey.

To install goconvey:
```
cd ~/somewhere_else
git checkout https://github.com/smartystreets/goconvey.git
go build
mv goconvey $GOPATH/bin/
```

To use goconvey:
```
cd ~/your_clone_of_this_repository
goconvey &
```
This will pop up a browser window which will aid in the red-green-refactor
cycle.

To run the tests on the command line:
`go test ./...` or `make test` or `make race`

To run the benchmarks:
`go test -run Bench -bench=. ./...` or `make bench`

Before committing any code, you should make sure you haven't introduced any
linting errors. First install the linters:
`curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | sh -s -- -b $(go env GOPATH)/bin v1.31.0`

Then:
`make lint`