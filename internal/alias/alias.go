package alias

import (
	"encoding/json"
	"os"
	"sync"
)

type Manager struct {
    aliases map[string]string
    file    string
    mu      sync.RWMutex
}

func NewManager(file string) *Manager {
    m := &Manager{
        aliases: make(map[string]string),
        file:    file,
    }
    m.Load()
    return m
}

func (m *Manager) Add(name, command string) error {
    m.mu.Lock()
    defer m.mu.Unlock()

    m.aliases[name] = command
    return m.Save()
}

func (m *Manager) GetAll() map[string]string {
    m.mu.RLock()
    defer m.mu.RUnlock()

    return m.aliases
}

func (m *Manager) Load() error {
    m.mu.Lock()
    defer m.mu.Unlock()

    data, err := os.ReadFile(m.file)
    if err != nil {
        if os.IsNotExist(err) {
            return nil
        }
        return err
    }

    return json.Unmarshal(data, &m.aliases)
}

func (m *Manager) Save() error {
    m.mu.RLock()
    defer m.mu.RUnlock()

    data, err := json.MarshalIndent(m.aliases, "", "    ")
    if err != nil {
        return err
    }

    return os.WriteFile(m.file, data, 0644)
}
