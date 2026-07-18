package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccFieldIndexResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccFieldIndexResourceConfig("test_collection_idx", "city", "Keyword"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("qdrant_field_index.test", "collection_name", "test_collection_idx"),
					resource.TestCheckResourceAttr("qdrant_field_index.test", "field_name", "city"),
					resource.TestCheckResourceAttr("qdrant_field_index.test", "field_type", "Keyword"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "qdrant_field_index.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateId:     "test_collection_idx/city",
			},
		},
	})
}

func testAccFieldIndexResourceConfig(collectionName, fieldName, fieldType string) string {
	return fmt.Sprintf(`
provider "qdrant" {
  host = "localhost"
  port = 6334
}

resource "qdrant_collection" "test" {
  name = %[1]q
  vectors = [
    {
      size     = 128
      distance = "Cosine"
    }
  ]
}

resource "qdrant_field_index" "test" {
  collection_name = qdrant_collection.test.name
  field_name      = %[2]q
  field_type      = %[3]q
}
`, collectionName, fieldName, fieldType)
}
