package main

import (
	"fmt"
	"strings"
	"testing"
)

func TestRemovedComputed(t *testing.T) {
	df := &differ{
		diffs: map[string]*diffs{
			"azurerm_virtual_network": {
				resourceType:        "azurerm_virtual_network",
				removedComputedInv4: []string{"subnet"},
			},
		},
	}
	sourceCode := loadSourceCode("test.go", `resource "azurerm_virtual_network" "test" {
  name                = "acctest-VNET-%d"
  address_space       = ["10.0.0.0/16"]
  location            = azurerm_resource_group.test.location
  resource_group_name = azurerm_resource_group.test.name
  lifecycle {
    ignore_changes = ["ddos_protection_plan"]
  }
}
  `,
	)
	sourceCode.analysisBlocks()
	df.fixSourceCode(sourceCode)
	t.Logf("fixed lines: %s", sourceCode.fixedCode())
}

func TestFixBySchema(t *testing.T) {
	df := newDiffer("", "")
	df.diff()
	source := loadSourceCode("test.go", `resource "azurerm_subnet" "test" {
  name                 = "internal"
  resource_group_name  = azurerm_resource_group.test.name
  virtual_network_name = azurerm_virtual_network.test.name
  address_prefixes     = ["10.0.2.0/24"]
}
`)
	source.analysisBlocks()
	df.fixSourceCode(source)
	t.Logf("fixed lines: %s", source.fixedCode())
}

func TestFindAllTestsFiles(t *testing.T) {
	files := findAllTestFiles("")
	fmt.Printf("aaaa all test files: %v", strings.Join(files, "\n"))
}
