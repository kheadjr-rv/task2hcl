# AWS ECS Task Definition to HCL

task2hcl converts `environment` and `secrets` within a json task definition to a hcl locals block.

- `environment` The environment variables to pass to a container.
- `secrets` The secrets to pass to the container.

### Installation

```
go install github.com/kheadjr-rv/task2hcl
```

### Usage

```
task2hcl example/foo-service-task-definition.json
```

[Example](./example/foo-service-task-definition.json) task definition
```json
{
    "environment": [
        {
            "name": "FOO_SETTING",
            "value": "true"
        }
    ],
    "secrets": [
        {
            "valueFrom": "/foo-service/bar-secret",
            "name": "BAR_SECRET"
        }
    ]
}
```

Converted to HCL

```
# The environment variables to pass to a container.

{
  name  = "FOO_SETTING",
  value = "true"
},

# The secrets to pass to the container.

{
  name      = "BAR_SECRET",
  valueFrom = "${local.app_paramstore_prefix}/bar_secret"
},
{
  name      = "BAR_KEY",
  valueFrom = "${local.app_paramstore_prefix}/bar_key"
},
```