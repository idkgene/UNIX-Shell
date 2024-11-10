package history

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"
)

type Manager struct {
	entries    []string
	filePath   string
	maxEntries int
	mu         sync.RWMutex
}

func NewManager(filePath string) (*Manager, error) {
	m := &Manager{
			filePath:   filePath,
			maxEntries: 1000,
	}

	if err := m.load(); err != nil {
			return nil, err
	}

	return m, nil
}

func (m *Manager) Add(entry string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	entry = strings.TrimSpace(entry)
	if entry == "" || (len(m.entries) > 0 && m.entries[len(m.entries)-1] == entry) {
			return
	}

	m.entries = append(m.entries, entry)

	if len(m.entries) > m.maxEntries {
			m.entries = m.entries[len(m.entries)-m.maxEntries:]
	}

	go m.Save()
}

func (m *Manager) Get(index int) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if index < 0 || index >= len(m.entries) {
			return "", fmt.Errorf("history index out of range")
	}

	return m.entries[index], nil
}

func (m *Manager) GetAll() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	result := make([]string, len(m.entries))
	copy(result, m.entries)
	return result
}

func (m *Manager) Search(prefix string) []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var results []string
	for i := len(m.entries) - 1; i >= 0; i-- {
			if strings.HasPrefix(m.entries[i], prefix) {
					results = append(results, m.entries[i])
			}
	}
	return results
}

func (m *Manager) load() error {
	file, err := os.OpenFile(m.filePath, os.O_CREATE|os.O_RDONLY, 0600)
	if err != nil {
			return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
			entry := strings.TrimSpace(scanner.Text())
			if entry != "" {
					m.entries = append(m.entries, entry)
			}
	}

	return scanner.Err()
}

func (m *Manager) Save() error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	file, err := os.OpenFile(m.filePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
			return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	for _, entry := range m.entries {
			if _, err := writer.WriteString(entry + "\n"); err != nil {
					return err
			}
	}

	return writer.Flush()
}
