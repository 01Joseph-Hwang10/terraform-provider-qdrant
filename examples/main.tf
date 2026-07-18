terraform {
  required_providers {
    qdrant = {
      source = "registry.terraform.io/qdrant/qdrant"
    }
  }
}

provider "qdrant" {
  host = "localhost"
  port = 6334
}

resource "qdrant_collection" "example" {
  name = "example_collection"
  vectors = [
    {
      size     = 128
      distance = "Cosine"
    }
  ]
}

resource "qdrant_field_index" "example" {
  collection_name = qdrant_collection.example.name
  field_name      = "city"
  field_type      = "Keyword"
}

data "qdrant_collections" "all" {}

output "collections" {
  value = data.qdrant_collections.all.collections
}
