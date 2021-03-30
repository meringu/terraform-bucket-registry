terraform {
  required_providers {
    example = {
      source  = "registry/foo/example"
    }
  }
}

provider "example" {}

data example_foo main {}

output foo {
  value = data.example_foo.main.bar
}
