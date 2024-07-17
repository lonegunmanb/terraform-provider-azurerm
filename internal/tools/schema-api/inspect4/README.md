# a tool to fix acceptance tests under 4.0 beta flag

## how to use

```bash
cd terraform-provider-azurerm/internal/tools/schema-api/inspect4
go run main.go -f network
```

command line arguments:

```
-f folder: specify the folder name of a service to fix all test files under the given folder
```