//nolint:lll
package format

import (
	"bytes"
	"testing"

	diff "github.com/AleksaC/tffumpt/diff"
)

type TestCase struct {
	Name     string
	Filename string
	Source   []byte
	Target   []byte
}

func AssertEqual(t *testing.T, name string, src, target []byte) {
	t.Run(name, func(t *testing.T) {
		if !bytes.Equal(src, target) {
			diff, _ := diff.BytesDiff(src, target, "")
			t.Fatalf("%s", diff)
		}
	})
}

func TestFormat(t *testing.T) {
	cases := []TestCase{
		{
			Name:     "Empty",
			Filename: "test.tf",
			Source:   []byte(""),
			Target:   []byte(""),
		},
		{
			Name:     "Newline",
			Filename: "test.tf",
			Source:   []byte("\n"),
			Target:   []byte("\n"),
		},
		{
			Name:     "Newlines",
			Filename: "test.tf",
			Source:   []byte("\n\n\n"),
			Target:   []byte("\n"),
		},
		{
			Name:     "Invalid",
			Filename: "test.tf",
			Source:   []byte("locals {\n  unclosed = true "),
			Target:   nil,
		},
	}

	for _, test := range cases {
		res, _ := Format(test.Source, test.Filename)
		AssertEqual(t, test.Name, res, test.Target)
	}
}

func TestParsing(t *testing.T) {
	cases := []TestCase{
		{
			Name:     "Number",
			Filename: "test.tf",
			Source:   []byte("locals {\n  num = 120.5\n}\n"),
			Target:   []byte("locals {\n  num = 120.5\n}\n"),
		},
		{
			Name:     "NumberComment",
			Filename: "test.tf",
			Source:   []byte("locals {\n  num = 120.5 # test\n}\n"),
			Target:   []byte("locals {\n  num = 120.5 # test\n}\n"),
		},
		{
			Name:     "NumberExpression",
			Filename: "test.tf",
			Source:   []byte("locals {\n  num = 2 + 4 / 2 + (3 - 1) / 2\n}\n"),
			Target:   []byte("locals {\n  num = 2 + 4 / 2 + (3 - 1) / 2\n}\n"),
		},
		{
			Name:     "Boolean",
			Filename: "test.tf",
			Source:   []byte("locals {\n  foo = true\n}\n"),
			Target:   []byte("locals {\n  foo = true\n}\n"),
		},
		{
			Name:     "String",
			Filename: "test.tf",
			Source:   []byte("locals {\n  foo = \"bar\"\n}\n"),
			Target:   []byte("locals {\n  foo = \"bar\"\n}\n"),
		},
		{
			Name:     "Interpolation",
			Filename: "test.tf",
			Source:   []byte("locals {\n  foo = \"${var.bar}-sth\"\n}\n"),
			Target:   []byte("locals {\n  foo = \"${var.bar}-sth\"\n}\n"),
		},
		{
			Name:     "Directive",
			Filename: "test.tf",
			Source:   []byte("locals {\n  foo = \"%{if var.foo == \"true\"} big %{endif}\"\n}\n"),
			Target:   []byte("locals {\n  foo = \"%{if var.foo == \"true\"} big %{endif}\"\n}\n"),
		},
		{
			Name:     "DirectiveWhitespaceStripping",
			Filename: "test.tf",
			Source:   []byte("locals {\n  foo = \"%{if var.foo == \"true\"~} big %{~endif}\"\n}\n"),
			Target:   []byte("locals {\n  foo = \"%{if var.foo == \"true\"~} big %{~endif}\"\n}\n"),
		},
		{
			Name:     "Heredoc",
			Filename: "test.tf",
			Source:   []byte("locals {\n  foo = <<EOT\n    bar\n  EOT\n}\n"),
			Target:   []byte("locals {\n  foo = <<EOT\n    bar\n  EOT\n}\n"),
		},
		{
			Name:     "HeredocIndented",
			Filename: "test.tf",
			Source:   []byte("locals {\n  foo = <<-EOT\n    bar\n  EOT\n}\n"),
			Target:   []byte("locals {\n  foo = <<-EOT\n    bar\n  EOT\n}\n"),
		},
		{
			Name:     "HeredocInterpolation",
			Filename: "test.tf",
			Source:   []byte("locals {\n  foo = <<-EOT\n    ${var.bar}\n  EOT\n}\n"),
			Target:   []byte("locals {\n  foo = <<-EOT\n    ${var.bar}\n  EOT\n}\n"),
		},
		{
			Name:     "HeredocDirective",
			Filename: "test.tf",
			Source:   []byte("locals {\n  foo = <<-EOT\n    %{if var.foo == \"true\"~} \n      big \n    %{~endif}\n  EOT\n}\n"),
			Target:   []byte("locals {\n  foo = <<-EOT\n    %{if var.foo == \"true\"~} \n      big \n    %{~endif}\n  EOT\n}\n"),
		},
		{
			Name:     "Variable",
			Filename: "test.tf",
			Source:   []byte("locals {\n  foo = var.bar\n}\n"),
			Target:   []byte("locals {\n  foo = var.bar\n}\n"),
		},
		{
			Name:     "VariableExpression",
			Filename: "test.tf",
			Source:   []byte("locals {\n  foo = var.bar + 1\n}\n"),
			Target:   []byte("locals {\n  foo = var.bar + 1\n}\n"),
		},
		{
			Name:     "TernaryExpression",
			Filename: "test.tf",
			Source:   []byte("locals {\n  foo = var.bar ? 1 : 0\n}\n"),
			Target:   []byte("locals {\n  foo = var.bar ? 1 : 0\n}\n"),
		},
		{
			Name:     "TernaryExpressionMultiline",
			Filename: "test.tf",
			Source:   []byte("locals {\n  foo = (\n    var.bar\n    ?\n    1\n    :\n    0\n  )\n}\n"),
			Target:   []byte("locals {\n  foo = (\n    var.bar\n    ?\n    1\n    :\n    0\n  )\n}\n"),
		},
		{
			Name:     "TernaryExpressionComments",
			Filename: "test.tf",
			Source:   []byte("locals {\n  foo = var.bar ? /*test*/ 1 /*test*/ : /*test*/ 0 /*test*/\n}\n"),
			Target:   []byte("locals {\n  foo = var.bar ? /*test*/ 1 /*test*/ : /*test*/ 0 /*test*/\n}\n"),
		},
		{
			Name:     "NestedTernaryExpressions",
			Filename: "test.tf",
			Source:   []byte("locals {\n  result = var.env == \"prod\" ? var.prod_config : var.env == \"staging\" ? var.staging_config : var.env == \"dev\" ? var.dev_config : var.default_config\n}\n"),
			Target:   []byte("locals {\n  result = var.env == \"prod\" ? var.prod_config : var.env == \"staging\" ? var.staging_config : var.env == \"dev\" ? var.dev_config : var.default_config\n}\n"),
		},
		{
			Name:     "ParenthesizedExpression",
			Filename: "test.tf",
			Source:   []byte("locals {\n  foo = (\n    /*test*/\n    2 + 2\n    # test\n  ) + 2\n}\n"),
			Target:   []byte("locals {\n  foo = (\n    /*test*/\n    2 + 2\n    # test\n  ) + 2\n}\n"),
		},
		{
			Name:     "Function",
			Filename: "test.tf",
			Source:   []byte("locals {\n  foo = upper(var.bar)\n}\n"),
			Target:   []byte("locals {\n  foo = upper(var.bar)\n}\n"),
		},
		{
			Name:     "FunctionNoArgs",
			Filename: "test.tf",
			Source:   []byte("locals {\n  foo = uuid()\n}\n"),
			Target:   []byte("locals {\n  foo = uuid()\n}\n"),
		},
		{
			Name:     "FunctionMultiline",
			Filename: "test.tf",
			Source:   []byte("locals {\n  foo = split(\n    var.bar,\n    var.baz\n  )\n}\n"),
			Target:   []byte("locals {\n  foo = split(\n    var.bar,\n    var.baz\n  )\n}\n"),
		},
		{
			Name:     "FunctionEllipsis",
			Filename: "test.tf",
			Source:   []byte("locals {\n  foo = [(min([55, 2453, 2]...) + 5) / 2, 3]\n}\n"),
			Target:   []byte("locals {\n  foo = [(min([55, 2453, 2]...) + 5) / 2, 3]\n}\n"),
		},
		{
			Name:     "FunctionComments",
			Filename: "test.tf",
			Target:   []byte("locals {\n  foo = {\n    bar = baz(\n      1,\n      # test\n      2\n    )\n  }\n}\n"),
			Source:   []byte("locals {\n  foo = {\n    bar = baz(\n      1,\n      # test\n      2\n    )\n  }\n}\n"),
		},
		{
			Name:     "ProviderDefinedFunction",
			Filename: "test.tf",
			Source:   []byte("locals {\n  foo = provider::assert::true(var.bar)\n}\n"),
			Target:   []byte("locals {\n  foo = provider::assert::true(var.bar)\n}\n"),
		},
		{
			Name:     "List",
			Filename: "test.tf",
			Source:   []byte("locals {\n  foo = [1, 2, 3]\n}\n"),
			Target:   []byte("locals {\n  foo = [1, 2, 3]\n}\n"),
		},
		{
			Name:     "ListEmpty",
			Filename: "test.tf",
			Source:   []byte("locals {\n  foo = []\n}\n"),
			Target:   []byte("locals {\n  foo = []\n}\n"),
		},
		{
			Name:     "ListEmptyMultiline",
			Filename: "test.tf",
			Source:   []byte("locals {\n  foo = [\n  ]\n}\n"),
			Target:   []byte("locals {\n  foo = []\n}\n"),
		},
		{
			Name:     "ListMultiline",
			Filename: "test.tf",
			Source:   []byte("locals {\n  foo = [\n    1,\n    2,\n    3,\n  ]\n}\n"),
			Target:   []byte("locals {\n  foo = [\n    1,\n    2,\n    3,\n  ]\n}\n"),
		},
		{
			Name:     "ListForExpression",
			Filename: "test.tf",
			Source:   []byte("locals {\n  map = [for s in var.list : upper(s)]\n}\n"),
			Target:   []byte("locals {\n  map = [for s in var.list : upper(s)]\n}\n"),
		},
		{
			Name:     "ListForExpressionMultiline",
			Filename: "test.tf",
			Source:   []byte("locals {\n  map = [\n    for s in var.list : upper(s)\n  ]\n}\n"),
			Target:   []byte("locals {\n  map = [\n    for s in var.list : upper(s)\n  ]\n}\n"),
		},
		{
			Name:     "ListForExpressionWithComplexCondition",
			Filename: "test.tf",
			Source:   []byte("locals {\n  filtered = [for item in var.list : upper(item) if item != null && length(item) > 0 && !startswith(item, \"_\")]\n}\n"),
			Target:   []byte("locals {\n  filtered = [for item in var.list : upper(item) if item != null && length(item) > 0 && !startswith(item, \"_\")]\n}\n"),
		},
		{
			Name:     "IndexingOperator",
			Filename: "test.tf",
			Source:   []byte("locals {\n  foo = var.bar[\"baz\"]\n}\n"),
			Target:   []byte("locals {\n  foo = var.bar[\"baz\"]\n}\n"),
		},
		{
			Name:     "IndexingOperatorStar",
			Filename: "test.tf",
			Source:   []byte("locals {\n  foo = var.bar[*].baz\n}\n"),
			Target:   []byte("locals {\n  foo = var.bar[*].baz\n}\n"),
		},
		{
			Name:     "Map",
			Filename: "test.tf",
			Source:   []byte("locals {\n  map = {\n    foo = \"bar\"\n    bar = \"baz\"\n  }\n}\n"),
			Target:   []byte("locals {\n  map = {\n    foo = \"bar\"\n    bar = \"baz\"\n  }\n}\n"),
		},
		{
			Name:     "MapEmpty",
			Filename: "test.tf",
			Source:   []byte("locals {\n  map = {}\n}\n"),
			Target:   []byte("locals {\n  map = {}\n}\n"),
		},
		{
			Name:     "MapInline",
			Filename: "test.tf",
			Source:   []byte("locals {\n  map = { foo = \"bar\" }\n}\n"),
			Target:   []byte("locals {\n  map = { foo = \"bar\" }\n}\n"),
		},
		{
			Name:     "MapNumberKeys",
			Filename: "test.tf",
			Source:   []byte("locals {\n  map = { 1 = \"foo\" }\n}\n"),
			Target:   []byte("locals {\n  map = { 1 = \"foo\" }\n}\n"),
		},
		{
			Name:     "MapComments",
			Filename: "test.tf",
			Target:   []byte("locals {\n  map = {\n    a = {\n      b = \"c\"\n\n      # test\n      c = \"d\"\n    }\n  }\n}\n"),
			Source:   []byte("locals {\n  map = {\n    a = {\n      b = \"c\"\n\n      # test\n      c = \"d\"\n    }\n  }\n}\n"),
		},
		{
			Name:     "MapCommentsMultiline",
			Filename: "test.tf",
			Target:   []byte("locals {\n  map = {\n    a /*test*/ = \"b\"\n\n    b = /*test*/ \"c\"\n\n    c = \"d\" /*test*/\n  }\n}\n"),
			Source:   []byte("locals {\n  map = {\n    a /*test*/ = \"b\"\n\n    b = /*test*/ \"c\"\n\n    c = \"d\" /*test*/\n  }\n}\n"),
		},
		{
			Name:     "MapCommentDelimiter",
			Filename: "test.tf",
			Source:   []byte("locals {\n  map = {\n    foo = \"bar\" # test\n    bar = \"baz\"\n  }\n}\n"),
			Target:   []byte("locals {\n  map = {\n    foo = \"bar\" # test\n    bar = \"baz\"\n  }\n}\n"),
		},
		{
			Name:     "MapInlineCommaDelimited",
			Filename: "test.tf",
			Source:   []byte("locals {\n  map = { foo = \"bar\", bar = \"baz\" }\n}\n"),
			Target:   []byte("locals {\n  map = { foo = \"bar\", bar = \"baz\" }\n}\n"),
		},
		{
			Name:     "MapInterpolation",
			Filename: "test.tf",
			Source:   []byte("locals {\n  map = {\n    (var.a) = \"b\"\n  }\n}\n"),
			Target:   []byte("locals {\n  map = {\n    (var.a) = \"b\"\n  }\n}\n"),
		},
		{
			Name:     "MapForExpression",
			Filename: "test.tf",
			Source:   []byte("locals {\n  map = { for s in var.list : s => upper(s) }\n}\n"),
			Target:   []byte("locals {\n  map = { for s in var.list : s => upper(s) }\n}\n"),
		},
		{
			Name:     "MapForExpressionEllipsis",
			Filename: "test.tf",
			Source:   []byte("locals {\n  map = merge({ for s in var.list : s.k => s... }, var.a)\n}\n"),
			Target:   []byte("locals {\n  map = merge({ for s in var.list : s.k => s... }, var.a)\n}\n"),
		},
		{
			Name:     "MapForExpressionMultiline",
			Filename: "test.tf",
			Source:   []byte("locals {\n  map = {\n    for s in var.list :\n    s => upper(s)\n  }\n}\n"),
			Target:   []byte("locals {\n  map = {\n    for s in var.list :\n    s => upper(s)\n  }\n}\n"),
		},
		{
			Name:     "MapForExpressionComments",
			Filename: "test.tf",
			Source:   []byte("locals {\n  map = {\n    # test\n    for k, v in var.map :\n    # test\n    k => upper(v)\n    # test\n  }\n}\n"),
			Target:   []byte("locals {\n  map = {\n    # test\n    for k, v in var.map :\n    # test\n    k => upper(v)\n    # test\n  }\n}\n"),
		},
		{
			Name:     "MapForExpressionNested",
			Filename: "test.tf",
			Source:   []byte("locals {\n  test = merge([for name, subnets in local.extra_public_subnets : {\n    for az, cidr in zipmap(local.azs, subnets) :\n    \"${name}-${az}\" => { name = name, az = az, cidr = cidr }\n  }]...)\n}\n"),
			Target:   []byte("locals {\n  test = merge([for name, subnets in local.extra_public_subnets : {\n    for az, cidr in zipmap(local.azs, subnets) :\n    \"${name}-${az}\" => { name = name, az = az, cidr = cidr }\n  }]...)\n}\n"),
		},
		{
			Name:     "MapForExpressionWithComplexCondition",
			Filename: "test.tf",
			Source:   []byte("locals {\n  filtered = { for item in var.list : item => upper(item) if item != null && length(item) > 0 && !startswith(item, \"_\") }\n}\n"),
			Target:   []byte("locals {\n  filtered = { for item in var.list : item => upper(item) if item != null && length(item) > 0 && !startswith(item, \"_\") }\n}\n"),
		},
		{
			Name:     "Combined",
			Filename: "test.tf",
			Source:   []byte("locals {\n  map = {\n    ident     = var.sth\n    bool      = true\n    str       = \"str\"\n    str-inter = \"str-${lower(\"inter\")}\"\n    num       = 1.24\n    expr      = 3 + 100 / 2\n    func      = join(\"-\", split(\"_\", var.list))\n    heredoc   = <<-EOF\n      hereodc\n    EOF\n    list      = [1, 2, 3]\n    mlist = [\n      1,\n      2,\n      3,\n    ]\n    map = { a = \"b\" }\n    mmap = {\n      a = \"b\"\n    }\n  }\n}\n"),
			Target:   []byte("locals {\n  map = {\n    ident     = var.sth\n    bool      = true\n    str       = \"str\"\n    str-inter = \"str-${lower(\"inter\")}\"\n    num       = 1.24\n    expr      = 3 + 100 / 2\n    func      = join(\"-\", split(\"_\", var.list))\n    heredoc   = <<-EOF\n      hereodc\n    EOF\n    list      = [1, 2, 3]\n    mlist = [\n      1,\n      2,\n      3,\n    ]\n    map = { a = \"b\" }\n    mmap = {\n      a = \"b\"\n    }\n  }\n}\n"),
		},
	}

	for _, test := range cases {
		res, _ := Format(test.Source, test.Filename)
		AssertEqual(t, test.Name, res, test.Target)
	}
}

func TestFormatComments(t *testing.T) {
	cases := []TestCase{
		{
			Name:     "Valid",
			Filename: "test.tf",
			Source:   []byte("# this is a comment\n"),
			Target:   []byte("# this is a comment\n"),
		},
		{
			Name:     "SingleCharComment",
			Filename: "test.tf",
			Source:   []byte("#"),
			Target:   []byte("#\n"),
		},
		{
			Name:     "EmptyCStyleComment",
			Filename: "test.tf",
			Source:   []byte("//"),
			Target:   []byte("#\n"),
		},
		{
			Name:     "DoubleSlash",
			Filename: "test.tf",
			Source:   []byte("// this is a comment\n"),
			Target:   []byte("# this is a comment\n"),
		},
		{
			Name:     "MissingSpace",
			Filename: "test.tf",
			Source:   []byte("#needs space\n"),
			Target:   []byte("# needs space\n"),
		},
		{
			Name:     "MissingSpacePattern",
			Filename: "test.tf",
			Source:   []byte("###############\n## doesn't need space\n###############\n"),
			Target:   []byte("###############\n## doesn't need space\n###############\n"),
		},
		{
			Name:     "MissingSpaceDoubleSlash",
			Filename: "test.tf",
			Source:   []byte("//needs space\n"),
			Target:   []byte("# needs space\n"),
		},
		{
			Name:     "SameLine",
			Filename: "test.tf",
			Source:   []byte("resource \"foo\" \"bar\" {} # comment\n"),
			Target:   []byte("resource \"foo\" \"bar\" {} # comment\n"),
		},
		{
			Name:     "Multiline",
			Filename: "test.tf",
			Source:   []byte("/* multiline comment ignored */\n"),
			Target:   []byte("/* multiline comment ignored */\n"),
		},
	}

	for _, test := range cases {
		res, _ := Format(test.Source, test.Filename)
		AssertEqual(t, test.Name, res, test.Target)
	}
}

func TestFormatNewlines(t *testing.T) {
	cases := []TestCase{
		{
			Name:     "LeadingNewline",
			Filename: "test.tf",
			Source:   []byte("\n# leading newline\n"),
			Target:   []byte("# leading newline\n"),
		},
		{
			Name:     "LeadingNewlines",
			Filename: "test.tf",
			Source:   []byte("\n\n\nlocals {\n  foo = true\n}\n"),
			Target:   []byte("locals {\n  foo = true\n}\n"),
		},
		{
			Name:     "ExtraNewlines",
			Filename: "test.tf",
			Source:   []byte("# lots of newlines\n\nresource \"foo\" \"bar\" {}\n\n"),
			Target:   []byte("# lots of newlines\n\nresource \"foo\" \"bar\" {}\n"),
		},
		{
			Name:     "ExtraNewlinesList",
			Filename: "test.tf",
			Source:   []byte("locals {\n  foo = [\n\n    1,\n\n  ]\n}\n"),
			Target:   []byte("locals {\n  foo = [\n    1,\n  ]\n}\n"),
		},
		{
			Name:     "ExtraNewlinesFunction",
			Filename: "test.tf",
			Source:   []byte("locals {\n  foo = upper(\n\n    \"hello\"\n\n  )\n}\n"),
			Target:   []byte("locals {\n  foo = upper(\n    \"hello\"\n  )\n}\n"),
		},
		{
			Name:     "ManyExtraNewlines",
			Filename: "test.tf",
			Source:   []byte("# lots of newlines\n\n\n\nresource \"foo\" \"bar\" {}\n"),
			Target:   []byte("# lots of newlines\n\nresource \"foo\" \"bar\" {}\n"),
		},
		{
			Name:     "BlockTrailingNewline",
			Filename: "test.tf",
			Source:   []byte("resource \"foo\" \"bar\" {\n  # test\n\n}\n"),
			Target:   []byte("resource \"foo\" \"bar\" {\n  # test\n}\n"),
		},
		{
			Name:     "BlockLeadingNewline",
			Filename: "test.tf",
			Source:   []byte("resource \"foo\" \"bar\" {\n\n  # test\n}\n"),
			Target:   []byte("resource \"foo\" \"bar\" {\n  # test\n}\n"),
		},
		{
			Name:     "TopLevelBlockMissingNewline",
			Filename: "test.tf",
			Source:   []byte("resource \"foo\" \"bar\" {}\nresource \"bar\" \"baz\" {}\n"),
			Target:   []byte("resource \"foo\" \"bar\" {}\n\nresource \"bar\" \"baz\" {}\n"),
		},
		{
			Name:     "TopLevelBlockMissingNewlineComment",
			Filename: "test.tf",
			Source:   []byte("resource \"foo\" \"bar\" {}\n# test\nresource \"bar\" \"baz\" {}\n"),
			Target:   []byte("resource \"foo\" \"bar\" {}\n\n# test\nresource \"bar\" \"baz\" {}\n"),
		},
		{
			Name:     "TopLevelBlockMissingNewlineInlineComment",
			Filename: "test.tf",
			Source:   []byte("resource \"foo\" \"bar\" {} # test\nresource \"bar\" \"baz\" {}\n"),
			Target:   []byte("resource \"foo\" \"bar\" {} # test\n\nresource \"bar\" \"baz\" {}\n"),
		},
		{
			Name:     "TopLevelBlockCommentsBetween",
			Filename: "test.tf",
			Source:   []byte("resource \"foo\" \"bar\" {}\n\n# test\n\nresource \"bar\" \"baz\" {}\n"),
			Target:   []byte("resource \"foo\" \"bar\" {}\n\n# test\n\nresource \"bar\" \"baz\" {}\n"),
		},
		{
			Name:     "MissingTrailingNewline",
			Filename: "test.tf",
			Source:   []byte("resource \"foo\" \"bar\" {}"),
			Target:   []byte("resource \"foo\" \"bar\" {}\n"),
		},
		{
			Name:     "MissingTrailingNewlineComment",
			Filename: "test.tf",
			Source:   []byte("# this also needs a trailing newline"),
			Target:   []byte("# this also needs a trailing newline\n"),
		},
		// false positive specific to how adding newlines between top-level blocks is implemented
		{
			Name:     "TfvarsMap",
			Filename: "test.tfvars",
			Source:   []byte("a = { foo = \"bar\" }\nb = 5"),
			Target:   []byte("a = { foo = \"bar\" }\nb = 5"),
		},
	}

	for _, test := range cases {
		res, _ := Format(test.Source, test.Filename)
		AssertEqual(t, test.Name, res, test.Target)
	}
}

func TestFormatParens(t *testing.T) {
	cases := []TestCase{
		{
			Name:     "RedundantParens",
			Filename: "test.tf",
			Source:   []byte("locals {\n  map = ({})\n}\n"),
			Target:   []byte("locals {\n  map = {}\n}\n"),
		},
		{
			Name:     "RedundantParensFunction",
			Filename: "test.tf",
			Source:   []byte("locals {\n  foo = upper((var.bar))\n}\n"),
			Target:   []byte("locals {\n  foo = upper(var.bar)\n}\n"),
		},
		{
			Name:     "RedundantParensFunctionMultiline",
			Filename: "test.tf",
			Source:   []byte("locals {\n  foo = upper(\n    (var.bar)\n  )\n}\n"),
			Target:   []byte("locals {\n  foo = upper(\n    var.bar\n  )\n}\n"),
		},
		{
			Name:     "RedundantParensFunctionArg",
			Filename: "test.tf",
			Source:   []byte("locals {\n  foo = split(\"/\", (var.bar))\n}\n"),
			Target:   []byte("locals {\n  foo = split(\"/\", var.bar)\n}\n"),
		},
		{
			Name:     "RedundantParensList",
			Filename: "test.tf",
			Source:   []byte("locals {\n  foo = [(var.bar)]\n}\n"),
			Target:   []byte("locals {\n  foo = [var.bar]\n}\n"),
		},
		{
			Name:     "RedundantParensMap",
			Filename: "test.tf",
			Source:   []byte("locals {\n  map = {\n    foo = (var.bar)\n    bar = var.baz\n  }\n}\n"),
			Target:   []byte("locals {\n  map = {\n    foo = var.bar\n    bar = var.baz\n  }\n}\n"),
		},
		{
			Name:     "NonRedundantParensMap",
			Filename: "test.tf",
			Source:   []byte("locals {\n  map = {\n    foo = (\n      2 + 2\n    )\n    bar = var.baz\n  }\n}\n"),
			Target:   []byte("locals {\n  map = {\n    foo = (\n      2 + 2\n    )\n    bar = var.baz\n  }\n}\n"),
		},
		{
			Name:     "NonRedundantParensTernary",
			Filename: "test.tf",
			Source:   []byte("locals {\n  foo = var.bar ? var.baz : (2 + 4) % 3\n}\n"),
			Target:   []byte("locals {\n  foo = var.bar ? var.baz : (2 + 4) % 3\n}\n"),
		},
		{
			Name:     "NonRedundantParens",
			Filename: "test.tf",
			Source:   []byte("locals {\n  foo = (2 + 2) / 2\n}\n"),
			Target:   []byte("locals {\n  foo = (2 + 2) / 2\n}\n"),
		},
		{
			Name:     "NonRedundantParensMultiline",
			Filename: "test.tf",
			Source:   []byte("locals {\n  foo = (2\n  + 2)\n}\n"),
			Target:   []byte("locals {\n  foo = (2\n  + 2)\n}\n"),
		},
		{
			Name:     "RedundantNonRedundant",
			Filename: "test.tf",
			Source:   []byte("locals {\n  res = func((var.a + (var.b * 2)))\n}\n"),
			Target:   []byte("locals {\n  res = func(var.a + (var.b * 2))\n}\n"),
		},
	}

	for _, test := range cases {
		res, _ := Format(test.Source, test.Filename)
		AssertEqual(t, test.Name, res, test.Target)
	}
}

func TestFormatLists(t *testing.T) {
	cases := []TestCase{
		{
			Name:     "TrailingComma",
			Filename: "test.tf",
			Source:   []byte("locals {\n  list = [\n    1,\n    2,\n    3\n  ]\n}\n"),
			Target:   []byte("locals {\n  list = [\n    1,\n    2,\n    3,\n  ]\n}\n"),
		},
		{
			// heredoc closing token needs to be followed by a newline so we shouldn't add a trailing comma
			Name:     "TrailingCommaHeredoc",
			Filename: "test.tf",
			Source:   []byte("locals {\n  list = [\n    <<EOF\n    test\n    EOF\n  ]\n}\n"),
			Target:   []byte("locals {\n  list = [\n    <<EOF\n    test\n    EOF\n  ]\n}\n"),
		},
		{
			Name:     "TrailingCommaCbrack",
			Filename: "test.tf",
			Source:   []byte("locals {\n  list = [\n    1,\n    2,\n    3 ]\n}\n"),
			Target:   []byte("locals {\n  list = [\n    1,\n    2,\n    3,\n  ]\n}\n"),
		},
		{
			Name:     "TrailingCommaInline",
			Filename: "test.tf",
			Source:   []byte("locals {\n  list = [1, 2, 3,]\n}\n"),
			Target:   []byte("locals {\n  list = [1, 2, 3]\n}\n"),
		},
		{
			Name:     "TrailingCommaComments",
			Filename: "test.tf",
			Source:   []byte("locals {\n  foo = [/*\n    */ 1 # test\n  ]\n}\n"),
			Target:   []byte("locals {\n  foo = [/*\n    */ 1, # test\n  ]\n}\n"),
		},
		{
			Name:     "TrailingCommaMultilineComments",
			Filename: "test.tf",
			Source:   []byte("locals {\n  foo = [/*\n    */ 1 # test\n    , /*\n    */ 2\n  ]\n}\n"),
			Target:   []byte("locals {\n  foo = [/*\n    */ 1, # test\n    /*\n    */ 2,\n  ]\n}\n"),
		},
		{
			Name:     "EmptyMultiline",
			Filename: "test.tf",
			Source:   []byte("locals {\n  foo = [\n  ]\n}\n"),
			Target:   []byte("locals {\n  foo = []\n}\n"),
		},
		{
			Name:     "EmptyMultilineComments",
			Filename: "test.tf",
			Source:   []byte("locals {\n  foo = [\n    # testing\n  ]\n}\n"),
			Target:   []byte("locals {\n  foo = [\n    # testing\n  ]\n}\n"),
		},
		{
			Name:     "BracketInlineElements",
			Filename: "test.tf",
			Source:   []byte("locals {\n  list = [1,\n  2]\n}\n"),
			Target:   []byte("locals {\n  list = [\n    1,\n    2,\n  ]\n}\n"),
		},
		{
			Name:     "BracketInlineElementsTrailingComma",
			Filename: "test.tf",
			Source:   []byte("locals {\n  list = [1,\n  2,]\n}\n"),
			Target:   []byte("locals {\n  list = [\n    1,\n    2,\n  ]\n}\n"),
		},
		{
			Name:     "UselessNewlines",
			Filename: "test.tf",
			Source:   []byte("locals {\n  list = [\n    1\n\n    ,\n    2\n    , 3,\n\n    4,\n\n\n\n    5\n  , ]\n}\n"),
			Target:   []byte("locals {\n  list = [\n    1,\n    2,\n    3,\n    4,\n    5,\n  ]\n}\n"),
		},
		{
			// heredoc closing token needs to be followed by a newline so we shouldn't hoist up the comma following it
			Name:     "UselessNewlinesHeredoc",
			Filename: "test.tf",
			Source:   []byte("locals {\n  list = [\n    <<EOF\n    test\n    EOF\n    ,\n    \"test\",\n  ]\n}\n"),
			Target:   []byte("locals {\n  list = [\n    <<EOF\n    test\n    EOF\n    ,\n    \"test\",\n  ]\n}\n"),
		},
		{
			Name:     "UselessNewlinesComments",
			Filename: "test.tf",
			Source:   []byte("locals {\n  list = [\n    1\n    # test\n    ,\n    2 # test\n    , 3, # test\n    # test\n    4\n    , # test\n    5,\n  ]\n}\n"),
			Target:   []byte("locals {\n  list = [\n    1,\n    # test\n    2, # test\n    3, # test\n    # test\n    4,\n    # test\n    5,\n  ]\n}\n"),
		},
	}

	for _, test := range cases {
		res, _ := Format(test.Source, test.Filename)
		AssertEqual(t, test.Name, res, test.Target)
	}
}

func TestFormatMaps(t *testing.T) {
	cases := []TestCase{
		{
			Name:     "ColonOverEquals",
			Filename: "test.tf",
			Source:   []byte("locals {\n  map = {\n    a : \"b\"\n  }\n}\n"),
			Target:   []byte("locals {\n  map = {\n    a = \"b\"\n  }\n}\n"),
		},
		{
			Name:     "ColonOverEqualsNestedBlock",
			Filename: "test.tf",
			Source:   []byte("resource \"foo\" \"bar\" {\n  baz {\n    map = {\n      a : \"b\"\n    }\n  }\n}\n"),
			Target:   []byte("resource \"foo\" \"bar\" {\n  baz {\n    map = {\n      a = \"b\"\n    }\n  }\n}\n"),
		},
		{
			Name:     "ColonOverEqualsNestedMap",
			Filename: "test.tf",
			Source:   []byte("locals {\n  map = {\n    a = {\n      b : \"c\"\n    }\n  }\n}\n"),
			Target:   []byte("locals {\n  map = {\n    a = {\n      b = \"c\"\n    }\n  }\n}\n"),
		},
		{
			Name:     "UselessQuotes",
			Filename: "test.tf",
			Source:   []byte("locals {\n  map = {\n    \"a_1\" = \"b\"\n    \"b-2\" = \"c\"\n  }\n}\n"),
			Target:   []byte("locals {\n  map = {\n    a_1 = \"b\"\n    b-2 = \"c\"\n  }\n}\n"),
		},
		{
			Name:     "UselessQuotesInline",
			Filename: "test.tf",
			Source:   []byte("locals {\n  map = { \"a\" = true, \"b\" = false }\n}\n"),
			Target:   []byte("locals {\n  map = { a = true, b = false }\n}\n"),
		},
		{
			Name:     "UsefulQuotes",
			Filename: "test.tf",
			Source:   []byte("locals {\n  map = {\n    \"eks.amazonaws.com/role-arn\" = \"...\"\n  }\n}\n"),
			Target:   []byte("locals {\n  map = {\n    \"eks.amazonaws.com/role-arn\" = \"...\"\n  }\n}\n"),
		},
		{
			Name:     "UselessComma",
			Filename: "test.tf",
			Source:   []byte("locals {\n  map = {\n    a = \"b\",\n    b = \"c\",\n  }\n}\n"),
			Target:   []byte("locals {\n  map = {\n    a = \"b\"\n    b = \"c\"\n  }\n}\n"),
		},
		{
			Name:     "UselessCommaInlineComment",
			Filename: "test.tf",
			Source:   []byte("locals {\n  map = {\n    a = \"b\" /*test*/,\n    b = \"c\",\n  }\n}\n"),
			Target:   []byte("locals {\n  map = {\n    a = \"b\" /*test*/\n    b = \"c\"\n  }\n}\n"),
		},
		{
			Name:     "UselessInterpolation",
			Filename: "test.tf",
			Source:   []byte("locals {\n  map = {\n    \"${var.a}\" = \"b\"\n  }\n}\n"),
			Target:   []byte("locals {\n  map = {\n    (var.a) = \"b\"\n  }\n}\n"),
		},
		{
			Name:     "UsefulInterpolation",
			Filename: "test.tf",
			Source:   []byte("locals {\n  map = {\n    \"${var.a}-${var.b}\" = \"b\"\n    \"a-${var.a}\"        = \"b\"\n  }\n}\n"),
			Target:   []byte("locals {\n  map = {\n    \"${var.a}-${var.b}\" = \"b\"\n    \"a-${var.a}\"        = \"b\"\n  }\n}\n"),
		},
		{
			Name:     "MapBareExpressionWrapping",
			Filename: "test.tf",
			Source:   []byte("locals {\n  map = {\n    1 + var.a + 3 = \"b\"\n  }\n}\n"),
			Target:   []byte("locals {\n  map = {\n    (1 + var.a + 3) = \"b\"\n  }\n}\n"),
		},
		{
			Name:     "MapBareExpressionIdentWrapping",
			Filename: "test.tf",
			Source:   []byte("locals {\n  map = {\n    var.a + 3 = \"b\"\n  }\n}\n"),
			Target:   []byte("locals {\n  map = {\n    (var.a + 3) = \"b\"\n  }\n}\n"),
		},
		{
			Name:     "MapBareExpressionProvider",
			Filename: "test.tf",
			Source:   []byte("module \"foo\" {\n  source = \"./foo\"\n\n  providers = {\n    bar.baz = bar.baz\n  }\n}\n"),
			Target:   []byte("module \"foo\" {\n  source = \"./foo\"\n\n  providers = {\n    bar.baz = bar.baz\n  }\n}\n"),
		},
		{
			Name:     "MapBareExpressionProviderFalsePositive",
			Filename: "test.tf",
			Source:   []byte("locals {\n  providers = {\n    var.a + 3 = \"b\"\n  }\n}\n"),
			Target:   []byte("locals {\n  providers = {\n    (var.a + 3) = \"b\"\n  }\n}\n"),
		},
		{
			Name:     "EmptyMultiline",
			Filename: "test.tf",
			Source:   []byte("locals {\n  foo = {\n  }\n}\n"),
			Target:   []byte("locals {\n  foo = {}\n}\n"),
		},
		{
			Name:     "EmptyMultilineComments",
			Filename: "test.tf",
			Source:   []byte("locals {\n  foo = {\n    # testing\n  }\n}\n"),
			Target:   []byte("locals {\n  foo = {\n    # testing\n  }\n}\n"),
		},
		{
			Name:     "BraceInlineElements",
			Filename: "test.tf",
			Source:   []byte("locals {\n  map = { foo = \"bar\"\n  bar = \"baz\" }\n}\n"),
			Target:   []byte("locals {\n  map = {\n    foo = \"bar\"\n    bar = \"baz\"\n  }\n}\n"),
		},
		{
			Name:     "BraceInlineElementsMultilineComments",
			Filename: "test.tf",
			Source:   []byte("locals {\n  map = { /*test*/ foo = \"bar\"\n  bar = \"baz\" /*test*/ }\n}\n"),
			Target:   []byte("locals {\n  map = { /*test*/\n    foo = \"bar\"\n    bar = \"baz\" /*test*/\n  }\n}\n"),
		},
		{
			Name:     "BraceInlineElementsTrailingComma",
			Filename: "test.tf",
			Source:   []byte("locals {\n  map = { foo = \"bar\",\n  bar = \"baz\", }\n}\n"),
			Target:   []byte("locals {\n  map = {\n    foo = \"bar\"\n    bar = \"baz\"\n  }\n}\n"),
		},
		{
			Name:     "MultilineWithMultipleElementsPerLineCommaTerminated",
			Filename: "test.tf",
			Source:   []byte("locals {\n  map = { foo = \"bar\", bar = \"baz\",\n  baz = \"spam\" }\n}\n"),
			Target:   []byte("locals {\n  map = {\n    foo = \"bar\"\n    bar = \"baz\"\n    baz = \"spam\"\n  }\n}\n"),
		},
		{
			Name:     "MultilineWithMultipleElementsPerLineMixedTerminators",
			Filename: "test.tf",
			Source:   []byte("locals {\n  map = { foo = \"bar\", bar = \"baz\"\n  baz = \"spam\" }\n}\n"),
			Target:   []byte("locals {\n  map = {\n    foo = \"bar\"\n    bar = \"baz\"\n    baz = \"spam\"\n  }\n}\n"),
		},
		{
			Name:     "UselessNewlines",
			Filename: "test.tf",
			Source:   []byte("locals {\n  map = {\n\n    foo = \"bar\"\n\n\n    bar = \"baz\"\n\n  }\n}\n"),
			Target:   []byte("locals {\n  map = {\n    foo = \"bar\"\n\n    bar = \"baz\"\n  }\n}\n"),
		},
		{
			Name:     "UselessNewlinesComments",
			Filename: "test.tf",
			Source:   []byte("locals {\n  map = {\n\n    foo = \"bar\"\n\n\n    # test\n    bar = \"baz\"\n\n  }\n}\n"),
			Target:   []byte("locals {\n  map = {\n    foo = \"bar\"\n\n    # test\n    bar = \"baz\"\n  }\n}\n"),
		},
	}

	for _, test := range cases {
		res, _ := Format(test.Source, test.Filename)
		AssertEqual(t, test.Name, res, test.Target)
	}
}

func TestFormatForExpressions(t *testing.T) {
	cases := []TestCase{
		{
			Name:     "MapKeyInterpolationUnwrapping",
			Filename: "test.tf",
			Source:   []byte("locals {\n  map = { for s in var.list : \"${s}\" => upper(s) }\n}\n"),
			Target:   []byte("locals {\n  map = { for s in var.list : s => upper(s) }\n}\n"),
		},
		{
			Name:     "MapKeyInterpolationUnwrappingComment",
			Filename: "test.tf",
			Source:   []byte("locals {\n  map = { for s in var.list : \"${s}\" /*test*/ => upper(s) }\n}\n"),
			Target:   []byte("locals {\n  map = { for s in var.list : s /*test*/ => upper(s) }\n}\n"),
		},
		{
			Name:     "MapKeyInterpolationUnwrappingMultiline",
			Filename: "test.tf",
			Source:   []byte("locals {\n  map = { for s in var.list : \"${\n    s\n  }\" => upper(s) }\n}\n"),
			Target:   []byte("locals {\n  map = { for s in var.list : s => upper(s) }\n}\n"),
		},
		{
			Name:     "MapKeyInterpolationUnwrappingFalsePositive",
			Filename: "test.tf",
			Source:   []byte("locals {\n  map = { for s in var.list : \"${s}-${s}\" => upper(s) }\n}\n"),
			Target:   []byte("locals {\n  map = { for s in var.list : \"${s}-${s}\" => upper(s) }\n}\n"),
		},
		{
			Name:     "SeparateLines",
			Filename: "test.tf",
			Source:   []byte("locals {\n  map = {\n    for\n    s\n    in\n    var.list\n    :\n    s\n    =>\n    upper(s)\n  }\n}\n"),
			Target:   []byte("locals {\n  map = {\n    for s in var.list :\n    s => upper(s)\n  }\n}\n"),
		},
		{
			Name:     "SeparateLinesComments",
			Filename: "test.tf",
			Source:   []byte("locals {\n  map = {\n    for\n    # test\n    s\n    # test\n    in\n    # test\n    var.list\n    # test\n    :\n    # test\n    s\n    # test\n    =>\n    # test\n    upper(s)\n  }\n}\n"),
			Target:   []byte("locals {\n  map = {\n    for\n    # test\n    s\n    # test\n    in\n    # test\n    var.list\n    # test\n    :\n    # test\n    s\n    # test\n    =>\n    # test\n    upper(s)\n  }\n}\n"),
		},
		{
			Name:     "WrongSplit",
			Filename: "test.tf",
			Source:   []byte("locals {\n  map = {\n    for s in var.list : s\n    => upper(s)\n  }\n}\n"),
			Target:   []byte("locals {\n  map = {\n    for s in var.list : s => upper(s)\n  }\n}\n"),
		},
		{
			Name:     "UselessNewlines",
			Filename: "test.tf",
			Source:   []byte("locals {\n  map = {\n    # testing\n\n    for s in var.list :\n\n    s => upper(s)\n  }\n}\n"),
			Target:   []byte("locals {\n  map = {\n    # testing\n    for s in var.list :\n    s => upper(s)\n  }\n}\n"),
		},
	}

	for _, test := range cases {
		res, _ := Format(test.Source, test.Filename)
		AssertEqual(t, test.Name, res, test.Target)
	}
}

func TestFormatFunctions(t *testing.T) {
	cases := []TestCase{
		{
			Name:     "TrailingComma",
			Filename: "test.tf",
			Source:   []byte("locals {\n  foo = bar(1, 2,)\n}\n"),
			Target:   []byte("locals {\n  foo = bar(1, 2)\n}\n"),
		},
		{
			Name:     "EmptyMultiline",
			Filename: "test.tf",
			Source:   []byte("locals {\n  foo = bar(\n  )\n}\n"),
			Target:   []byte("locals {\n  foo = bar()\n}\n"),
		},
		{
			Name:     "EmptyMultilineComments",
			Filename: "test.tf",
			Source:   []byte("locals {\n  foo = bar(\n    # testing\n  )\n}\n"),
			Target:   []byte("locals {\n  foo = bar(\n    # testing\n  )\n}\n"),
		},
		{
			Name:     "ParenInlineArgument",
			Filename: "test.tf",
			Source:   []byte("locals {\n  foo = bar(1,\n  2)\n}\n"),
			Target:   []byte("locals {\n  foo = bar(\n    1,\n    2\n  )\n}\n"),
		},
		{
			Name:     "ParenInlineArgumentTrailingComma",
			Filename: "test.tf",
			Source:   []byte("locals {\n  foo = bar(1,\n  2,)\n}\n"),
			Target:   []byte("locals {\n  foo = bar(\n    1,\n    2\n  )\n}\n"),
		},
		{
			Name:     "UselessNewlines",
			Filename: "test.tf",
			Source:   []byte("locals {\n  foo = bar(\n    1\n\n    ,\n    2\n    , 3,\n\n    4,\n\n\n\n    5\n  , )\n}\n"),
			Target:   []byte("locals {\n  foo = bar(\n    1,\n    2,\n    3,\n    4,\n    5\n  )\n}\n"),
		},
		{
			// heredoc closing token needs to be followed by a newline so we shouldn't hoist up the comma following it
			Name:     "UselessNewlinesHeredoc",
			Filename: "test.tf",
			Source:   []byte("locals {\n  foo = bar(\n    <<EOF\n    test\n    EOF\n    ,\n    \"test\"\n  )\n}\n"),
			Target:   []byte("locals {\n  foo = bar(\n    <<EOF\n    test\n    EOF\n    ,\n    \"test\"\n  )\n}\n"),
		},
		{
			Name:     "UselessNewlinesComments",
			Filename: "test.tf",
			Source:   []byte("locals {\n  foo = bar(\n    1\n    # test\n    ,\n    2 # test\n    , 3, # test\n    # test\n    4\n    , # test\n    5,\n  )\n}\n"),
			Target:   []byte("locals {\n  foo = bar(\n    1,\n    # test\n    2, # test\n    3, # test\n    # test\n    4,\n    # test\n    5\n  )\n}\n"),
		},
		{
			Name:     "UselessNewlinesComments",
			Filename: "test.tf",
			Source:   []byte("locals {\n  foo = bar(\n    1\n    # test\n    ,\n    2 # test\n    , 3, # test\n    # test\n    4\n    , # test\n    5,\n  )\n}\n"),
			Target:   []byte("locals {\n  foo = bar(\n    1,\n    # test\n    2, # test\n    3, # test\n    # test\n    4,\n    # test\n    5\n  )\n}\n"),
		},
	}

	for _, test := range cases {
		res, _ := Format(test.Source, test.Filename)
		AssertEqual(t, test.Name, res, test.Target)
	}
}
