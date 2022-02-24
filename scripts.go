package darkman

import (
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/adrg/xdg"
)

var scriptsRunning sync.Mutex

/// Run transition scripts for a given mode.
///
/// Fires up all scripts asyncrhonously and returns immediately.
func RunScripts(mode Mode) {
	executables := make(map[string]string)
	directories := make([]string, len(xdg.DataDirs)+1)

	copy(directories, xdg.DataDirs)
	directories[len(directories)-1] = xdg.DataHome

	for _, dir := range directories {
		modeDir := filepath.Join(dir, fmt.Sprintf("%v-mode.d", mode))

		files, err := os.ReadDir(modeDir)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}

		if err != nil {
			log.Println(err.Error())
			log.Printf("Error reading entries in %v: %v.\n", dir, err)
		}

		for _, file := range files {
			filePath := fmt.Sprintf("%v/%v", modeDir, file.Name())
			log.Printf("Found %v.", filePath)
			executables[file.Name()] = filePath
		}
	}

	go func() {
		scriptsRunning.Lock()
		defer scriptsRunning.Unlock()

		for _, executable := range executables {
			log.Printf("Running %v...", executable)

			cmd := exec.Command("bash", "-c", executable)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr

			err := cmd.Run()
			if err != nil {
				log.Printf("Failed to run: %v.\n", err.Error())
			}
		}
	}()
}