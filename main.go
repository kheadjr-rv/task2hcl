package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/zclconf/go-cty/cty"

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
	localsBlock.Body().SetAttributeValue("env_vars", cty.ListVal(envs))

	secrets := make([]cty.Value, 0)
	for _, v := range parsed.ContainerDefinitions[0].Secrets {
		secret := cty.ObjectVal(map[string]cty.Value{
			"name":      cty.StringVal(*v.Name),
			"valueFrom": cty.StringVal(*v.ValueFrom),
		})
		secrets = append(secrets, secret)
	}
	localsBlock.Body().SetAttributeValue("secret_env_vars", cty.ListVal(secrets))

	fmt.Printf("%s", hclwrite.Format(tf.Bytes()))
}
