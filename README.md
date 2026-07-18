# Terraform Provider for Qdrant

[![CI](https://github.com/01Joseph-Hwang10/terraform-provider-qdrant/actions/workflows/test.yml/badge.svg)](https://github.com/01Joseph-Hwang10/terraform-provider-qdrant/actions/workflows/test.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/01Joseph-Hwang10/terraform-provider-qdrant)](https://goreportcard.com/report/github.com/01Joseph-Hwang10/terraform-provider-qdrant)
[![License: MPL 2.0](https://img.shields.io/badge/License-MPL_2.0-brightgreen.svg)](https://opensource.org/licenses/MPL-2.0)

This is a Terraform provider for managing [Qdrant](https://qdrant.tech/) vector database resources. It allows you to manage collections and payload field indexes using HashiCorp Configuration Language (HCL).

**Note**: This is another implementation of [yagurcloud/terraform-provider-qdrant](https://github.com/yagurcloud/terraform-provider-qdrant).

## Features

- **Collections**: Create, update (limited), and delete Qdrant collections.
- **Named Vectors**: Support for single and multiple named vectors with configurable distances (Cosine, Euclidean, Dot, Manhattan).
- **Field Indexes**: Manage payload schema indexes (Keyword, Integer, Float, Geo, Text, Bool, Datetime, etc.) for efficient filtering.
- **Data Sources**: Discover existing collections.
- **Authentication**: Native support for API keys.
- **Secure Communication**: Support for gRPC over TLS (HTTPS).

## Installation

To use this provider, add it to your Terraform configuration:

```hcl
terraform {
  required_providers {
    qdrant = {
      source = "01Joseph-Hwang10/qdrant"
    }
  }
}
```

## Provider Configuration

You can configure the provider using the following attributes or environment variables:

| Argument   | Environment Variable | Description              | Default    |
| ---------- | -------------------- | ------------------------ | ---------- |
| `host`     | `QDRANT_HOST`        | Qdrant gRPC host         | (Required) |
| `port`     | `QDRANT_PORT`        | Qdrant gRPC port         | `6334`     |
| `https`    | `QDRANT_HTTPS`       | Use HTTPS for connection | `false`    |
| `insecure` | `QDRANT_INSECURE`    | Skip TLS verification    | `false`    |
| `api_key`  | `QDRANT_API_KEY`     | Qdrant API key           |            |

```hcl
provider "qdrant" {
  host    = "localhost"
  port    = 6334
  api_key = "your-secret-key"
}
```

## Resources

### qdrant_collection

Manages a Qdrant collection.

```hcl
resource "qdrant_collection" "example" {
  name = "my_collection"
  vectors = [
    {
      name     = "dense"
      size     = 1536
      distance = "Cosine"
    }
  ]
}
```

### qdrant_field_index

Manages a payload field index.

```hcl
resource "qdrant_field_index" "city_index" {
  collection_name = qdrant_collection.example.name
  field_name      = "city"
  field_type      = "Keyword"
}
```

## Data Sources

### qdrant_collections

Lists all collection names.

```hcl
data "qdrant_collections" "all" {}

output "collection_names" {
  value = data.qdrant_collections.all.collections
}
```

## Development

### Requirements

- [Go](https://golang.org/doc/install) 1.22+ (to build the provider plugin)
- [Terraform](https://www.terraform.io/downloads.html) 1.0+

### Building

```bash
# Clone the repository
git clone https://github.com/01Joseph-Hwang10/terraform-provider-qdrant
cd terraform-provider-qdrant

# Build the provider
make build
```

### Testing

```bash
# Run unit tests
make test
```

## License

This provider is distributed under the [Mozilla Public License 2.0](LICENSE).
