package portal

import (
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/liteyuki/yuki-id-portal/internal/auth"
	"github.com/liteyuki/yuki-id-portal/internal/logto"
)

type AppCatalogItem struct {
	ID                    string   `json:"id" yaml:"id"`
	Name                  string   `json:"name" yaml:"name"`
	Description           string   `json:"description" yaml:"description"`
	URL                   string   `json:"url" yaml:"url"`
	Icon                  string   `json:"icon" yaml:"icon"`
	Type                  string   `json:"type" yaml:"type"`
	RequiredRoles         []string `json:"requiredRoles" yaml:"required_roles"`
	RequiredOrganizations []string `json:"requiredOrganizations" yaml:"required_organizations"`
	Enabled               bool     `json:"enabled" yaml:"enabled"`
	SortOrder             int      `json:"sortOrder" yaml:"sort_order"`
	Source                string   `json:"source" yaml:"source"`
}

type AppCatalogResponseItem struct {
	AppCatalogItem
	Accessible bool     `json:"accessible"`
	Reasons    []string `json:"reasons"`
}

type Announcement struct {
	ID                  string     `json:"id" yaml:"id"`
	Title               string     `json:"title" yaml:"title"`
	Content             string     `json:"content" yaml:"content"`
	Severity            string     `json:"severity" yaml:"severity"`
	TargetRoles         []string   `json:"targetRoles" yaml:"target_roles"`
	TargetOrganizations []string   `json:"targetOrganizations" yaml:"target_organizations"`
	TargetUsers         []string   `json:"targetUsers" yaml:"target_users"`
	Pinned              bool       `json:"pinned" yaml:"pinned"`
	StartsAt            *time.Time `json:"startsAt,omitempty" yaml:"starts_at"`
	EndsAt              *time.Time `json:"endsAt,omitempty" yaml:"ends_at"`
}

func FilterApps(apps []AppCatalogItem, user auth.User) []AppCatalogResponseItem {
	result := make([]AppCatalogResponseItem, 0, len(apps))
	for _, app := range apps {
		if !app.Enabled {
			continue
		}
		accessible, reasons := canAccessApp(app, user)
		result = append(result, AppCatalogResponseItem{
			AppCatalogItem: app,
			Accessible:     accessible,
			Reasons:        reasons,
		})
	}
	return result
}

func MergeManagedApps(managed []logto.ManagementApplication, overrides []AppCatalogItem) []AppCatalogItem {
	overrideByID := make(map[string]AppCatalogItem, len(overrides))
	overrideByName := make(map[string]AppCatalogItem, len(overrides))
	for _, item := range overrides {
		overrideByID[item.ID] = item
		overrideByName[normalizeAppKey(item.Name)] = item
	}

	result := make([]AppCatalogItem, 0, len(managed))
	for index, app := range managed {
		item := appFromManagement(app, index)
		override, ok := overrideByID[item.ID]
		if !ok {
			override, ok = overrideByName[normalizeAppKey(item.Name)]
		}
		if ok {
			item = mergeAppOverride(item, override)
		}
		result = append(result, item)
	}

	sort.SliceStable(result, func(i, j int) bool {
		if result[i].SortOrder != result[j].SortOrder {
			return result[i].SortOrder < result[j].SortOrder
		}
		return strings.ToLower(result[i].Name) < strings.ToLower(result[j].Name)
	})
	return result
}

func FilterAnnouncements(items []Announcement, user auth.User, now time.Time) []Announcement {
	result := make([]Announcement, 0, len(items))
	for _, item := range items {
		if item.StartsAt != nil && now.Before(*item.StartsAt) {
			continue
		}
		if item.EndsAt != nil && now.After(*item.EndsAt) {
			continue
		}
		if !matchesAudience(item.TargetUsers, []string{user.Sub}) {
			if len(item.TargetUsers) > 0 {
				continue
			}
			if !matchesAudience(item.TargetRoles, user.Roles) {
				continue
			}
			if !matchesAudience(item.TargetOrganizations, user.Organizations) {
				continue
			}
		}
		result = append(result, item)
	}
	return result
}

func canAccessApp(app AppCatalogItem, user auth.User) (bool, []string) {
	reasons := []string{}
	if len(app.RequiredRoles) > 0 && !matchesAudience(app.RequiredRoles, user.Roles) {
		reasons = append(reasons, "missing_required_role")
	}
	if len(app.RequiredOrganizations) > 0 && !matchesAudience(app.RequiredOrganizations, user.Organizations) {
		reasons = append(reasons, "missing_required_organization")
	}
	return len(reasons) == 0, reasons
}

func appFromManagement(app logto.ManagementApplication, index int) AppCatalogItem {
	return AppCatalogItem{
		ID:          app.ID,
		Name:        fallbackString(app.Name, app.ID),
		Description: app.Description,
		URL:         appURL(app),
		Icon:        "workflow",
		Type:        app.Type,
		Enabled:     true,
		SortOrder:   10_000 + index,
		Source:      "management",
	}
}

func mergeAppOverride(base AppCatalogItem, override AppCatalogItem) AppCatalogItem {
	if override.ID != "" {
		base.ID = override.ID
	}
	if override.Name != "" {
		base.Name = override.Name
	}
	if override.Description != "" {
		base.Description = override.Description
	}
	if override.URL != "" {
		base.URL = override.URL
	}
	if override.Icon != "" {
		base.Icon = override.Icon
	}
	if override.Type != "" {
		base.Type = override.Type
	}
	if override.RequiredRoles != nil {
		base.RequiredRoles = override.RequiredRoles
	}
	if override.RequiredOrganizations != nil {
		base.RequiredOrganizations = override.RequiredOrganizations
	}
	base.Enabled = override.Enabled
	if override.SortOrder != 0 {
		base.SortOrder = override.SortOrder
	}
	base.Source = "management"
	return base
}

func appURL(app logto.ManagementApplication) string {
	if value := stringMapValue(app.ProtectedAppMetadata, "origin"); value != "" {
		return value
	}
	if value := stringMapValue(app.CustomData, "portalUrl"); value != "" {
		return value
	}
	if value := stringMapValue(app.CustomClientMetadata, "portalUrl"); value != "" {
		return value
	}
	if uri := firstStringSliceValue(app.OidcClientMetadata, "redirectUris"); uri != "" {
		parsed, err := url.Parse(uri)
		if err == nil && parsed.Scheme != "" && parsed.Host != "" {
			return parsed.Scheme + "://" + parsed.Host
		}
	}
	return ""
}

func stringMapValue(values map[string]any, key string) string {
	value, _ := values[key].(string)
	return value
}

func firstStringSliceValue(values map[string]any, key string) string {
	value, ok := values[key]
	if !ok {
		return ""
	}
	switch typed := value.(type) {
	case []string:
		if len(typed) == 0 {
			return ""
		}
		return typed[0]
	case []any:
		if len(typed) == 0 {
			return ""
		}
		text, _ := typed[0].(string)
		return text
	default:
		return ""
	}
}

func fallbackString(value, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}

func normalizeAppKey(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func matchesAudience(required []string, actual []string) bool {
	if len(required) == 0 {
		return true
	}
	actualSet := make(map[string]struct{}, len(actual))
	for _, value := range actual {
		actualSet[value] = struct{}{}
	}
	for _, value := range required {
		if _, ok := actualSet[value]; ok {
			return true
		}
	}
	return false
}
