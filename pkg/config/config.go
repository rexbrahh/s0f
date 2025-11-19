package config

// IPCConfig defines socket / named pipe settings.
type IPCConfig struct {
    SocketPath  string `json:"socketPath"`
    RequireToken bool   `json:"requireToken"`
    TokenRef    string `json:"tokenRef,omitempty"`
}

// StorageConfig defines SQLite tuning options.
type StorageConfig struct {
    DBPath      string `json:"dbPath"`
    JournalMode string `json:"journalMode"`
    Synchronous string `json:"synchronous"`
}

// VCSConfig defines Git options.
type VCSConfig struct {
    Enabled bool   `json:"enabled"`
    Remote  string `json:"remote,omitempty"`
    Branch  string `json:"branch"`
}

// ProfileConfig aggregates service configuration for a profile.
type ProfileConfig struct {
    ProfileName string        `json:"profileName"`
    Storage     StorageConfig `json:"storage"`
    VCS         VCSConfig     `json:"vcs"`
    IPC         IPCConfig     `json:"ipc"`
}
