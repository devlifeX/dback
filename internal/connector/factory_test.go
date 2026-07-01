package connector

import (
	"testing"

	"dback/models"
)

func TestNewConnectorLocalhost(t *testing.T) {
	p := models.Profile{ConnectionType: models.ConnectionTypeLocalhost}
	c, err := NewConnector(p)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
}

func TestNewConnectorWordPressUnsupported(t *testing.T) {
	p := models.Profile{ConnectionType: models.ConnectionTypeWordPress}
	_, err := NewConnector(p)
	if err == nil {
		t.Fatal("expected error for wordpress")
	}
}
