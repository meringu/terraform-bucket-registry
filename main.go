package main

import (
	"github.com/meringu/terraform-bucket-registry/pkg/cmd"
)

var version = "v0.0.0-local"
var commit = "HEAD"

func init() {
	cmd.Version = version
	cmd.Commit = commit
}

func main() {
	cmd.Execute()
}
