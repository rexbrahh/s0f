package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// IPCConfig defines socket / named pipe settings.
type IPCConfig struct {
	SocketPath   string `toml:"socketPath"`
	RequireToken bool   `toml:"requireToken"`
	TokenRef     string `toml:"tokenRef"`
}

// StorageConfig defines SQLite tuning options.
type StorageConfig struct {
	DBPath      string `toml:"dbPath"`
	JournalMode string `toml:"journalMode"`
	Synchronous string `toml:"synchronous"`
}

// VCSRemote config.
type VCSRemote struct {
	URL           string `toml:"url"`
	CredentialRef string `toml:"credentialRef"`
}

// VCSConfig defines Git options.
type VCSConfig struct {
	Enabled  bool      `toml:"enabled"`
	Branch   string    `toml:"branch"`
	AutoPush bool      `toml:"autoPush"`
	Remote   VCSRemote `toml:"remote"`
}

// LoggingConfig defines basic logging knobs.
type LoggingConfig struct {
	Level       string `toml:"level"`
	FilePath    string `toml:"filePath"`
	FileMaxSize int    `toml:"fileMaxSizeMB"`
	FileBackups int    `toml:"fileMaxBackups"`
}

// ProfileConfig aggregates service configuration for a profile.
type ProfileConfig struct {
	ProfileName string        `toml:"profileName"`
	Storage     StorageConfig `toml:"storage"`
	VCS         VCSConfig     `toml:"vcs"`
	IPC         IPCConfig     `toml:"ipc"`
	Logging     LoggingConfig `toml:"logging"`
}

// Load reads config.toml from the provided path.
func Load(path string) (*ProfileConfig, error) {
	var cfg ProfileConfig
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	cfg.applyDefaults()
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// LoadProfile loads config.toml residing under the profile directory.
func LoadProfile(profileDir string) (*ProfileConfig, error) {
	return Load(filepath.Join(profileDir, "config.toml"))
}

// Save writes the configuration to disk.
func Save(path string, cfg *ProfileConfig) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	cfg.applyDefaults()
	if err := cfg.validate(); err != nil {
		return err
	}
	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(cfg); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0o600)
}

// DefaultProfile returns a baseline configuration for a new profile.
func DefaultProfile(name string) *ProfileConfig {
	return &ProfileConfig{
		ProfileName: name,
		Storage: StorageConfig{
			DBPath:      "state.db",
			JournalMode: "DELETE",
			Synchronous: "FULL",
		},
		VCS: VCSConfig{
			Enabled:  false,
			Branch:   "main",
			AutoPush: false,
		},
		IPC: IPCConfig{
			SocketPath: "ipc.sock",
		},
	}
}

// ResolvePath joins base and p when p is relative.
func ResolvePath(base, p string) string {
	if p == "" {
		return ""
	}
	if filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(base, p)
}

func (cfg *ProfileConfig) applyDefaults() {
	if cfg.Storage.JournalMode == "" {
		cfg.Storage.JournalMode = "DELETE"
	}
	if cfg.Storage.Synchronous == "" {
		cfg.Storage.Synchronous = "FULL"
	}
	if cfg.IPC.SocketPath == "" {
		cfg.IPC.SocketPath = "ipc.sock"
	}
	if cfg.VCS.Branch == "" {
		cfg.VCS.Branch = "main"
	}
}

func (cfg *ProfileConfig) validate() error {
	if cfg.ProfileName == "" {
		return fmt.Errorf("profileName required")
	}
	if cfg.Storage.DBPath == "" {
		return fmt.Errorf("storage.dbPath required")
	}
	if cfg.IPC.SocketPath == "" {
		return fmt.Errorf("ipc.socketPath required")
	}
	return nil
}
