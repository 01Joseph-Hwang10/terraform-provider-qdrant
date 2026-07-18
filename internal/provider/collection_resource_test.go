package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccCollectionResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccCollectionResourceConfig("test_collection", 128, "Cosine"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("qdrant_collection.test", "name", "test_collection"),
					resource.TestCheckResourceAttr("qdrant_collection.test", "vectors.0.size", "128"),
					resource.TestCheckResourceAttr("qdrant_collection.test", "vectors.0.distance", "Cosine"),
				),
			},
			// ImportState testing
			{
				ResourceName:                         "qdrant_collection.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateId:                        "test_collection",
				ImportStateVerifyIdentifierAttribute: "name",
			},
			// Update testing (not supported, should trigger replacement if changed)
			// For now just test Delete by finishing the test
		},
	})
}

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"qdrant": providerserver.NewProtocol6WithError(New("test")()),
}

func testAccCollectionResourceConfig(name string, size int, distance string) string {
	return fmt.Sprintf(`
provider "qdrant" {
  host = "localhost"
  port = 6334
}

resource "qdrant_collection" "test" {
  name = %[1]q
  vectors = [
    {
      size     = %[2]d
      distance = %[3]q
    }
  ]
}
`, name, size, distance)
}
