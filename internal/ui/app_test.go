package ui

import (
	"testing"

	"github.com/BitwiseSang/dots/internal/config"
	"github.com/BitwiseSang/dots/internal/ui/views"
)

func TestAppModelPreservesHomeCursorOnReturn(t *testing.T) {
	cfg := config.DefaultConfig()
	app := NewApp(cfg, views.ViewHome)

	// Initially on ViewHome, cursor at 0 (Backup)
	if app.currentView != views.ViewHome {
		t.Fatalf("expected initial view to be ViewHome, got %v", app.currentView)
	}
	if app.home.Cursor() != 0 {
		t.Fatalf("expected initial home cursor to be 0 (Backup), got %d", app.home.Cursor())
	}

	// Navigate to Setup (item 1)
	m, _ := app.Update(views.NavigateMsg{View: views.ViewSetup})
	app = m.(AppModel)
	if app.currentView != views.ViewSetup {
		t.Fatalf("expected current view to be ViewSetup, got %v", app.currentView)
	}

	// Return to Home
	m, _ = app.Update(views.NavigateMsg{View: views.ViewHome})
	app = m.(AppModel)
	if app.currentView != views.ViewHome {
		t.Fatalf("expected view to return to ViewHome, got %v", app.currentView)
	}
	if app.home.Cursor() != 1 {
		t.Errorf("expected home cursor to remain on 1 (Setup) after returning, got %d", app.home.Cursor())
	}

	// Navigate to Edit (item 2)
	m, _ = app.Update(views.NavigateMsg{View: views.ViewEdit})
	app = m.(AppModel)

	// Return to Home
	m, _ = app.Update(views.NavigateMsg{View: views.ViewHome})
	app = m.(AppModel)
	if app.home.Cursor() != 2 {
		t.Errorf("expected home cursor to remain on 2 (Edit) after returning, got %d", app.home.Cursor())
	}
}
