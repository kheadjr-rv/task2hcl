package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/service/ecs"

	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

var (
	task = flag.Bool("task", false, "input is a ecs task definition")
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

	for _, v := range parsed.ContainerDefinitions[0].Environment {
		body.AppendUnstructuredTokens(environment(*v.Name, *v.Value))
	}

	body.AppendNewline()

	for _, v := range parsed.ContainerDefinitions[0].Secrets {
		body.AppendUnstructuredTokens(secrets(*v.Name, *v.ValueFrom))
	}

	fmt.Printf("%s", hclwrite.Format(tf.Bytes()))

}

func environment(name, value string) hclwrite.Tokens {
	return hclwrite.Tokens{
		{
			Type:  hclsyntax.TokenCBrace,
			Bytes: []byte{'{'},
		},
		{
			Type:  hclsyntax.TokenNewline,
			Bytes: []byte{'\n'},
		},
		{
			Type:  hclsyntax.TokenIdent,
			Bytes: []byte("name"),
		},
		{
			Type:  hclsyntax.TokenEqual,
			Bytes: []byte{'='},
		},
		{
			Type:  hclsyntax.TokenStringLit,
			Bytes: []byte(fmt.Sprintf("\"%s\"", name)),
		},
		{
			Type:  hclsyntax.TokenNewline,
			Bytes: []byte{'\n'},
		},
		{
			Type:  hclsyntax.TokenIdent,
			Bytes: []byte("value"),
		},
		{
			Type:  hclsyntax.TokenEqual,
			Bytes: []byte{'='},
		},
		{
			Type:  hclsyntax.TokenStringLit,
			Bytes: []byte(fmt.Sprintf("\"%s\"", value)),
		},
		{
			Type:  hclsyntax.TokenNewline,
			Bytes: []byte{'\n'},
		},
		{
			Type:  hclsyntax.TokenCBrace,
			Bytes: []byte{'}'},
		},
		{
			Type:  hclsyntax.TokenComma,
			Bytes: []byte{','},
		},
		{
			Type:  hclsyntax.TokenNewline,
			Bytes: []byte{'\n'},
		},
	}
}

func secrets(name, valueFrom string) hclwrite.Tokens {
	return hclwrite.Tokens{
		{
			Type:  hclsyntax.TokenCBrace,
			Bytes: []byte{'{'},
		},
		{
			Type:  hclsyntax.TokenNewline,
			Bytes: []byte{'\n'},
		},
		{
			Type:  hclsyntax.TokenIdent,
			Bytes: []byte("name"),
		},
		{
			Type:  hclsyntax.TokenEqual,
			Bytes: []byte{'='},
		},
		{
			Type:  hclsyntax.TokenStringLit,
			Bytes: []byte(fmt.Sprintf("\"%s\"", name)),
		},
		{
			Type:  hclsyntax.TokenNewline,
			Bytes: []byte{'\n'},
		},
		{
			Type:  hclsyntax.TokenIdent,
			Bytes: []byte("valueFrom"),
		},
		{
			Type:  hclsyntax.TokenEqual,
			Bytes: []byte{'='},
		},
		{
			Type:  hclsyntax.TokenStringLit,
			Bytes: []byte(fmt.Sprintf("\"%s\"", valueFrom)),
		},
		{
			Type:  hclsyntax.TokenNewline,
			Bytes: []byte{'\n'},
		},
		{
			Type:  hclsyntax.TokenCBrace,
			Bytes: []byte{'}'},
		},
		{
			Type:  hclsyntax.TokenComma,
			Bytes: []byte{','},
		},
		{
			Type:  hclsyntax.TokenNewline,
			Bytes: []byte{'\n'},
		},
	}
}
