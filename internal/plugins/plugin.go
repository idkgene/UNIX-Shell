package plugins

import (
	"fmt"
	"path/filepath"
	"plugin"
)

type Plugin interface {
    Init() error
    Name() string
    Execute(args []string) error
}

type Manager struct {
    plugins map[string]Plugin
    pluginsDir string
}

func NewManager(pluginsDir string) *Manager {
    return &Manager{
        plugins: make(map[string]Plugin),
        pluginsDir: pluginsDir,
    }
}

func (m *Manager) LoadPlugins() error {
    files, err := filepath.Glob(filepath.Join(m.pluginsDir, "*.so"))
    if err != nil {
        return err
    }

    for _, file := range files {
        if err := m.loadPlugin(file); err != nil {
            fmt.Printf("Warning: failed to load plugin %s: %v\n", file, err)
            continue
        }
			}
			return nil
	}
	
	func (m *Manager) loadPlugin(path string) error {
			plug, err := plugin.Open(path)
			if err != nil {
					return fmt.Errorf("cannot open plugin: %v", err)
			}
	
			symPlugin, err := plug.Lookup("Plugin")
			if err != nil {
					return fmt.Errorf("plugin does not export 'Plugin' symbol: %v", err)
			}
	
			plugin, ok := symPlugin.(Plugin)
			if !ok {
					return fmt.Errorf("plugin does not implement Plugin interface")
			}
	
			if err := plugin.Init(); err != nil {
					return fmt.Errorf("plugin initialization failed: %v", err)
			}
	
			m.plugins[plugin.Name()] = plugin
			return nil
	}
	
	func (m *Manager) Execute(name string, args []string) error {
			plugin, exists := m.plugins[name]
			if !exists {
					return fmt.Errorf("plugin %s not found", name)
			}
			return plugin.Execute(args)
	}
