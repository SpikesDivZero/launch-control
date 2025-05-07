module example-app

go 1.24.2

replace github.com/spikesdivzero/launch-control v0.0.0 => ..

require (
	github.com/dpotapov/slogpfx v0.0.0-20230917063348-41a73c95c536
	github.com/lmittmann/tint v1.0.7
	github.com/mattn/go-colorable v0.1.14
	github.com/spikesdivzero/launch-control v0.0.0
)

require (
	github.com/mattn/go-isatty v0.0.20 // indirect
	golang.org/x/sys v0.29.0 // indirect
)
