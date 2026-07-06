package portal

import (
	"testing"

	"github.com/liteyuki/yuki-id-portal/internal/logto"
)

func TestMergeManagedAppsDoesNotAppendStaticOnlyEntries(t *testing.T) {
	apps := MergeManagedApps([]logto.ManagementApplication{
		{
			ID:   "real-app",
			Name: "Real App",
			Type: "SPA",
		},
	}, []AppCatalogItem{
		{
			ID:        "fake-app",
			Name:      "Fake App",
			URL:       "https://fake.example.com",
			Enabled:   true,
			SortOrder: 1,
		},
	})

	if len(apps) != 1 {
		t.Fatalf("expected only managed applications, got %d", len(apps))
	}
	if apps[0].ID != "real-app" {
		t.Fatalf("expected real-app, got %q", apps[0].ID)
	}
}

func TestMergeManagedAppsAppliesMatchingOverride(t *testing.T) {
	apps := MergeManagedApps([]logto.ManagementApplication{
		{
			ID:   "real-app",
			Name: "Real App",
			Type: "SPA",
		},
	}, []AppCatalogItem{
		{
			ID:        "real-app",
			Icon:      "git-branch",
			URL:       "https://real.example.com",
			Enabled:   true,
			SortOrder: 1,
		},
	})

	if len(apps) != 1 {
		t.Fatalf("expected one managed application, got %d", len(apps))
	}
	if apps[0].URL != "https://real.example.com" {
		t.Fatalf("expected override url, got %q", apps[0].URL)
	}
	if apps[0].Icon != "git-branch" {
		t.Fatalf("expected override icon, got %q", apps[0].Icon)
	}
}
