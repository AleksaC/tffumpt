package format

import (
	"bytes"
	"log"
	"slices"
	"strings"
	"unicode"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"

	tffmt "github.com/AleksaC/tffumpt/fmt"
)

func Format(src []byte, filename string) ([]byte, hcl.Diagnostics) {
	f, diags := tffmt.FormatSourceCode(src, filename)
	if diags.HasErrors() {
		return nil, diags
	}

	for _, block := range f.Body().Blocks() {
		formatExpressions(block)
	}

	// hclwrite AST doesn't expose comments and newlines between blocks so we need to handle them separately
	res := formatCommentsAndNewlines(f.Bytes(), filename)

	return res, diags
}

type Context struct {
	attributeName string
	blockType     string
}

func formatExpressions(b *hclwrite.Block) {
	body := b.Body()
	for name, attr := range body.Attributes() {
		ctx := Context{attributeName: name, blockType: b.Type()}
		formattedAttr, _ := formatExpression(attr.Expr().BuildTokens(nil), 0, ctx, true)
		body.SetAttributeRaw(name, formattedAttr)
	}
	for _, block := range body.Blocks() {
		formatExpressions(block)
	}
}

func formatExpression(tokens hclwrite.Tokens, pos int, ctx Context, continueOnNewline bool) (hclwrite.Tokens, int) {
	var res, formattedTokens hclwrite.Tokens

	for pos < len(tokens) {
		//nolint:exhaustive // default case handles all the other token types
		switch tokens[pos].Type {
		// :: as used in provider-defined functions is a separate token, not two colon tokens
		case hclsyntax.TokenCBrace, hclsyntax.TokenCBrack, hclsyntax.TokenCParen, hclsyntax.TokenTemplateSeqEnd,
			hclsyntax.TokenComma, hclsyntax.TokenColon, hclsyntax.TokenEqual, hclsyntax.TokenFatArrow:
			return formattedTokens, pos
		case hclsyntax.TokenComment, hclsyntax.TokenNewline:
			if !continueOnNewline &&
				(tokens[pos].Type == hclsyntax.TokenNewline ||
					isSingleLineComment(tokens[pos].Bytes)) {
				return formattedTokens, pos
			}

			// TODO: likely needs to handle cases when there are comments
			// we don't need to check if continueOnNewline is true, because if it wasn't
			// the function would have already returned
			if tokens[pos].Type == hclsyntax.TokenNewline {
				// TODO: bounds check ?
				if tokens[pos+1].Type == hclsyntax.TokenNewline {
					pos++
					continue
				}
			}

			res, pos = tokens[pos:pos+1], pos+1
		case hclsyntax.TokenOParen:
			res, pos = formatParens(tokens, pos, ctx)
		case hclsyntax.TokenQuestion:
			// ternary operator
			res = hclwrite.Tokens{tokens[pos]}

			var expr1, expr2 hclwrite.Tokens
			expr1, pos = formatExpression(tokens, pos+1, ctx, continueOnNewline)
			delim := tokens[pos]
			expr2, pos = formatExpression(tokens, pos+1, ctx, continueOnNewline)

			if delim.Type != hclsyntax.TokenColon {
				// this should never happen if the parser is implemented correctly
				log.Fatalf("Expected ':', got `%s`", tokens[pos].Bytes)
			}

			res = append(res, expr1...)
			res = append(res, delim)
			res = append(res, expr2...)
		case hclsyntax.TokenOQuote:
			res, pos = formatString(tokens, pos, ctx)
		case hclsyntax.TokenOHeredoc:
			res, pos = formatHeredoc(tokens, pos, ctx)
		case hclsyntax.TokenIdent:
			if pos+1 < len(tokens) && tokens[pos+1].Type == hclsyntax.TokenOParen {
				res, pos = formatFunction(tokens, pos, ctx)
				break
			}
			res, pos = tokens[pos:pos+1], pos+1
		case hclsyntax.TokenOBrack:
			// check if it's indexing operator instead of list literal
			if pos > 0 && (tokens[pos-1].Type == hclsyntax.TokenIdent || tokens[pos-1].Type == hclsyntax.TokenCParen ||
				tokens[pos-1].Type == hclsyntax.TokenCBrace || tokens[pos-1].Type == hclsyntax.TokenCBrack) {
				oBracketPos := pos
				res, pos = formatExpression(tokens, pos+1, ctx, true)
				res = append(hclwrite.Tokens{tokens[oBracketPos]}, res...)
				res = append(res, tokens[pos])
				pos++
				break
			}
			if resFor, posFor := formatForExpression(tokens, pos, ctx); resFor != nil {
				res, pos = resFor, posFor
				break
			}
			res, pos = formatList(tokens, pos, ctx)
		case hclsyntax.TokenOBrace:
			if resFor, posFor := formatForExpression(tokens, pos, ctx); resFor != nil {
				res, pos = resFor, posFor
				break
			}
			res, pos = formatMap(tokens, pos, ctx)
		default:
			res, pos = tokens[pos:pos+1], pos+1
		}

		formattedTokens = append(formattedTokens, res...)
	}

	return formattedTokens, pos
}

func formatString(tokens hclwrite.Tokens, pos int, ctx Context) (hclwrite.Tokens, int) {
	var res, formattedTokens hclwrite.Tokens

	for {
		if tokens[pos].Type == hclsyntax.TokenCQuote {
			formattedTokens = append(formattedTokens, tokens[pos])
			pos++
			break
		}

		//nolint:exhaustive // default case handles all the other token types
		switch tokens[pos].Type {
		case hclsyntax.TokenTemplateInterp:
			formattedTokens = append(formattedTokens, tokens[pos])
			res, pos = formatExpression(tokens, pos+1, ctx, true)
			// expression doesn't include template closing
			res = append(res, tokens[pos])
		case hclsyntax.TokenTemplateControl:
			formattedTokens = append(formattedTokens, tokens[pos])
			res, pos = formatExpression(tokens, pos+1, ctx, true)
			// expression doesn't include template closing
			res = append(res, tokens[pos])
		default:
			res = tokens[pos : pos+1]
		}

		formattedTokens = append(formattedTokens, res...)
		pos++
	}

	return formattedTokens, pos
}

func formatHeredoc(tokens hclwrite.Tokens, pos int, ctx Context) (hclwrite.Tokens, int) {
	var res, formattedTokens hclwrite.Tokens

	for {
		if tokens[pos].Type == hclsyntax.TokenCHeredoc {
			formattedTokens = append(formattedTokens, tokens[pos])
			pos++
			break
		}

		//nolint:exhaustive // default case handles all the other token types
		switch tokens[pos].Type {
		case hclsyntax.TokenTemplateInterp:
			formattedTokens = append(formattedTokens, tokens[pos])
			res, pos = formatExpression(tokens, pos+1, ctx, true)
			// expression doesn't include template closing
			res = append(res, tokens[pos])
		case hclsyntax.TokenTemplateControl:
			formattedTokens = append(formattedTokens, tokens[pos])
			res, pos = formatExpression(tokens, pos+1, ctx, false)
			// expression doesn't include template closing
			res = append(res, tokens[pos])
		default:
			res = tokens[pos : pos+1]
		}

		formattedTokens = append(formattedTokens, res...)
		pos++
	}

	return formattedTokens, pos
}

func formatFunction(tokens hclwrite.Tokens, pos int, ctx Context) (hclwrite.Tokens, int) {
	var res, formattedTokens hclwrite.Tokens
	isMultiLine := false

	// we assume it's a proper function, which always starts with TokenIdent TokenOParen sequence
	// in case of provider-defined functions the part before the function itself is parsed separately
	formattedTokens = append(formattedTokens, tokens[pos:pos+2]...)
	pos += 2

	for {
		if tokens[pos].Type == hclsyntax.TokenCParen {
			formattedTokens = append(formattedTokens, tokens[pos])
			pos++
			break
		}

		prevPos := pos - 1
		res, pos = formatExpression(tokens, pos, ctx, true)

		isMultiLine = isMultiLine || isNewline(res[0]) || isNewline(res[len(res)-1])

		delimiter := tokens[pos]

		// in case we have something like `(<element>` in a multiline function,
		// we move the element to a separate line by prepending the newline
		if tokens[prevPos].Type == hclsyntax.TokenOParen && res[0].Type != hclsyntax.TokenComment &&
			(isMultiLine || (delimiter.Type == hclsyntax.TokenComma && isNewline(tokens[pos+1]))) {
			res = append(hclwrite.Tokens{
				&hclwrite.Token{
					Type:         hclsyntax.TokenNewline,
					Bytes:        []byte{'\n'},
					SpacesBefore: 0,
				},
			}, res...)
			isMultiLine = true
		}

		if delimiter.Type == hclsyntax.TokenCParen {
			if isMultiLine {
				i := findLast(res, len(res)-1)

				if i >= 0 {
					// <element>), in a multiline function move paren to a separate line
					if i == len(res)-1 {
						res = append(res, &hclwrite.Token{
							Type:         hclsyntax.TokenNewline,
							Bytes:        []byte{'\n'},
							SpacesBefore: 0,
						})
					}
				} else {
					// this can only happen in an empty function, res only contains newlines and comments
					onlyNewlines := true
					for _, token := range res {
						if token.Type != hclsyntax.TokenNewline {
							onlyNewlines = false
							break
						}
					}
					// if there are no comments we collapse the function to a single line
					if onlyNewlines {
						res = hclwrite.Tokens{}
					}
				}
			}

			formattedTokens = append(formattedTokens, res...)
			formattedTokens = append(formattedTokens, delimiter)
			pos++
			break // ) is the delimiter, we don't need to continue the loop
		} else if delimiter.Type == hclsyntax.TokenComma {
			// Look ahead to find the next non-newline, non-comment token
			nextTokenIndex := findFirst(tokens, pos+1)

			isTrailingComma := tokens[nextTokenIndex].Type == hclsyntax.TokenCParen

			// when there is a comma on its own line, we move it up next to the
			// corresponding element if it's not a trailing comma
			isFloatingComma := isNewline(res[len(res)-1])
			if isFloatingComma && !isTrailingComma {
				i := findLast(res, len(res)-1)
				res = append(res[:i+1], append(hclwrite.Tokens{delimiter}, res[i+1:]...)...)
			}

			formattedTokens = append(formattedTokens, res...)

			nextToken := tokens[pos+1]
			// <element>,)
			if isTrailingComma {
				// remove trailing comma
				if isMultiLine {
					if !isFloatingComma {
						formattedTokens = append(formattedTokens, &hclwrite.Token{
							Type:         hclsyntax.TokenNewline,
							Bytes:        []byte{'\n'},
							SpacesBefore: 0,
						})
					}
					pos = nextTokenIndex
					continue
				} else {
					// remove trailing comma for single-line functions
					pos = nextTokenIndex
					continue
				}
			} else if isNewline(nextToken) {
				// include the newline following an item with the previous comma
				if !isFloatingComma {
					formattedTokens = append(formattedTokens, delimiter, nextToken)
				} else if nextToken.Type == hclsyntax.TokenComment {
					// when we have a floating comma there is a newline between
					// the element and the comma, we insert the comma between the
					// element and the newline, so we don't need to include the
					// next token unless it's a comment
					formattedTokens = append(formattedTokens, nextToken)
				}
				pos += 2

				for isNewlineOrComment(tokens[pos]) {
					if tokens[pos].Type == hclsyntax.TokenComment {
						formattedTokens = append(formattedTokens, tokens[pos])
					}
					pos++
				}

				continue
			} else {
				// floating comma has already been appended, we can go to the next iteration
				if isFloatingComma {
					pos++
					continue
				}
			}
		} else {
			log.Fatalf("Expected `,` or `)` got `%s`", tokens[pos].Bytes)
		}

		formattedTokens = append(formattedTokens, delimiter)
		pos++
	}

	return formattedTokens, pos
}

func formatList(tokens hclwrite.Tokens, pos int, ctx Context) (hclwrite.Tokens, int) {
	var res, formattedTokens hclwrite.Tokens
	isMultiLine := false

	// [
	formattedTokens = append(formattedTokens, tokens[pos])
	pos++

	for {
		if tokens[pos].Type == hclsyntax.TokenCBrack {
			formattedTokens = append(formattedTokens, tokens[pos])
			pos++
			break
		}

		prevPos := pos - 1
		res, pos = formatExpression(tokens, pos, ctx, true)

		isMultiLine = isMultiLine || isNewline(res[0]) || isNewline(res[len(res)-1])

		delimiter := tokens[pos]

		// in case we have something like `[<element>` in a multiline list,
		// we move the element to a separate line by prepending the newline
		if tokens[prevPos].Type == hclsyntax.TokenOBrack && res[0].Type != hclsyntax.TokenComment &&
			(isMultiLine || (delimiter.Type == hclsyntax.TokenComma && isNewline(tokens[pos+1]))) {
			res = append(hclwrite.Tokens{
				&hclwrite.Token{
					Type:         hclsyntax.TokenNewline,
					Bytes:        []byte{'\n'},
					SpacesBefore: 0,
				},
			}, res...)
			isMultiLine = true
		}

		if delimiter.Type == hclsyntax.TokenCBrack {
			if isMultiLine {
				i := findLast(res, len(res)-1)

				if i >= 0 {
					trailingComma := &hclwrite.Token{
						Type:         hclsyntax.TokenComma,
						Bytes:        []byte{','},
						SpacesBefore: 0,
					}

					// <element>], since it's a multiline list we move ] to a
					// separate line and add a trailing comma after the element
					if i == len(res)-1 {
						res = append(res, trailingComma)
						res = append(res, &hclwrite.Token{
							Type:         hclsyntax.TokenNewline,
							Bytes:        []byte{'\n'},
							SpacesBefore: 0,
						})
					} else {
						// we append trailing comma before any comments or newlines
						// between the element and the closing bracket
						left, right := res[:i+1], res[i+1:]
						res = append(hclwrite.Tokens{}, left...)
						res = append(res, trailingComma)
						res = append(res, right...)
					}
				} else {
					// this can only happen in an empty map, res only contains  newlines and comments
					onlyNewlines := true
					for _, token := range res {
						if token.Type != hclsyntax.TokenNewline {
							onlyNewlines = false
							break
						}
					}
					// if there are no comments we collapse the map to a single line
					if onlyNewlines {
						res = hclwrite.Tokens{}
					}
				}
			}

			formattedTokens = append(formattedTokens, res...)
			formattedTokens = append(formattedTokens, delimiter)
			pos++
			break // ] is the delimiter, we don't need to the loop
		} else if delimiter.Type == hclsyntax.TokenComma {
			// when there is a comma on its own line, we move it up next to the
			// corresponding element
			isFloatingComma := isNewline(res[len(res)-1])
			if isFloatingComma {
				i := findLast(res, len(res)-1)
				res = append(res[:i+1], append(hclwrite.Tokens{delimiter}, res[i+1:]...)...)
			}

			formattedTokens = append(formattedTokens, res...)

			nextToken := tokens[pos+1]
			// <element>,]
			if nextToken.Type == hclsyntax.TokenCBrack {
				if isMultiLine {
					// ,] with element on the previous line
					if !isFloatingComma {
						formattedTokens = append(formattedTokens, delimiter)
						formattedTokens = append(formattedTokens, &hclwrite.Token{
							Type:         hclsyntax.TokenNewline,
							Bytes:        []byte{'\n'},
							SpacesBefore: 0,
						})
					}
					pos++
					continue
				} else {
					// remove trailing comma for single-line lists
					pos++
					continue
				}
			} else if isNewline(nextToken) {
				// this removes any trailing newlines after the comma, it's mostly here
				// to avoid having to parse the newline between trailing comma and closing
				// bracket as a separate element

				// include the newline following an item with the previous comma
				if !isFloatingComma {
					formattedTokens = append(formattedTokens, delimiter, nextToken)
				} else if nextToken.Type == hclsyntax.TokenComment {
					// when we have a floating comma there is a newline between
					// the element and the comma, we insert the comma between the
					// element and the newline, so we don't need to include the
					// next token unless it's a comment
					formattedTokens = append(formattedTokens, nextToken)
				}
				pos += 2

				for isNewlineOrComment(tokens[pos]) {
					if tokens[pos].Type == hclsyntax.TokenComment {
						formattedTokens = append(formattedTokens, tokens[pos])
					}
					pos++
				}

				continue
			} else {
				// floating comma has already been appended, we can go to the next iteration
				if isFloatingComma {
					pos++
					continue
				}
			}
		} else {
			log.Fatalf("Expected `,` or `]` got `%s`", tokens[pos].Bytes)
		}

		formattedTokens = append(formattedTokens, delimiter)
		pos++
	}

	return formattedTokens, pos
}

func formatParens(tokens hclwrite.Tokens, pos int, ctx Context) (hclwrite.Tokens, int) {
	var res, formattedTokens hclwrite.Tokens

	oParenPos := pos
	pos++

	res, pos = formatExpression(tokens, pos, ctx, true)

	if tokens[pos].Type != hclsyntax.TokenCParen {
		// this should never happen if the parser is implemented correctly
		log.Fatalf("Expected ')', got `%s`", tokens[pos].Bytes)
	}

	if oParenPos == 0 {
		if pos+1 == len(tokens) && !hasNewlines(res, ctx) {
			return res, pos
		}
	} else if !hasNewlines(res, ctx) {
		before := tokens[findLast(tokens, oParenPos-1)]
		// newline-delimited map item
		if (before.Type == hclsyntax.TokenColon || before.Type == hclsyntax.TokenEqual) && isNewline(tokens[pos+1]) {
			return res, pos + 1
		}

		afterPos := findFirst(tokens, pos+1)
		if afterPos < len(tokens) {
			after := tokens[afterPos]

			//nolint:lll // colon includes ternary operator
			pairs := map[hclsyntax.TokenType][]hclsyntax.TokenType{
				hclsyntax.TokenOParen:   {hclsyntax.TokenComma, hclsyntax.TokenCParen},
				hclsyntax.TokenOBrack:   {hclsyntax.TokenComma, hclsyntax.TokenCBrack},
				hclsyntax.TokenComma:    {hclsyntax.TokenComma, hclsyntax.TokenCParen, hclsyntax.TokenCBrack},
				hclsyntax.TokenEqual:    {hclsyntax.TokenComma, hclsyntax.TokenCBrace},
				hclsyntax.TokenColon:    {hclsyntax.TokenComma, hclsyntax.TokenCBrace, hclsyntax.TokenCBrack, hclsyntax.TokenCParen},
				hclsyntax.TokenFatArrow: {hclsyntax.TokenCBrace},
			}

			if v, ok := pairs[before.Type]; ok {
				if slices.Contains(v, after.Type) {
					return res, pos + 1
				}
			}
		}
	}

	formattedTokens = append(hclwrite.Tokens{tokens[oParenPos]}, res...)
	formattedTokens = append(formattedTokens, tokens[pos])
	pos++

	return formattedTokens, pos
}

func formatForExpression(tokens hclwrite.Tokens, pos int, ctx Context) (hclwrite.Tokens, int) {
	var res, formattedTokens hclwrite.Tokens

	isMapForExpression := tokens[pos].Type == hclsyntax.TokenOBrace

	// { or [
	formattedTokens = hclwrite.Tokens{tokens[pos]}
	pos++

	for {
		res, pos = collapseNewlines(tokens, pos)
		formattedTokens = append(formattedTokens, res...)

		// first non-newline / comment token we reach after opening brace, if "for" we are in a for expression
		if tokens[pos].Type == hclsyntax.TokenIdent && bytes.Equal(tokens[pos].Bytes, []byte("for")) {
			// for
			formattedTokens = append(formattedTokens, tokens[pos])
			pos++

			res, pos = collapseNewlines(tokens, pos)
			if len(res) > 1 || len(res) > 0 && res[0].Type != hclsyntax.TokenNewline {
				formattedTokens = append(formattedTokens, res...)
			}

			// k / k, v in
			for {
				res, pos = collapseNewlines(tokens, pos)
				// returns either an empty slice, a newline, a newline followed by comments
				// or two newlines with comments in between, we don't care about the
				// first two cases
				if len(res) > 1 || len(res) > 0 && res[0].Type != hclsyntax.TokenNewline {
					formattedTokens = append(formattedTokens, res...)
				}

				formattedTokens = append(formattedTokens, tokens[pos])

				if bytes.Equal(tokens[pos].Bytes, []byte("in")) {
					break
				}

				pos++
			}
			pos++

			res, pos = collapseNewlines(tokens, pos)
			if len(res) > 1 || len(res) > 0 && res[0].Type != hclsyntax.TokenNewline {
				formattedTokens = append(formattedTokens, res...)
			}

			// iterable :
			res, pos = formatExpression(tokens, pos, ctx, true)
			if tokens[pos].Type != hclsyntax.TokenColon {
				// should never happen
				log.Fatalf("Expected `:`, found %s", tokens[pos].Bytes)
			}

			lastToken := findLast(res, len(res)-1)
			if lastToken == len(res)-1 {
				res = append(res, tokens[pos])
			} else {
				trimmedRes, _ := collapseNewlines(res, lastToken+1)
				res = res[:lastToken+1]
				if len(trimmedRes) > 1 || len(trimmedRes) > 0 && trimmedRes[0].Type != hclsyntax.TokenNewline {
					res = append(res, trimmedRes...)
				}
				res = append(res, tokens[pos])
			}
			pos++
			formattedTokens = append(formattedTokens, res...)

			res, pos = collapseNewlines(tokens, pos)
			formattedTokens = append(formattedTokens, res...)

			// result expression
			res, pos = formatExpression(tokens, pos, ctx, true)
			if tokens[pos].Type == hclsyntax.TokenFatArrow {
				if formattedTokens[0].Type != hclsyntax.TokenOBrace {
					// should never happen
					log.Fatal("Found `=>` in list for expression")
				}

				if res[0].Type == hclsyntax.TokenOQuote {
					strRes, strPos := formatString(res, 0, ctx)

					if strRes[1].Type == hclsyntax.TokenTemplateInterp {
						if len(strRes) == 3 {
							strRes = trimNewlines(strRes[2 : len(strRes)-2])
						} else {
							c := 1
							for i, t := range strRes[2:] {
								//nolint:exhaustive // we don't need to handle other cases
								switch t.Type {
								case hclsyntax.TokenTemplateInterp:
									c++
								case hclsyntax.TokenTemplateSeqEnd:
									c--
								}
								// will be 0 once we reach TokenTemplateSeqEnd to close the initial TokenTemplateInterp
								if c == 0 {
									// if it's second to last token the whole string contains
									// a single top level interpolation sequence
									// we subtract 4 instead of 2 because we start from the
									// third element in the list instead of the first one
									if i == len(strRes)-4 {
										strRes = trimNewlines(strRes[2 : len(strRes)-2])
									}
									break
								}
							}
						}

						for strPos < len(res) {
							strRes = append(strRes, res[strPos])
							strPos++
						}
						res = strRes
					}
				}

				lastToken := findLast(res, len(res)-1)
				if lastToken == len(res)-1 {
					res = append(res, tokens[pos])
				} else {
					trimmedRes, _ := collapseNewlines(res, lastToken+1)
					res = res[:lastToken+1]
					if len(trimmedRes) > 1 || len(trimmedRes) > 0 && trimmedRes[0].Type != hclsyntax.TokenNewline {
						res = append(res, trimmedRes...)
					}
					res = append(res, tokens[pos])
				}
				pos++

				trimmedRes, trimmedPos := collapseNewlines(tokens, pos)
				if len(trimmedRes) > 1 || len(trimmedRes) > 0 && trimmedRes[0].Type != hclsyntax.TokenNewline {
					res = append(res, trimmedRes...)
				}

				valueRes, valuePos := formatExpression(tokens, trimmedPos, ctx, true)
				pos = valuePos

				res = append(res, valueRes...)
			}
			formattedTokens = append(formattedTokens, res...)

			// ] or }
			if tokens[pos].Type != hclsyntax.TokenCBrace && tokens[pos].Type != hclsyntax.TokenCBrack {
				// should only happen in case there's a bug in the parser
				expected := "]"
				if isMapForExpression {
					expected = "}"
				}
				log.Fatalf("Expected `%s`, got `%s`", expected, tokens[pos].Bytes)
			}
			formattedTokens = append(formattedTokens, tokens[pos])
			pos++

			break
		} else {
			return nil, pos
		}
	}

	return formattedTokens, pos
}

func formatMap(tokens hclwrite.Tokens, pos int, ctx Context) (hclwrite.Tokens, int) {
	var res, formattedTokens hclwrite.Tokens
	isMultiLine := false

	// {
	formattedTokens = append(formattedTokens, tokens[pos])
	pos++

	var mapElements []hclwrite.Tokens
	var mapElement hclwrite.Tokens

	for {
		if mapElement != nil {
			mapElements = append(mapElements, mapElement)
		}

		if tokens[pos].Type == hclsyntax.TokenCBrace {
			break
		}

		mapElement = hclwrite.Tokens{}

		// only handles "loose" newlines and single-line comments as the one at the
		// end of an element or following the comma at the end of element is picked
		// up with that element
		if isNewlineOrComment(tokens[pos]) {
			if tokens[pos].Type == hclsyntax.TokenComment && !bytes.Contains(tokens[pos].Bytes, []byte{'\n'}) {
				mapElement = append(mapElement, tokens[pos])
				pos++
				continue
			}

			isMultiLine = true

			// skip newline if the next one is newline or single-line comment
			// collapse empty multi-line map (`{\n}``) to inline one (`{}``)
			if tokens[pos].Type != hclsyntax.TokenNewline ||
				tokens[pos-1].Type != hclsyntax.TokenOBrace && tokens[pos+1].Type != hclsyntax.TokenNewline ||
				tokens[pos-1].Type == hclsyntax.TokenOBrace && tokens[pos+1].Type != hclsyntax.TokenCBrace {
				mapElement = append(mapElement, tokens[pos])
			}

			pos++
			continue
		}

		// parse key
		//nolint:exhaustive // default case handles all the other token types
		switch tokens[pos].Type {
		case hclsyntax.TokenIdent:
			i := pos + 1
			for tokens[i].Type == hclsyntax.TokenComment {
				i++
			}

			if tokens[i].Type == hclsyntax.TokenColon || tokens[i].Type == hclsyntax.TokenEqual {
				res = hclwrite.Tokens{tokens[pos]}
				pos++
				break
			}

			res, pos = formatExpression(tokens, pos, ctx, false)

			// bare expressions are valid keys in tf (e.g. `var.a + 5`),
			// we wrap them in parens unless used in providers attribute
			if ctx.blockType == "module" && ctx.attributeName == "providers" {
				break
			}

			oParen := hclwrite.Token{
				Type:         hclsyntax.TokenOParen,
				Bytes:        []byte("("),
				SpacesBefore: res[0].SpacesBefore,
			}
			cParen := hclwrite.Token{
				Type:         hclsyntax.TokenCParen,
				Bytes:        []byte(")"),
				SpacesBefore: 0,
			}

			res[0].SpacesBefore = 0
			res = append(hclwrite.Tokens{&oParen}, res...)
			res = append(res, &cParen)
		case hclsyntax.TokenOQuote:
			res, pos = formatString(tokens, pos, ctx)
			// remove useless quotes
			if len(res) == 3 && res[1].Type == hclsyntax.TokenQuotedLit {
				value, _ := hclsyntax.ParseStringLiteralToken(hclsyntax.Token{Type: res[1].Type, Bytes: res[1].Bytes})
				// we cannot unquote unless it's a valid identifier
				if IsIdentifier(value) {
					res = hclwrite.Tokens{&hclwrite.Token{
						// important to set teokn type to identifier, otherwise we may lose spaces!
						Type:         hclsyntax.TokenIdent,
						Bytes:        res[1].Bytes,
						SpacesBefore: res[0].SpacesBefore,
					}}
				}
			}
			if len(res) >= 5 && res[1].Type == hclsyntax.TokenTemplateInterp {
				c := 1
				for i, t := range res[2:] {
					switch t.Type {
					case hclsyntax.TokenTemplateInterp:
						c++
					case hclsyntax.TokenTemplateSeqEnd:
						c--
					}
					// will be 0 once we reach TokenTemplateSeqEnd to close the initial TokenTemplateInterp
					if c == 0 {
						// if it's second to last token the whole string contains
						// a single top level interpolation sequence
						// we subtract 4 instead of 2 because we start from the
						// third element in the list instead of the first one
						if i == len(res)-4 {
							res[1] = &hclwrite.Token{
								Type:         hclsyntax.TokenOParen,
								Bytes:        []byte{'('},
								SpacesBefore: res[0].SpacesBefore,
							}
							res[len(res)-2] = &hclwrite.Token{
								Type:         hclsyntax.TokenCParen,
								Bytes:        []byte{')'},
								SpacesBefore: res[len(res)-1].SpacesBefore,
							}
							res = res[1 : len(res)-1]
						}
						break
					}
				}
			}
		case hclsyntax.TokenOParen:
			res, pos = formatParens(tokens, pos, ctx)
		default:
			// bare expressions are valid keys in tf (e.g. `var.a + 5`)
			res, pos = formatExpression(tokens, pos, ctx, false)

			if len(res) == 1 && res[0].Type == hclsyntax.TokenNumberLit {
				break
			}

			oParen := hclwrite.Token{
				Type:         hclsyntax.TokenOParen,
				Bytes:        []byte("("),
				SpacesBefore: res[0].SpacesBefore,
			}
			cParen := hclwrite.Token{
				Type:         hclsyntax.TokenCParen,
				Bytes:        []byte(")"),
				SpacesBefore: 0,
			}

			res[0].SpacesBefore = 0
			res = append(hclwrite.Tokens{&oParen}, res...)
			res = append(res, &cParen)
		}

		// there can be multiline comment between map key and delimiter
		for tokens[pos].Type == hclsyntax.TokenComment {
			isMultiLine = isMultiLine || bytes.Contains(tokens[pos].Bytes, []byte{'\n'})
			res = append(res, tokens[pos])
			pos++
		}

		delimiter := tokens[pos]
		if delimiter.Type == hclsyntax.TokenColon {
			delimiter.Type = hclsyntax.TokenEqual
			delimiter.Bytes = []byte{'='}
		} else if delimiter.Type != hclsyntax.TokenEqual {
			// since the syntax of the input source is checked by hcl parser before
			// the formatting logic is even hit, this can only happen if there's
			// a bug inside the parser, which should hopefully be never
			log.Fatalf("Expected `=` or `:`, got `%s`", tokens[pos].Bytes)
		}
		res = append(res, delimiter)
		pos++

		// there can be multiline comment between map delimiter and value
		for tokens[pos].Type == hclsyntax.TokenComment {
			isMultiLine = isMultiLine || bytes.Contains(tokens[pos].Bytes, []byte{'\n'})
			res = append(res, tokens[pos])
			pos++
		}

		// parse value
		var value hclwrite.Tokens
		value, pos = formatExpression(tokens, pos, ctx, false)
		res = append(res, value...)

		mapElement = append(mapElement, res...)

		if tokens[pos].Type == hclsyntax.TokenNewline {
			if !isMultiLine {
				mapElements = adjustMapDelimiters(mapElements)
			}
			isMultiLine = true
		} else if tokens[pos].Type == hclsyntax.TokenCBrace {
			if isMultiLine {
				mapElement = append(mapElement, &hclwrite.Token{
					Type:         hclsyntax.TokenNewline,
					Bytes:        []byte{'\n'},
					SpacesBefore: 0,
				})
			}
			continue
		} else if tokens[pos].Type == hclsyntax.TokenComma {
			if isNewline(tokens[pos+1]) {
				if !isMultiLine {
					mapElements = adjustMapDelimiters(mapElements)
				}
				isMultiLine = true
				mapElement = append(mapElement, tokens[pos+1])
				pos++
			} else {
				if isMultiLine {
					mapElement = append(mapElement, &hclwrite.Token{
						Type:         hclsyntax.TokenNewline,
						Bytes:        []byte{'\n'},
						SpacesBefore: 0,
					})
				} else {
					mapElement = append(mapElement, tokens[pos])
				}
			}

			pos++
			continue
		} else {
			shouldBeComment := tokens[pos]
			if shouldBeComment.Type != hclsyntax.TokenComment || !isSingleLineComment(shouldBeComment.Bytes) {
				log.Fatalf("Expected `\\n`, `,` or '}', got `%s`", tokens[pos].Bytes)
			}
		}

		mapElement = append(mapElement, tokens[pos])
		pos++
	}

	for _, mapElement := range mapElements {
		formattedTokens = append(formattedTokens, mapElement...)
	}
	formattedTokens = append(formattedTokens, tokens[pos])
	pos++

	return formattedTokens, pos
}

func adjustMapDelimiters(mapElements []hclwrite.Tokens) []hclwrite.Tokens {
	if len(mapElements) == 0 {
		mapElements = append(mapElements, hclwrite.Tokens{
			&hclwrite.Token{
				Type:         hclsyntax.TokenNewline,
				Bytes:        []byte{'\n'},
				SpacesBefore: 0,
			},
		})
		return mapElements
	}

	for _, el := range mapElements {
		// mapElements can contain individual comments and newlines as elements,
		// this makes sure to only update terminators of proper map keys of the
		// form <key> <delimiter> <value> <terminator>
		if len(el) > 3 {
			el[len(el)-1] = &hclwrite.Token{
				Type:         hclsyntax.TokenNewline,
				Bytes:        []byte{'\n'},
				SpacesBefore: 0,
			}
		}
	}

	i := 0
	for i < len(mapElements) && isMultilineComment(mapElements[i][0]) {
		i++
	}

	if i == len(mapElements) || !isNewline(mapElements[i][0]) {
		mapElements = slices.Insert(
			mapElements,
			i,
			hclwrite.Tokens{
				&hclwrite.Token{
					Type:         hclsyntax.TokenNewline,
					Bytes:        []byte{'\n'},
					SpacesBefore: 0,
				},
			},
		)
	}

	return mapElements
}

func collapseNewlines(tokens hclwrite.Tokens, pos int) (hclwrite.Tokens, int) {
	var res hclwrite.Tokens

	for pos < len(tokens) && isNewlineOrComment(tokens[pos]) {
		if tokens[pos].Type != hclsyntax.TokenNewline || !isNewlineOrComment(tokens[pos-1]) {
			res = append(res, tokens[pos])
		}
		pos++
		continue
	}

	return res, pos
}

func trimNewlines(tokens hclwrite.Tokens) hclwrite.Tokens {
	// trim newlines in case the expression inside the ${} was multiline
	i := 0
	for tokens[i].Type == hclsyntax.TokenNewline {
		i++
	}
	j := len(tokens) - 1
	for tokens[j].Type == hclsyntax.TokenNewline {
		j--
	}
	return tokens[i : j+1]
}

// returns the position of the first token that isn't a comment or a newline
// if len(tokens) is returned it means there are no such tokens
func findFirst(tokens hclwrite.Tokens, start int) int {
	i := start
	for i < len(tokens) && isNewlineOrComment(tokens[i]) {
		i++
	}
	return i
}

// returns the position of first token that isn't a comment or a newline when going in reverse
// if -1 is returned it means there are no such tokens
func findLast(tokens hclwrite.Tokens, start int) int {
	i := start
	for i >= 0 && isNewlineOrComment(tokens[i]) {
		i--
	}
	return i
}

// checks if expression parsed with continueOnNewline indeed has newlines
// can be very efficient, but may be very hard to implement properly without major parser rework
func hasNewlines(expr hclwrite.Tokens, ctx Context) bool {
	res, _ := formatExpression(expr, 0, ctx, false)
	return len(expr) != len(res)
}

func isNewlineOrComment(token *hclwrite.Token) bool {
	return token.Type == hclsyntax.TokenNewline || token.Type == hclsyntax.TokenComment
}

func isNewline(token *hclwrite.Token) bool {
	return token.Type == hclsyntax.TokenNewline ||
		(token.Type == hclsyntax.TokenComment && bytes.HasSuffix(token.Bytes, []byte("\n")))
}

func isMultilineComment(token *hclwrite.Token) bool {
	return token.Type == hclsyntax.TokenComment && token.Bytes[0] == '/' && token.Bytes[1] == '*'
}

// https://developer.hashicorp.com/terraform/language/syntax/configuration#identifiers
func IsIdentifier(s string) bool {
	for i, r := range s {
		if unicode.IsDigit(r) {
			// first character cannot be a digit
			if i == 0 {
				return false
			}
			continue
		}
		if !unicode.IsLetter(r) && r != '-' && r != '_' {
			return false
		}
	}
	return true
}

func formatCommentsAndNewlines(src []byte, filename string) []byte {
	tokens, diags := hclsyntax.LexConfig(src, filename, hcl.InitialPos)

	if diags.HasErrors() {
		return src
	}

	// remove leading newlines from the file
	pos := 0
	for tokens[pos].Type == hclsyntax.TokenNewline {
		pos++
	}

	// the file contains only newlines
	if tokens[pos].Type == hclsyntax.TokenEOF && len(tokens) > 1 {
		return []byte("\n")
	}

	res := []byte{}
	for pos < len(tokens) {
		token := tokens[pos]
		//nolint:exhaustive // default case handles all the other token types
		switch token.Type {
		case hclsyntax.TokenComment:
			res = append(res, formatComment(token)...)
			// adds missing trailing newline in case file ends with a comment
			if tokens[pos+1].Type == hclsyntax.TokenEOF {
				if !bytes.HasSuffix(token.Bytes, []byte("\n")) {
					res = append(res, '\n')
				}
			} else if !strings.HasSuffix(filename, ".tfvars") &&
				pos > 0 && tokens[pos-1].Type == hclsyntax.TokenCBrace {
				// add newline between top level blocks if closing brace is followed by comment
				nextToken := tokens[pos+1]
				if nextToken.Type == hclsyntax.TokenIdent || nextToken.Type == hclsyntax.TokenComment {
					// if end of the newline token is the same as the start of the next token
					// and the source is formatted with tf fmt, it means that these are top
					// level blocks since there would be indentation for the next block if
					// that wasn't the case
					tokenEnd := token.Range.End.Byte
					nextTokenStart := nextToken.Range.Start.Byte
					if tokenEnd == nextTokenStart {
						res = append(res, '\n')
					}
				}
			}
		case hclsyntax.TokenNewline:
			if skipNewline(tokens, pos) {
				nextToken := tokens[pos+1]
				// next token is one of ), ], }; if we skip newline we will lose space
				// before the token. This doesn't happen with (, [, { because there
				// we always skip the newline in the "middle" as opposed at the end
				if nextToken.Type != hclsyntax.TokenNewline {
					nextTokenStart := nextToken.Range.Start.Byte
					tokenEnd := token.Range.End.Byte
					if tokenEnd < nextTokenStart {
						res = append(res, src[tokenEnd:nextTokenStart]...)
					}
				}
				pos++
				continue
			}
			res = append(res, token.Bytes...)
		case hclsyntax.TokenCBrace:
			res = append(res, token.Bytes...)

			// there's no need to add newlines between top level blocks since there are none in tfvars file
			if strings.HasSuffix(filename, ".tfvars") {
				pos++
				continue
			}

			nextToken := tokens[pos+1]
			// missing trailing newline
			switch nextToken.Type {
			case hclsyntax.TokenEOF:
				res = append(res, '\n')
			case hclsyntax.TokenNewline:
				// missing newline between blocks
				nextNextToken := tokens[pos+2]
				if nextNextToken.Type == hclsyntax.TokenIdent || nextNextToken.Type == hclsyntax.TokenComment {
					// if end of the newline token is the same as the start of the next token
					// and the source is formatted with tf fmt, it means that these are top
					// level blocks since there would be indentation for the next block if
					// that wasn't the case
					nextTokenEnd := nextToken.Range.End.Byte
					nextNextTokenStart := nextNextToken.Range.Start.Byte
					if nextTokenEnd == nextNextTokenStart {
						res = append(res, '\n')
					}
				}
			}
		default:
			res = append(res, token.Bytes...)
		}

		// whitespace isn't included in the token, so we need to check the token ranges and
		// include the bytes between the end of one token and the start of the next one
		if token.Type != hclsyntax.TokenEOF {
			tokenEnd := token.Range.End.Byte
			nextTokenStart := tokens[pos+1].Range.Start.Byte
			// [start, end)
			if tokenEnd < nextTokenStart {
				res = append(res, src[tokenEnd:nextTokenStart]...)
			}
		}

		pos++
	}

	return res
}

func formatComment(token hclsyntax.Token) []byte {
	var res []byte

	if token.Bytes[0] != '#' {
		// multiline comment
		if token.Bytes[1] == '*' {
			return slices.Clone(token.Bytes)
		}
		// C-style inline comment
		res = slices.Clone(token.Bytes[1:])
		res[0] = '#'
	} else {
		// # comment
		res = slices.Clone(token.Bytes)
	}

	// we don't insert space if the comment is followed by #, not to mess with shapes drawn using #
	if len(res) > 1 && res[1] != ' ' && res[1] != '#' {
		res = slices.Insert(res, 1, ' ')
	}

	return res
}

func skipNewline(tokens []hclsyntax.Token, pos int) bool {
	if pos == 0 {
		return true
	}

	// remove double trailing newline at the end of the file
	if tokens[pos+1].Type == hclsyntax.TokenEOF {
		return hasNewline(tokens[pos-1])
	}

	prev := tokens[pos-1]
	next := tokens[pos+1]

	if hasNewline(prev) &&
		(next.Type == hclsyntax.TokenNewline ||
			next.Type == hclsyntax.TokenCBrack ||
			next.Type == hclsyntax.TokenCParen ||
			next.Type == hclsyntax.TokenCBrace) {
		return true
	}

	if next.Type == hclsyntax.TokenNewline &&
		(prev.Type == hclsyntax.TokenOBrack ||
			prev.Type == hclsyntax.TokenOParen ||
			prev.Type == hclsyntax.TokenOBrace) {
		return true
	}

	return false
}

func hasNewline(token hclsyntax.Token) bool {
	return token.Type == hclsyntax.TokenNewline ||
		(token.Type == hclsyntax.TokenComment && bytes.HasSuffix(token.Bytes, []byte("\n")))
}

func isSingleLineComment(c []byte) bool {
	return c[0] == '#' || c[1] == '/'
}
