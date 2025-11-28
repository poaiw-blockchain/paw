package tests

import (
	"context"
	"testing"
	"time"

	"status/pkg/config"
	"status/pkg/incidents"

	"github.com/stretchr/testify/assert"
)

func TestNewManager(t *testing.T) {
	cfg := &config.Config{}
	manager := incidents.NewManager(cfg)
	assert.NotNil(t, manager)
}

func TestCreateIncident(t *testing.T) {
	cfg := &config.Config{}
	manager := incidents.NewManager(cfg)

	// Start manager
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go manager.Start(ctx)

	incident, err := manager.CreateIncident(
		"Test Incident",
		"This is a test incident",
		incidents.SeverityMajor,
		[]string{"API", "Database"},
	)

	assert.NoError(t, err)
	assert.NotNil(t, incident)
	assert.Equal(t, "Test Incident", incident.Title)
	assert.Equal(t, incidents.SeverityMajor, incident.Severity)
	assert.Equal(t, incidents.StatusInvestigating, incident.Status)
	assert.NotEmpty(t, incident.Updates)
}

func TestUpdateIncident(t *testing.T) {
	cfg := &config.Config{}
	manager := incidents.NewManager(cfg)

	// Start manager
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go manager.Start(ctx)

	// Create incident
	incident, _ := manager.CreateIncident(
		"Test Incident",
		"Test description",
		incidents.SeverityMinor,
		[]string{"API"},
	)

	// Update incident
	err := manager.UpdateIncident(
		incident.ID,
		"Issue identified and fix deployed",
		incidents.StatusIdentified,
	)

	assert.NoError(t, err)

	// Verify update
	updated, _ := manager.GetIncident(incident.ID)
	assert.Equal(t, incidents.StatusIdentified, updated.Status)
	assert.Len(t, updated.Updates, 2) // Initial + new update
}

func TestResolveIncident(t *testing.T) {
	cfg := &config.Config{}
	manager := incidents.NewManager(cfg)

	// Start manager
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go manager.Start(ctx)

	// Create incident
	incident, _ := manager.CreateIncident(
		"Test Incident",
		"Test description",
		incidents.SeverityCritical,
		[]string{"Blockchain"},
	)

	// Resolve incident
	err := manager.UpdateIncident(
		incident.ID,
		"Issue resolved",
		incidents.StatusResolved,
	)

	assert.NoError(t, err)

	// Verify resolution
	resolved, _ := manager.GetIncident(incident.ID)
	assert.Equal(t, incidents.StatusResolved, resolved.Status)
	assert.NotNil(t, resolved.ResolvedAt)
}

func TestGetIncident(t *testing.T) {
	cfg := &config.Config{}
	manager := incidents.NewManager(cfg)

	// Start manager
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go manager.Start(ctx)

	// Wait for historical data to load
	time.Sleep(100 * time.Millisecond)

	// Test getting non-existent incident
	_, err := manager.GetIncident(9999)
	assert.Error(t, err)

	// Create and get incident
	incident, _ := manager.CreateIncident(
		"Test Incident",
		"Test description",
		incidents.SeverityMinor,
		[]string{"API"},
	)

	retrieved, err := manager.GetIncident(incident.ID)
	assert.NoError(t, err)
	assert.Equal(t, incident.ID, retrieved.ID)
	assert.Equal(t, incident.Title, retrieved.Title)
}

func TestGetActiveIncidents(t *testing.T) {
	cfg := &config.Config{}
	manager := incidents.NewManager(cfg)

	// Start manager
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go manager.Start(ctx)

	// Create active incident
	manager.CreateIncident(
		"Active Incident",
		"Test",
		incidents.SeverityMajor,
		[]string{"API"},
	)

	active := manager.GetActiveIncidents()
	assert.NotNil(t, active)

	// Should have at least the one we created
	hasActive := false
	for _, inc := range active {
		if inc.Title == "Active Incident" {
			hasActive = true
			break
		}
	}
	assert.True(t, hasActive)
}

func TestGetIncidentHistory(t *testing.T) {
	cfg := &config.Config{}
	manager := incidents.NewManager(cfg)

	// Start manager
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go manager.Start(ctx)

	// Wait for historical data to load
	time.Sleep(100 * time.Millisecond)

	history := manager.GetIncidentHistory(10)
	assert.NotNil(t, history)

	// Should have historical data loaded
	assert.NotEmpty(t, history)
}

func TestGetAllIncidents(t *testing.T) {
	cfg := &config.Config{}
	manager := incidents.NewManager(cfg)

	// Start manager
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go manager.Start(ctx)

	// Wait for historical data
	time.Sleep(100 * time.Millisecond)

	all := manager.GetAllIncidents()
	assert.NotNil(t, all)
	assert.Contains(t, all, "active")
	assert.Contains(t, all, "history")
}

func TestSubscribe(t *testing.T) {
	cfg := &config.Config{}
	manager := incidents.NewManager(cfg)

	err := manager.Subscribe("user@domain.com")
	assert.NoError(t, err)

	// Subscribe again (should not error)
	err = manager.Subscribe("user@domain.com")
	assert.NoError(t, err)
}

func TestUnsubscribe(t *testing.T) {
	cfg := &config.Config{}
	manager := incidents.NewManager(cfg)

	// Subscribe first
	manager.Subscribe("user@domain.com")

	// Unsubscribe
	err := manager.Unsubscribe("user@domain.com")
	assert.NoError(t, err)

	// Unsubscribe non-existent
	err = manager.Unsubscribe("nonexistent@domain.com")
	assert.Error(t, err)
}

func TestSeverityValues(t *testing.T) {
	assert.Equal(t, incidents.Severity("critical"), incidents.SeverityCritical)
	assert.Equal(t, incidents.Severity("major"), incidents.SeverityMajor)
	assert.Equal(t, incidents.Severity("minor"), incidents.SeverityMinor)
}

func TestStatusValues(t *testing.T) {
	assert.Equal(t, incidents.IncidentStatus("investigating"), incidents.StatusInvestigating)
	assert.Equal(t, incidents.IncidentStatus("identified"), incidents.StatusIdentified)
	assert.Equal(t, incidents.IncidentStatus("monitoring"), incidents.StatusMonitoring)
	assert.Equal(t, incidents.IncidentStatus("resolved"), incidents.StatusResolved)
}
