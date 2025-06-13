module github.com/protolite/benchmark

go 1.21

require (
	github.com/bufbuild/protocompile v0.14.1
	github.com/protolite v0.0.0
	google.golang.org/protobuf v1.34.2
)

require golang.org/x/sync v0.8.0 // indirect

replace github.com/protolite => ../
