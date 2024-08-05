package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-provider-azurerm/internal/provider"
	"github.com/hashicorp/terraform-provider-azurerm/internal/tools/schema-api/providerjson"
)

func getSchema(v4 bool) *providerjson.ProviderSchemaJSON {
	envVal := ""
	if v4 {
		envVal = "true"
	}
	os.Setenv("ARM_FOURPOINTZERO_BETA", envVal)
	data := providerjson.LoadData()
	schema, err := providerjson.ProviderFromRaw(data)
	if err != nil {
		panic(err)
	}
	return schema
}

type XPath string
type DiffContainer map[string][]XPath

func (d DiffContainer) addItem(rt string, path string) {
	d[rt] = append(d[rt], XPath(path))
}

type diffPaths []string

type diffs struct {
	ResourceType        string      `json:"resource_type"`
	DeletedInV4         diffPaths   `json:"deleted"`
	RemovedComputedInv4 diffPaths   `json:"oc_removed"`
	RenamedInV4         [][2]string `json:"reanmed"`
}

func (d *diffs) clone() *diffs {
	item := &diffs{
		ResourceType:        d.ResourceType,
		DeletedInV4:         make(diffPaths, len(d.DeletedInV4)),
		RemovedComputedInv4: make(diffPaths, len(d.RemovedComputedInv4)),
		RenamedInV4:         make([][2]string, len(d.RenamedInV4)),
	}
	copy(item.DeletedInV4, d.DeletedInV4)
	copy(item.RemovedComputedInv4, d.RemovedComputedInv4)
	copy(item.RenamedInV4, d.RenamedInV4)
	return item
}

func (d *diffs) addDeleted(path string) {
	d.DeletedInV4 = append(d.DeletedInV4, path)
}

func (d *diffs) addRemovedComputed(path string) {
	d.RemovedComputedInv4 = append(d.RemovedComputedInv4, path)
}

func (d *diffs) deleteRemovedComputed(path string) {
	d.RemovedComputedInv4 = removeFromSlice(d.RemovedComputedInv4, path)
}

func (d *diffs) addRenamed(pathV3, pathV4 string) {
	d.RenamedInV4 = append(d.RenamedInV4, [2]string{pathV3, pathV4})
}

func appendPath(parent, key string) string {
	if parent == "" {
		return key
	}
	return fmt.Sprintf("%s.%s", parent, key)
}

type differ struct {
	fixResources map[string]bool
	SchemaV3     *providerjson.ProviderSchemaJSON
	SchemaV4     *providerjson.ProviderSchemaJSON
	diffs        map[string]*diffs
}

func newDiffer() *differ {
	d := &differ{
		SchemaV3:     getSchema(false),
		SchemaV4:     getSchema(true),
		diffs:        map[string]*diffs{},
		fixResources: map[string]bool{},
	}

	os.Setenv("ARM_FOURPOINTZERO_BETA", "true")
	for _, item := range provider.SupportedTypedServices() {
		for _, r := range item.Resources() {
			d.fixResources[r.ResourceType()] = true
		}
	}

	for _, item := range provider.SupportedUntypedServices() {
		for rt := range item.SupportedResources() {
			d.fixResources[rt] = true
		}
	}
	return d
}

func (d *differ) shouldFixResource(rt string) bool {
	if len(d.fixResources) == 0 {
		return true
	}
	return d.fixResources[rt]
}

func (d *differ) patch() {
	if diff := d.diffs["azurerm_network_interface"]; diff != nil {
		diff.RemovedComputedInv4 = removeFromSlice(diff.RemovedComputedInv4, "dns_servers")
	}
}

func (d *differ) diff() {
	for rt, r3 := range d.SchemaV3.ResourcesMap {
		r4, ok := d.SchemaV4.ResourcesMap[rt]
		if !ok {
			// log.Printf("%s not exists in v4", rt)
			continue
		}
		if _, ok := d.diffs[rt]; !ok {
			d.diffs[rt] = &diffs{ResourceType: rt}
		}
		d.diffResource(rt, r3, r4, "")
	}

	d.patch()
}

func skipComputed(rt string, key string) bool {
	switch rt {
	case "azurerm_route_filter":
		if key == "rule" {
			return true
		}
	}

	return false
}

var deprecateReg = regexp.MustCompile("(in favour of|superseded by|been renamed to|please use)[^`]*`([^`]*)`")

func (d *differ) diffResource(rt string, v3, v4 providerjson.ResourceJSON, parentPath string) {
	diffs := d.diffs[rt]
	for key, schemaV3 := range v3.Schema {
		// deleted in v4
		itemPath := appendPath(parentPath, key)
		schemaV4, ok := v4.Schema[key]
		if !ok {
			// if there is a near edit-distance key then add as renamed
			if match := deprecateReg.FindStringSubmatch(schemaV3.Deprecated); len(match) > 2 {
				// if moved to a new resource, it's not a rename. also some special props should not be treated as a rename
				if key != "resource_group_name" && !strings.HasPrefix(match[2], "azurerm_") {
					diffs.addRenamed(itemPath, match[2])
					continue
				}
			}
			diffs.addDeleted(itemPath)
			continue
		}

		// O+C in v3 but O only in v4 and v4 has not set default value for it
		if schemaV3.Optional && schemaV3.Computed && !schemaV4.Computed && schemaV4.Default == nil {
			// prop may renamed (enable_xxx -> xxx_enabled), both props exists in v3 and v4, but v3 is compute and v4 is not
			// while v4 will remove the old props
			if len(schemaV3.ConflictWith) <= len(schemaV4.ConflictWith) {
				if !skipComputed(rt, key) {
					diffs.addRemovedComputed(itemPath)
				}
			}
		}

		if schemaV3.Elem == nil {
			continue
		}

		if eleV3, ok := schemaV3.Elem.(*providerjson.ResourceJSON); ok {
			eleV4, ok := schemaV4.Elem.(*providerjson.ResourceJSON)
			if !ok {
				log.Printf("%s:%s v4 ele is not resource schema while v3 is", rt, itemPath)
				continue
			} else {
				d.diffResource(rt, *eleV3, *eleV4, itemPath)
			}
		}
	}
}
func (d *differ) printDiffs() {
	for rt, diffs := range d.diffs {
		if len(diffs.DeletedInV4)+len(diffs.RemovedComputedInv4)+len(diffs.RenamedInV4) == 0 {
			continue
		}

		log.Printf("===== %s =====\n", rt)
		if len(diffs.DeletedInV4) > 0 {
			log.Printf("deleted In V4: %v\n", diffs.DeletedInV4)
		}
		if len(diffs.RenamedInV4) > 0 {
			log.Printf("renamed In V4: %v\n", diffs.RenamedInV4)
		}
		if len(diffs.RemovedComputedInv4) > 0 {
			log.Printf("removed computed In V4: %v\n", diffs.RemovedComputedInv4)
		}
	}
}

var (
	onlyFixResourcesArgs  = flag.String("resource", "", "only fix specified resources")
	fixResourcesOfService = flag.String("ros", "", "fix all resources-of-service(ROS)")
	serviceFolder         = flag.String("f", "", "specify folder name to fix")
)

func main() {
	d := newDiffer()
	d.diff()
	for k, diff := range d.diffs {
		if len(diff.RemovedComputedInv4) == 0 && len(diff.DeletedInV4) == 0 && len(diff.RenamedInV4) == 0 {
			delete(d.diffs, k)
		}
	}
	b, err := json.Marshal(d.diffs)
	if err != nil {
		panic(err.Error())
	}
	fmt.Println(string(b))
}
