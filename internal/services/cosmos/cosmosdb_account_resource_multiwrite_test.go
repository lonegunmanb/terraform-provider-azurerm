// Copyright IBM Corp. 2014, 2025
// SPDX-License-Identifier: MPL-2.0

package cosmos_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-provider-azurerm/internal/acceptance"
	"github.com/hashicorp/terraform-provider-azurerm/internal/acceptance/check"
)

func TestAccCosmosDBAccount_updateConsistencyWithMultipleWriteLocations(t *testing.T) {
	data := acceptance.BuildTestData(t, "azurerm_cosmosdb_account", "test")
	r := CosmosDBAccountResource{}

	data.ResourceTest(t, r, []acceptance.TestStep{
		{
			Config: r.multiWriteWithConsistency(data, "Session", true),
			Check: acceptance.ComposeTestCheckFunc(
				check.That(data.ResourceName).ExistsInAzure(r),
			),
		},
		data.ImportStep(),
		{
			Config: r.multiWriteWithConsistency(data, "Strong", false),
			Check: acceptance.ComposeTestCheckFunc(
				check.That(data.ResourceName).ExistsInAzure(r),
			),
		},
		data.ImportStep(),
		{
			Config: r.multiWriteWithConsistency(data, "Session", true),
			Check: acceptance.ComposeTestCheckFunc(
				check.That(data.ResourceName).ExistsInAzure(r),
			),
		},
		data.ImportStep(),
	})
}

func (CosmosDBAccountResource) multiWriteWithConsistency(data acceptance.TestData, consistencyLevel string, multipleWriteLocationsEnabled bool) string {
	geoLocations := `
  geo_location {
    location          = azurerm_resource_group.test.location
    failover_priority = 0
  }`

	if multipleWriteLocationsEnabled {
		geoLocations = fmt.Sprintf(`
  geo_location {
    location          = azurerm_resource_group.test.location
    failover_priority = 0
  }

  geo_location {
    location          = "%s"
    failover_priority = 1
  }`, data.Locations.Secondary)
	}

	return fmt.Sprintf(`
provider "azurerm" {
  features {}
}

resource "azurerm_resource_group" "test" {
  name     = "acctestRG-%d"
  location = "%s"
}

resource "azurerm_cosmosdb_account" "test" {
  name                             = "acctest-%d"
  location                         = azurerm_resource_group.test.location
  resource_group_name              = azurerm_resource_group.test.name
  offer_type                       = "Standard"
  multiple_write_locations_enabled = %t

  consistency_policy {
    consistency_level = "%s"
  }

%s
}
`, data.RandomInteger, data.Locations.Primary, data.RandomInteger, multipleWriteLocationsEnabled, consistencyLevel, geoLocations)
}
