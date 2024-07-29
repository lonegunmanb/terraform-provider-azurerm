package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

func (d *diffs) fixBlock(block *codeBlock) {
	if block.fixedLines == nil {
		block.fixedLines = make([]string, len(block.lines))
		copy(block.fixedLines, block.lines)
	}
	deleteProps(d.resourceType, d.deletedInV4, block)
	fixRenamed(d.resourceType, d.RenamedInV4, block)
	fixRemovedComputed(d.removedComputedInv4, block)
	specialPatch(d.resourceType, block)
}

// add some special patch logic to specific resources
func specialPatch(resourceType string, block *codeBlock) {
	switch resourceType {
	case "azurerm_public_ip":
		// stanard sku only support static allocate, basic sku support dynamic&static
		// update dynamic allocate to static, if not specify sku as basic
		allocateIdx := 0
		specifiedSKU := false
		isStaticAllocate := false
		for idx, line := range block.fixedLines {
			if strings.Contains(line, "sku") {
				specifiedSKU = true
			}
			// no need to fix for static allocation
			if strings.Contains(line, "allocation_method") {
				allocateIdx = idx
				isStaticAllocate = strings.Contains(line, "Static")
			}
		}
		// 4.0 changes sku default value to Standard from Basic, so need specify to Basic in acc tests
		if !specifiedSKU && !isStaticAllocate {
			block.fixedLines[allocateIdx] += "\n  sku = \"Basic\""
		}
	}
}

func fixRemovedComputed(removedComputedInv4 []string, block *codeBlock) {
	// if field not exits in the block, then we need a ignore instruction
	newComputed := []string{}
	fullText := strings.Join(block.fixedLines, "\n")
	for _, item := range removedComputedInv4 {
		// if item not specified in the config
		if !strings.Contains(fullText, item) {
			newComputed = append(newComputed, item)
		}
	}
	removedComputedInv4 = newComputed
	if len(removedComputedInv4) > 0 {
		hasLifeCycle := false
		newFixed := make([]string, 0, len(block.fixedLines)+10)
		for idx, line := range block.fixedLines {
			if strings.Contains(line, "lifecycle") {
				hasLifeCycle = true
			} else if strings.Contains(line, "ignore_changes") {
				if strings.HasSuffix(line, "]") {
					// ignore changes in the same line
					toAppend := []string{}
					for _, item := range removedComputedInv4 {
						if !strings.Contains(line, item) {
							toAppend = append(toAppend, item)
						}
					}
					if len(toAppend) > 0 {
						block.fixedLines[idx] = fmt.Sprintf("%s, %s]", line[:len(line)-1], strings.Join(toAppend, ", "))
					}
				} else {
					log.Printf("ignore_changes in multiple line cannot process currently")
				}
			}
		}
		if hasLifeCycle {
			log.Printf("skip set ignore for for lifecycle exists")
		} else {
			// append a lifecycle to the end of the block
			for idx, line := range block.fixedLines {
				if strings.HasPrefix(line, "}") {
					if strings.TrimSpace(block.fixedLines[idx-1]) != "" {
						newFixed = append(newFixed, "")
					}
					newFixed = append(newFixed,
						"  lifecycle {",
						fmt.Sprintf("    ignore_changes = [ %s ]", strings.Join(removedComputedInv4, ", ")),
						"  }",
						line,
					)
				} else {
					newFixed = append(newFixed, line)
				}
			}
			block.fixedLines = newFixed
		}

	}

}

func patchRenamed(resourceType, updatedLine string, rename [2]string) (res string) {

	// if xxx_disabled prop renamed to xxx_enabled
	if strings.Contains(rename[0], "disable") && strings.Contains(rename[1], "enable") {
		res = strings.Replace(updatedLine, "true", "false", 1)
		if res == updatedLine {
			res = strings.Replace(res, "false", "true", 1)
		}
		return res
	}

	// some value changed for renamed properties
	res = updatedLine
	if resourceType == "azurerm_subnet" {
		// strange logic in `func expandEnforceSubnetNetworkPolicy(enabled bool)`
		// internal/services/network/subnet_resource.go#L825
		if rename[0] == "enforce_private_link_endpoint_network_policies" {
			res = strings.Replace(res, "true", "\"Disabled\"", 1)
			res = strings.Replace(res, "false", "\"Enabled\"", 1)
		} else if rename[0] == "enforce_private_link_service_network_policies" {
			res = strings.Replace(res, "true", "false", 1)
			if res == updatedLine {
				res = strings.Replace(res, "false", "true", 1)
			}
		} else if rename[1] == "private_endpoint_network_policies" {
			res = strings.Replace(res, "true", "\"Enabled\"", 1)
			res = strings.Replace(res, "false", "\"Disabled\"", 1)
		}
	}
	return res
}

func fixRenamed(resourceType string, renamed [][2]string, block *codeBlock) {
	for _, rename := range renamed {
		for idx, line := range block.fixedLines {
			if strings.Contains(line, rename[0]+" =") {
				updatedLine := strings.Replace(line, rename[0], rename[1], 1)
				block.fixedLines[idx] = patchRenamed(resourceType, updatedLine, rename)
			}
		}
	}

	if resourceType == "azurerm_vpn_gateway_nat_rule" {
		inReg := regexp.MustCompile(`(internal_mapping|external_mapping).*=.*\[(.*)\]`)
		for idx, line := range block.fixedLines {
			if matches := inReg.FindStringSubmatch(line); len(matches) > 0 {
				name := matches[1]
				maps := []string{}
				for _, val := range strings.Split(matches[2], ",") {
					val = strings.Trim(val, "\" \t")
					vals := strings.Split(val, ":")
					addr := fmt.Sprintf("  %s {\n    address_space = \"%s\"\n", name, vals[0])
					if len(vals) > 1 {
						addr += fmt.Sprintf("    port_range = \"%s\"\n", vals[1])
					}
					addr += "  }\n"
					maps = append(maps, addr)
					block.fixedLines[idx] = strings.Join(maps, "\n")
				}
			}
		}

	}
}

func deleteProps(resourceType string, deleted []string, block *codeBlock) {
	_ = resourceType
	if len(deleted) == 0 {
		return
	}

	newFixes := make([]string, 0, len(block.fixedLines))
	for _, line := range block.fixedLines {
		toDelete := false
		for _, item := range deleted {
			if strings.HasPrefix(strings.TrimSpace(line), item) {
				toDelete = true
				break
			}
		}
		if !toDelete {
			newFixes = append(newFixes, line)
		}
	}
	block.fixedLines = newFixes
}

func (d *differ) fixAllTestsByDir(dir string) {
	allTestFiles := findAllTestFiles(dir)
	writeBack := true // os.Getenv("WRITE_BACK") == "true"
	for _, file := range allTestFiles {
		d.fixByFilePath(file, writeBack)
	}
}

func (d *differ) fixByFilePath(file string, writeBack bool) {
	source := loadSourceCode(file, "")
	source.analysisBlocks()
	d.fixSourceCode(source)
	if writeBack {
		fixedCode := source.fixedCode()
		if !strings.EqualFold(fixedCode, source.content) {
			os.WriteFile(file, []byte(fixedCode), 0644)
			go func() {
				cmd := exec.Command("terrafmt", "fmt", "-f", file)
				_, err := cmd.CombinedOutput()
				if err != nil {
					log.Printf("terrafmt failed with %s\n", err)
				}
			}()
		}
	}
}

func (d *differ) fixSourceCode(source *SourceCodeFile) {
	// get resourc block
	log.Printf("processing source code of %s", source.file)
	for idx, b := range source.blocks {
		if b.resource == "" {
			continue
		}

		if !d.shouldFixResource(b.resource) {
			continue
		}

		diff, ok := d.diffs[b.resource]
		// no
		if !ok {
			continue
		}

		// if go source code contains logic of 4.0beta check, skip it
		if goBlock := source.lastGoBlockOf(idx); goBlock != nil {
			if strings.Contains(goBlock.origin, "features.FourPointOhBeta") {
				continue
			}
		}

		log.Printf("try to fix block of resource %s", diff.resourceType)
		diff = diff.patchByContext(source.contextOf(idx))
		diff.fixBlock(b)
	}
}

type computedOfResource struct {
	parentResource string
	eleResource    string
	filedName      string
}

var patchComputedFieldOfResource = []computedOfResource{
	{"azurerm_key_vault", "azurerm_key_vault_access_policy", "access_policy"},
	{"azurerm_automation_runbook", "azurerm_automation_job_schedule", "job_schedule"},
	{"azurerm_network_security_group", "azurerm_network_security_rule", "security_rule"},
	{"azurerm_virtual_network", "azurerm_subnet", "subnet"},
}

func (d *diffs) patchByContext(ctx []*codeBlock) *diffs {
	for _, item := range patchComputedFieldOfResource {
		if item.parentResource == d.resourceType {
			dd := d.clone()
			// if there is no access policy resource, then we don't need add such ignore_changes to access_policy prop
			if !contextHasResource(ctx, item.eleResource) {
				dd.deleteRemovedComputed(item.filedName)
			}
			return dd
		}
	}

	switch d.resourceType {
	case "azurerm_vpn_gateway_nat_rule":
		item := d.clone()
		item.deleteRemovedComputed("external_mapping")
		item.deleteRemovedComputed("internal_mapping")
		item.addRenamed("external_address_space_mappings", "external_mapping")
		item.addRenamed("internal_address_space_mappings", "internal_mapping")
	}
	return d
}
