package portal

import (
	"os"
	"sort"

	"gopkg.in/yaml.v3"
)

func LoadAppCatalog(path string) ([]AppCatalogItem, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var apps []AppCatalogItem
	if err := yaml.Unmarshal(data, &apps); err != nil {
		return nil, err
	}
	sort.SliceStable(apps, func(i, j int) bool {
		return apps[i].SortOrder < apps[j].SortOrder
	})
	return apps, nil
}

func LoadAnnouncements(path string) ([]Announcement, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var items []Announcement
	if err := yaml.Unmarshal(data, &items); err != nil {
		return nil, err
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Pinned != items[j].Pinned {
			return items[i].Pinned
		}
		return items[i].ID < items[j].ID
	})
	return items, nil
}
