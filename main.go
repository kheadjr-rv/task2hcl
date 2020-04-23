package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"text/template"

	"github.com/Masterminds/sprig"

	"github.com/aws/aws-sdk-go/service/ecs"
)

var envs string = `
{{- if .}}
# The environment variables to pass to a container.
{{range .}}
{
  name  = "{{.Name}}",
  value = "{{.Value}}"
},{{end}}{{end}}
`

var secrets string = `
{{- if .}}
# The secrets to pass to the container.
{{range .}}
{
  name      = "{{.Name}}",
  valueFrom = "${local.app_paramstore_prefix}/{{.Name | lower | replace "-" "_"}}"
},{{end}}{{end}}
`

func main() {
	flag.Parse()

	if flag.NArg() < 1 {
		fmt.Fprintf(os.Stderr, "usage: %s TASK_FILE\n\n", os.Args[0])
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

	toHCL(parsed.ContainerDefinitions[0].Environment, envs)

	toHCL(parsed.ContainerDefinitions[0].Secrets, secrets)

}

func toHCL(data interface{}, task string) {
	tmpl := template.Must(template.New("hcl").Funcs(sprig.TxtFuncMap()).Parse(task))

	err := tmpl.Execute(os.Stdout, data)
	if err != nil {
		panic(err)
	}
}
