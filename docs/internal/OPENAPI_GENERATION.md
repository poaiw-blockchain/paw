# OpenAPI/Swagger Documentation Generation

## Current State

PAW uses a manually maintained OpenAPI spec at `docs/api/openapi.yaml` covering core endpoints.

## Automated Generation from Proto Files

### Option 1: buf with protoc-gen-openapiv2 (Recommended)

Create `proto/buf.gen.openapi.yaml`:
```yaml
version: v1
plugins:
  - name: openapiv2
    out: ../docs/api/generated
    opt:
      - logtostderr=true
      - allow_merge=true
      - merge_file_name=openapi
```

Run: `cd proto && buf generate --template buf.gen.openapi.yaml`

### Option 2: grpc-gateway swagger plugin

Use existing SDK pattern from `.tmp/sdk/proto/buf.gen.swagger.yaml`:
```yaml
version: v1
plugins:
  - name: swagger
    out: ../docs/api/generated
    opt: logtostderr=true,fqn_for_swagger_name=true,simple_operation_ids=true
```

### Prerequisites

```bash
go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@latest
```

### Makefile Target

Add to Makefile:
```makefile
proto-swagger:
	@echo "Generating OpenAPI spec..."
	@cd proto && buf generate --template buf.gen.openapi.yaml
	@echo "OpenAPI spec generated at docs/api/generated/openapi.swagger.json"
```

## Notes

- Proto files already have `google.api.http` annotations for REST endpoints
- Generated spec will be OpenAPI 2.0 (Swagger); convert to 3.0 with swagger2openapi if needed
- Keep manual `docs/api/openapi.yaml` as source of truth for custom descriptions
