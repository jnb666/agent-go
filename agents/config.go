package agents

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/jnb666/agent-go/llm"
	"github.com/jnb666/agent-go/util"
)

// Agent configuration data.
type Config struct {
	Model        string                          `json:"model"`
	Models       map[string]llm.GenerationConfig `json:"models"`
	SystemPrompt string                          `json:"system_prompt"`
	Tools        []ToolConfig                    `json:"tools"`
}

type ToolConfig struct {
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
}

// Load saved configuration. Reads util.ConfigDir and current dir for config_<server>.json
func LoadConfig(server string, cfg *Config) error {
	var errs []error
	filename := "config_" + server + ".json"
	err := util.LoadJSON(filepath.Join(util.ConfigDir, filename), cfg)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		errs = append(errs, fmt.Errorf("error loading %s config from %s: %w", server, util.ConfigDir, err))
	}
	if dir, err := os.Getwd(); err == nil {
		err = util.LoadJSON(filepath.Join(dir, filename), cfg)
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			errs = append(errs, fmt.Errorf("error loading %s config from %s: %w", server, dir, err))
		}
	}
	return errors.Join(errs...)
}
