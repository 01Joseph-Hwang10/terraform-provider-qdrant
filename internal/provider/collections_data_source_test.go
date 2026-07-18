package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccCollectionsDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCollectionsDataSourceConfig("data_test_collection"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.qdrant_collections.test", "collections.0", "data_test_collection"),
				),
			},
		},
	})
}

func testAccCollectionsDataSourceConfig(name string) string {
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

data "qdrant_collections" "test" {
  depends_on = [qdrant_collection.test]
}
`, name)
}
