package job

import (
    "sync"
)

type Status int

const (
    StatusRunning Status = iota
    StatusStopped
    StatusDone
)

type Job struct {
    Command     string
    Pid         int
    Status      Status
    Background  bool
    ProcessGroup int
}

type Manager struct {
    jobs     map[int]*Job
    mu       sync.RWMutex
    nextJobId int
}

func NewManager() *Manager {
    return &Manager{
        jobs: make(map[int]*Job),
        nextJobId: 1,
    }
}

func (m *Manager) Add(command string, pid int, background bool) int {
    m.mu.Lock()
    defer m.mu.Unlock()

    jobId := m.nextJobId
    m.nextJobId++

    m.jobs[jobId] = &Job{
        Command:    command,
        Pid:        pid,
        Status:     StatusRunning,
        Background: background,
    }

    return jobId
}

func (m *Manager) Get(jobId int) (*Job, bool) {
    m.mu.RLock()
    defer m.mu.RUnlock()
    
		job, exists := m.jobs[jobId]
    return job, exists
}

func (m *Manager) UpdateStatus(jobId int, status Status) {
    m.mu.Lock()
    defer m.mu.Unlock()
    
    if job, exists := m.jobs[jobId]; exists {
        job.Status = status
    }
}
