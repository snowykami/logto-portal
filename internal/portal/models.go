package portal

import (
	"time"

	"github.com/liteyuki/yuki-id-portal/internal/auth"
)

type AppCatalogItem struct {
	ID                    string   `json:"id" yaml:"id"`
	Name                  string   `json:"name" yaml:"name"`
	Description           string   `json:"description" yaml:"description"`
	URL                   string   `json:"url" yaml:"url"`
	Icon                  string   `json:"icon" yaml:"icon"`
	RequiredRoles         []string `json:"requiredRoles" yaml:"required_roles"`
	RequiredOrganizations []string `json:"requiredOrganizations" yaml:"required_organizations"`
	Enabled               bool     `json:"enabled" yaml:"enabled"`
	SortOrder             int      `json:"sortOrder" yaml:"sort_order"`
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
