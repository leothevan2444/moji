package qbittorrent

import (
	"context"
	"testing"
)

func TestGetApplicationVersion(t *testing.T) {
	c := requireQBT(t)
	version, err := c.GetApplicationVersion(context.Background())
	if err != nil {
		t.Fatalf("Error getting application version: %v", err)
	}
	if version == "" {
		t.Fatal("Application version is empty")
	}
	t.Logf("Application version: %s", version)
}

func TestGetAPIVersion(t *testing.T) {
	c := requireQBT(t)
	apiVersion, err := c.GetAPIVersion(context.Background())
	if err != nil {
		t.Fatalf("Error getting API version: %v", err)
	}
	if apiVersion == "" {
		t.Fatal("API version is empty")
	}
	t.Logf("API version: %s", apiVersion)
}

func TestGetBuildInfo(t *testing.T) {
	c := requireQBT(t)
	buildInfo, err := c.GetBuildInfo(context.Background())
	if err != nil {
		t.Fatalf("Error getting build info: %v", err)
	}
	if buildInfo == nil {
		t.Fatal("Build info is nil")
	}
	t.Logf("Build info: %+v", buildInfo)
}

func TestGetApplicationPreferences(t *testing.T) {
	c := requireQBT(t)
	prefs, err := c.GetApplicationPreferences(context.Background())
	if err != nil {
		t.Fatalf("Error getting application preferences: %v", err)
	}
	if prefs == nil {
		t.Fatal("Application preferences is nil")
	}
	t.Logf("Application preferences: %+v", prefs)
	t.Log(ScanDirToMonitoredFolder, prefs.ProxyType)
}

func TestSetApplicationPreferences(t *testing.T) {
	c := requireDestructiveQBT(t)
	originalPrefs, err := c.GetApplicationPreferences(context.Background())
	if err != nil {
		t.Fatalf("Error getting original application preferences: %v", err)
	}

	newPrefs := *originalPrefs
	newPrefs.ProxyType.Value = 1 // Change a preference for testing

	err = c.SetApplicationPreferences(context.Background(), &newPrefs)
	if err != nil {
		t.Fatalf("Error setting application preferences: %v", err)
	}

	updatedPrefs, err := c.GetApplicationPreferences(context.Background())
	if err != nil {
		t.Fatalf("Error getting updated application preferences: %v", err)
	}

	if updatedPrefs.ProxyType != newPrefs.ProxyType {
		t.Fatalf("Application preferences not updated correctly. Expected ProxyType %d, got %d", newPrefs.ProxyType, updatedPrefs.ProxyType)
	}

	// Restore original preferences
	err = c.SetApplicationPreferences(context.Background(), originalPrefs)
	if err != nil {
		t.Fatalf("Error restoring original application preferences: %v", err)
	}
}

func TestGetDefaultSavePath(t *testing.T) {
	c := requireQBT(t)
	savePath, err := c.GetDefaultSavePath(context.Background())
	if err != nil {
		t.Fatalf("Error getting default save path: %v", err)
	}
	if savePath == "" {
		t.Fatal("Default save path is empty")
	}
	t.Logf("Default save path: %s", savePath)
}

func TestGetCookies(t *testing.T) {
	c := requireQBT(t)
	cookies, err := c.GetCookies(context.Background())
	if err != nil {
		t.Fatalf("Error getting cookies: %v", err)
	}
	if len(cookies) == 0 {
		t.Fatal("No cookies found")
	}
	for _, cookie := range cookies {
		t.Logf("Cookie: %s; Domain=%s; Path=%s", cookie.Name, cookie.Domain, cookie.Path)
	}
}
