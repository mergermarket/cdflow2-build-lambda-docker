Still a work in progress, but planned usage in cdflow.yaml something like:

```yaml
version: 2
build:
  node_lambda:
    image: mergermarket/cdflow2-build-lambda
    params:
      image: node:latest
      command: npm test && npm run build
      target: .
      handler: index.handler
  go_lambda:
    image: mergermarket/cdflow2-build-lambda
    params:
      image: golang:latest
      command: go test ./... && go build cmd/app
      target: app
      handler: app
config:
  image: mergermarket/cdflow2-config-aws-simple
terraform:
  image: hashicorp/terraform
```

