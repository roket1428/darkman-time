package darkman

import (
	"os"
	"testing"
)

func TestLoadFromYaml(t *testing.T) {
	dir := t.TempDir()
	testPath := dir + "/test1.yaml"
	if err := os.WriteFile(testPath, []byte("usegeoclue: true\n"), 0666); err != nil {
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

	testfile, err := os.Open(testPath)
	if err != nil {
		t.Fatal("failed to open just-created configuration file:", err)
	}
	if err := config.LoadFromYaml(testfile); err != nil {
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

func TestHashing(t *testing.T) {
	lat := 12.12
	lng := 14.14
	config1 := Config{
		Lat:        &lat,
		Lng:        &lng,
		UseGeoclue: false,
		DBusServer: true,
		Portal:     true,
	}

	lng2 := 14.14
	config2 := Config{
		Lat:        &lat,
		Lng:        &lng2,
		UseGeoclue: false,
		DBusServer: true,
		Portal:     true,
	}

	hash1, err := config1.Hash()
	if err != nil {
		t.Fatal("error calculating hash for config1:", err)
	}

	hash2, err := config2.Hash()
	if err != nil {
		t.Fatal("error calculating hash for config2:", err)
	}

	if hash1 != hash2 {
		t.Error("hash for identical configs does not match")
	}

	config3 := Config{
		Lat:        &lat,
		Lng:        &lng2,
		UseGeoclue: true, // This one is different
		DBusServer: true,
		Portal:     true,
	}

	hash3, err := config3.Hash()
	if err != nil {
		t.Fatal("error calculating hash for config3:", err)
	}

	if hash1 == hash3 {
		t.Error("hash for different configs is the same")
	}
}
