package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

func main() {
	flag.Parse()

	if flag.NArg() < 1 {
		fmt.Fprintf(os.Stderr, "usage: %s FILE\n\n", os.Args[0])
		os.Exit(1)
	}

	path := flag.Arg(0)

	f, err := os.Open(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	defer f.Close()

	parsed := &ecs.TaskDefinition{}

	dec := json.NewDecoder(f)
	dec.DisallowUnknownFields()
	if err = dec.Decode(parsed); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	tf := hclwrite.NewEmptyFile()
	body := tf.Body()

	localsBlock := body.AppendNewBlock("locals", nil)

	envs := make([]cty.Value, 0)
	for _, v := range parsed.ContainerDefinitions[0].Environment {
		env := cty.ObjectVal(map[string]cty.Value{
			"name":  cty.StringVal(*v.Name),
			"value": cty.StringVal(*v.Value),
		})
		envs = append(envs, env)
	}
	localsBlock.Body().SetAttributeRaw("env_vars", tokensForValue(cty.ListVal(envs)))

	secrets := make([]cty.Value, 0)
	for _, v := range parsed.ContainerDefinitions[0].Secrets {

		// format the valueFrom to match name
		valueFrom := strings.ReplaceAll(strings.ToLower(*v.Name), "-", "_")

		secret := cty.ObjectVal(map[string]cty.Value{
			"name":      cty.StringVal(*v.Name),
			"valueFrom": cty.StringVal(valueFrom),
		})
		secrets = append(secrets, secret)
	}
	localsBlock.Body().SetAttributeRaw("secret_env_vars", tokensForValue(cty.ListVal(secrets)))

	fmt.Printf("%s", hclwrite.Format(tf.Bytes()))
}

// credit hashicorp/hcl/v2/hclwrite/generate.go
func tokensForValue(val cty.Value) hclwrite.Tokens {
	toks := appendTokensForValue(val, nil)
	// format(toks) // fiddle with the SpacesBefore field to get canonical spacing
	return toks
}

func appendTokensForValue(val cty.Value, toks hclwrite.Tokens) hclwrite.Tokens {
	switch {

	case !val.IsKnown():
		panic("cannot produce tokens for unknown value")

	case val.IsNull():
		toks = append(toks, &hclwrite.Token{
			Type:  hclsyntax.TokenIdent,
			Bytes: []byte(`null`),
		})

	case val.Type() == cty.Bool:
		var src []byte
		if val.True() {
			src = []byte(`true`)
		} else {
			src = []byte(`false`)
		}
		toks = append(toks, &hclwrite.Token{
			Type:  hclsyntax.TokenIdent,
			Bytes: src,
		})

	case val.Type() == cty.Number:
		bf := val.AsBigFloat()
		srcStr := bf.Text('f', -1)
		toks = append(toks, &hclwrite.Token{
			Type:  hclsyntax.TokenNumberLit,
			Bytes: []byte(srcStr),
		})

	case val.Type() == cty.String:
		// TODO: If it's a multi-line string ending in a newline, format
		// it as a HEREDOC instead.
		src := escapeQuotedStringLit(val.AsString())
		toks = append(toks, &hclwrite.Token{
			Type:  hclsyntax.TokenOQuote,
			Bytes: []byte{'"'},
		})
		if len(src) > 0 {
			toks = append(toks, &hclwrite.Token{
				Type:  hclsyntax.TokenQuotedLit,
				Bytes: src,
			})
		}
		toks = append(toks, &hclwrite.Token{
			Type:  hclsyntax.TokenCQuote,
			Bytes: []byte{'"'},
		})

	case val.Type().IsListType() || val.Type().IsSetType() || val.Type().IsTupleType():
		toks = append(toks, &hclwrite.Token{
			Type:  hclsyntax.TokenOBrack,
			Bytes: []byte{'['},
		})

		i := 0
		for it := val.ElementIterator(); it.Next(); {
			if i > 0 {
				toks = append(toks, &hclwrite.Token{
					Type:  hclsyntax.TokenComma,
					Bytes: []byte{','},
				}, &hclwrite.Token{
					Type:  hclsyntax.TokenNewline,
					Bytes: []byte{'\n'},
				})
			}
			_, eVal := it.Element()
			toks = appendTokensForValue(eVal, toks)
			i++
		}

		toks = append(toks, &hclwrite.Token{
			Type:  hclsyntax.TokenCBrack,
			Bytes: []byte{']'},
		})

	case val.Type().IsMapType() || val.Type().IsObjectType():
		toks = append(toks, &hclwrite.Token{
			Type:  hclsyntax.TokenOBrace,
			Bytes: []byte{'{'},
		}, &hclwrite.Token{
			Type:  hclsyntax.TokenNewline,
			Bytes: []byte{'\n'},
		})

		i := 0
		for it := val.ElementIterator(); it.Next(); {
			if i > 0 {
				toks = append(toks, &hclwrite.Token{
					Type:  hclsyntax.TokenComma,
					Bytes: []byte{','},
				}, &hclwrite.Token{
					Type:  hclsyntax.TokenNewline,
					Bytes: []byte{'\n'},
				})
			}
			eKey, eVal := it.Element()
			if hclsyntax.ValidIdentifier(eKey.AsString()) {
				toks = append(toks, &hclwrite.Token{
					Type:  hclsyntax.TokenIdent,
					Bytes: []byte(eKey.AsString()),
				})
			} else {
				toks = appendTokensForValue(eKey, toks)
			}
			toks = append(toks, &hclwrite.Token{
				Type:  hclsyntax.TokenEqual,
				Bytes: []byte{'='},
			})
			toks = appendTokensForValue(eVal, toks)
			i++
		}

		toks = append(toks, &hclwrite.Token{
			Type:  hclsyntax.TokenNewline,
			Bytes: []byte{'\n'},
		}, &hclwrite.Token{
			Type:  hclsyntax.TokenCBrace,
			Bytes: []byte{'}'},
		})

	default:
		panic(fmt.Sprintf("cannot produce tokens for %#v", val))
	}

	return toks
}

func escapeQuotedStringLit(s string) []byte {
	if len(s) == 0 {
		return nil
	}
	buf := make([]byte, 0, len(s))
	for i, r := range s {
		switch r {
		case '\n':
			buf = append(buf, '\\', 'n')
		case '\r':
			buf = append(buf, '\\', 'r')
		case '\t':
			buf = append(buf, '\\', 't')
		case '"':
			buf = append(buf, '\\', '"')
		case '\\':
			buf = append(buf, '\\', '\\')
		case '$', '%':
			buf = appendRune(buf, r)
			remain := s[i+1:]
			if len(remain) > 0 && remain[0] == '{' {
				// Double up our template introducer symbol to escape it.
				buf = appendRune(buf, r)
			}
		default:
			if !unicode.IsPrint(r) {
				var fmted string
				if r < 65536 {
					fmted = fmt.Sprintf("\\u%04x", r)
				} else {
					fmted = fmt.Sprintf("\\U%08x", r)
				}
				buf = append(buf, fmted...)
			} else {
				buf = appendRune(buf, r)
			}
		}
	}
	return buf
}

func appendRune(b []byte, r rune) []byte {
	l := utf8.RuneLen(r)
	for i := 0; i < l; i++ {
		b = append(b, 0) // make room at the end of our buffer
	}
	ch := b[len(b)-l:]
	utf8.EncodeRune(ch, r)
	return b
}
