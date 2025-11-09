# tffumpt

A stricter terraform fmt.

## About

tffumpt is compatible with terraform fmt, i.e. every file with valid formatting
according to tffumpt will also be valid according to terraform fmt, the reverse
isn't necessarily true as tffumpt adds additional formatting rules terraform fmt
isn't opinionated about.

The inspiration for this project comes from [gofumpt](https://github.com/mvdan/gofumpt).

Since core formatting logic in terraform fmt hasn't changed since before the license
change, this project contains a fork of pre-BUSL terraform fmt and is therefore
free of any BUSL code.

## Getting started

### Installation

Using the install script:

```
curl -sSfL https://raw.githubusercontent.com/AleksaC/tffumpt/refs/heads/main/install.sh | sh
```

or

```
wget -O- -nv https://raw.githubusercontent.com/AleksaC/tffumpt/refs/heads/main/install.sh | sh
```

Direct download (replace `darwin` and `arm64` with your operating system and CPU architecture):

```
cd /tmp
curl -O https://github.com/AleksaC/tffumpt/releases/download/v0.1.0/tffumpt-v0.1.0-darwin-arm64.zip
unzip -oqj "${ARCHIVE_NAME}" -x "LICENSE*" "README*"
mv ./tffumpt ~/.local/bin/
```

Using the go toolchain:

```shell
go install github.com/AleksaC/tffumpt/cmd/tffumpt@latest
```

### Usage

Run the help command to learn how to use the tffumpt CLI:

```
tffumpt -help
```

Additionally, tffumpt mirrors terraform fmt command line interface, so you may
find some useful information in the terraform fmt command [reference](https://developer.hashicorp.com/terraform/cli/commands/fmt)

**Example**: non-recursively format all terraform files in the current directory:

```shell
tffumpt
```

### Integrations

tffumpt can be integrated with various other systems and tools. Currently, the only
officially supported integration is with [pre-commit](https://pre-commit.com/).

The configuration you need to add to `.pre-commit-config.yaml` looks something like this:

```yaml
repos:
  - repo: https://github.com/AleksaC/tffumpt
    rev: v0.1.1
    hooks:
      - id: tffumpt
```

## Rules

In the spirit of terraform fmt, as well as many other formatters, tffumpt doesn't
allow configuration. New rules, as well as modifications to the existing ones, can
be accepted if you make a good case for doing so.

Note that the real value of a formatter isn't in producing aesthetically pleasing
code (which we do our best to do anyway), but to produce consistent code.

### Trailing newlines

Adds a newline at the end of file if it's missing.

### No extra newlines

Limits the number of blank lines between blocks and attributes to 1. Removes leading
and trailing newlines inside blocks and multiline literals.

```hcl


resource "foo" "bar" {

  a = "b"
}


resource "bar" "baz" {
  b = "c"

}
```

```hcl
resource "foo" "bar" {
  a = "b"
}

resource "bar" "baz" {
  b = "c"
}
```

### Newlines between top level blocks

Adds a blank line between the top-level blocks.

```hcl
resource "foo" "bar" {
  a = "b"
}
resource "bar" "baz" {
  b = "c"
}
```

```hcl
resource "foo" "bar" {
  a = "b"
}

resource "bar" "baz" {
  b = "c"
}
```

### `#` for comments

Enforces usage of `#` for single-line comments.

```hcl
// comment
```

```hcl
# comment
```

### Space before comment text

Adds a space between `#` and the comment text.

```hcl
#comment
```

```hcl
# comment
```

### No extraneous parentheses

Removes parentheses that serve no purpose

```hcl
foo = lower(("Hello"))
bar = (2 + 2)
```

```hcl
foo = lower("Hello")
bar = 2 + 2
```

### `=` over `:` in map literals

Enforces usage of `=` between key and value in map literals.

```hcl
map = {
  foo : "bar"
  bar : "baz"
}
```

```hcl
map = {
  foo = "bar"
  bar = "baz"
}
```

### No extraneous quotes in map literal keys

Removes quotes from map keys that don't need to be quoted.

```hcl
map = {
  "foo" = "bar"
  "bar" = "baz"
}
```

```hcl
map = {
  foo = "bar"
  bar = "baz"
}
```

### No unnecessary interpolation in map keys

Replaces traditional string interpolation with parenthesized expressions for dynamic map keys.

```hcl
map = {
  "${var.foo}" = "bar"
}
```

```hcl
map = {
  (var.foo) = "bar"
}
```

### Bare expression in map keys wrapping

Wraps expression in parentheses if it's used as a map key.

```hcl
map = {
  upper(var.foo) = "bar"
}
```

```hcl
map = {
  (upper(var.foo)) = "bar"
}
```

### Map for expression key unwrapping

Removes unnecessary string interpolation in map for expression keys.

```hcl
map = {
  for foo in var.bar: "${foo.a}" => foo.b
}
```

```hcl
map = {
  for foo in var.bar: foo.a => foo.b
}
```

### No commas in multiline map literals

Removes commas between map items in multiline map literals.

```hcl
map = {
  foo = "bar",
  bar = "baz"
}
```

```hcl
map = {
  foo = "bar"
  bar = "baz"
}
```

### Trailing commas in multiline list literals

Adds a trailing comma in multiline list literals.

```hcl
list = [
  "foo",
  "bar",
  "baz"
]
```

```hcl
list = [
  "foo",
  "bar",
  "baz",
]
```

## Roadmap

This project was primarily developed for personal use, and that purpose has been
largely fulfilled, therefore it will mostly receive bug fixes if they pop up. If
there is enough community interest and support, things like editor support,
shell completions, first-party CI support and additional rules may be implemented.

If you'd like to contribute to this project check [CONTRIBUTING.md](./CONTRIBUTING.md)
for details.
