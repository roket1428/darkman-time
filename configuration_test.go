package darkman

import (
	"os"
	"testing"
)

func TestLoadFromYaml(t *testing.T) {
	dir := t.TempDir()
	testfile := dir + "/test1.yaml"
	if err := os.WriteFile(testfile, []byte("usegeoclue: true\n"), 0666); err != nil {
		t.Fatal("failed to write test file:", err)
	}

	// Check that loading from a file doesn't overwrite previous values
	lat := 1.2
	lng := 2.4
	config := &Config{
		Lat:        &lat,
		Lng:        &lng,
		UseGeoclue: true,
		DBusServer: true,
		Portal:     true,
	}
	if err := config.LoadFromYamlFile(testfile); err != nil {
		t.Fatal("failed to read configuration file:", err)
	}

	if *config.Lat != 1.2 {
		t.Errorf("config.Lat want=1.2, got=%e", *config.Lat)
	}
	if *config.Lng != 2.4 {
		t.Errorf("config.Lng want=2.4, got=%e", *config.Lng)
	}
	if config.UseGeoclue != true {
		t.Errorf("config.UseGeoclue want=1.2, got=%t", config.UseGeoclue)
	}
	if config.DBusServer != true {
		t.Errorf("config.LDBusServerat want=1.2, got=%t", config.DBusServer)
	}
	if config.Portal != true {
		t.Errorf("config.Portal want=1.2, got=%t", config.Portal)
	}
}
