package routeros

import (
	"cmp"
	"log"
	"slices"
)

var driftAttributeSlice driftObjects

// Drift objects

type driftObjects []driftObject

type driftObject struct {
	Version   uint64
	ros       string
	Resources map[string][]driftAttribute
}

type driftAttribute struct {
	TF string
	MT string
	// Before inverts the version check: the mapping applies to RouterOS versions
	// strictly lower than the object version. Used when the schema already carries
	// the new (current) MikroTik name and older routers still use the old one.
	Before bool
}

// Add an object to the slice.
// ros - RouterOS version (7.17.2)
// resourcePath - resource path
// attrTF - attribute name in the latest schema version
// attrMT - attribute name in MikroTik parameters
func (do *driftObjects) Add(ros, resourcePath, attrTF, attrMT string) {
	do.add(ros, resourcePath, driftAttribute{TF: attrTF, MT: attrMT})
}

// AddBefore adds a mapping that applies to RouterOS versions lower than ros.
// ros - the first RouterOS version where the MikroTik parameter got its current name (7.21)
// attrTF - attribute name in the schema (equal to the current MikroTik name)
// attrMT - the old MikroTik parameter name used before ros
func (do *driftObjects) AddBefore(ros, resourcePath, attrTF, attrMT string) {
	do.add(ros, resourcePath, driftAttribute{TF: attrTF, MT: attrMT, Before: true})
}

func (do *driftObjects) add(ros, resourcePath string, attr driftAttribute) {
	version, err := parseRouterOSVersion(ros)
	if err != nil {
		log.Fatal(err)
	}

	if i := do.index(version); i != -1 {
		(*do)[i].Resources[resourcePath] = append((*do)[i].Resources[resourcePath], attr)
	} else {
		*do = append(*do, driftObject{
			Version:   version,
			ros:       ros,
			Resources: map[string][]driftAttribute{resourcePath: {attr}},
		})
	}
}

// Sorting a slice in descending order.
func (do *driftObjects) SortDesc() {
	slices.SortFunc(driftAttributeSlice, func(a, b driftObject) int {
		return cmp.Compare(a.Version, b.Version) * -1
	})
}

// Getting the object index by RouterOS version.
func (do *driftObjects) index(version uint64) int {
	for i := range *do {
		if version == (*do)[i].Version {
			return i
		}
	}
	return -1
}

// Obtaining a map to match TF attributes and MT parameters for further transformation.
// Direct output (for TF to MT serialization) and reverse output (MT to TF) are provided.
func (do *driftObjects) GetDriftMap(ros, resName string, reverse bool) (res map[string]string) {
	version, err := parseRouterOSVersion(ros)
	if err != nil {
		log.Fatal(err)
	}

	res = map[string]string{}
	for i := range *do {
		for _, attr := range (*do)[i].Resources[resName] {
			if attr.Before != (version >= (*do)[i].Version) {
				if !reverse {
					res[attr.TF] = attr.MT
				} else {
					res[attr.MT] = attr.TF
				}
			}
		}
	}
	return
}
