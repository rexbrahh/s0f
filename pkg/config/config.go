package config

import (
    "fmt"
    "os"

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
    URL          string `toml:"url"`
    CredentialRef string `toml:"credentialRef"`
}

// VCSConfig defines Git options.
type VCSConfig struct {
    Enabled bool      `toml:"enabled"`
    Branch  string    `toml:"branch"`
    AutoPush bool     `toml:"autoPush"`
    Remote  VCSRemote `toml:"remote"`
}

// LoggingConfig defines basic logging knobs.
type LoggingConfig struct {
    Level        string `toml:"level"`
    FileMaxSize  int    `toml:"fileMaxSizeMB"`
    FileBackups  int    `toml:"fileMaxBackups"`
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
    if err := cfg.validate(); err != nil {
        return nil, err
    }
    return &cfg, nil
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
    if cfg.VCS.Branch == "" {
        cfg.VCS.Branch = "main"
    }
    return nil
}
