package subagent

import (
    "context"
    "fmt"
    "sync"

    "github.com/supcode/supcode/pkg"
)

// Orchestrator implements pkg.SubAgentOrchestrator with a pool of workers.
type Orchestrator struct {
    llmFactory   func() (pkg.LLMClient, error)
    toolRegistry pkg.ToolRegistry
    worktreeMgr  pkg.WorktreeManager
    maxWorkers   int
    maxIters     int
    pool         *WorkerPool
    mu           sync.Mutex
    started      bool
}

// NewOrchestrator creates a new Orchestrator.
// llmFactory is called once per worker to create independent LLM connections.
func NewOrchestrator(llmFactory func() (pkg.LLMClient, error), toolRegistry pkg.ToolRegistry, worktreeManager pkg.WorktreeManager, maxWorkers int) *Orchestrator {
    if maxWorkers <= 0 {
        maxWorkers = 3
    }
    return &Orchestrator{
        llmFactory:   llmFactory,
        toolRegistry: toolRegistry,
        worktreeMgr:  worktreeManager,
        maxWorkers:   maxWorkers,
        maxIters:     25,
    }
}

// Dispatch sends subtasks to the worker pool and collects results.
func (o *Orchestrator) Dispatch(ctx context.Context, tasks []pkg.SubTask) ([]pkg.SubTaskResult, error) {
    o.mu.Lock()
    if !o.started {
        pool, err := NewWorkerPool(o.maxWorkers, o.llmFactory, o.toolRegistry, o.worktreeMgr, o.maxIters)
        if err != nil {
            o.mu.Unlock()
            return nil, fmt.Errorf("start worker pool: %w", err)
        }
        o.pool = pool
        o.started = true
    }
    pool := o.pool
    o.mu.Unlock()

    // Push tasks
    go func() {
        for _, task := range tasks {
            if !pool.Submit(task) {
                break
            }
        }
    }()

    // Collect results
    var results []pkg.SubTaskResult
    remaining := len(tasks)

    for remaining > 0 {
        select {
        case <-ctx.Done():
            return results, ctx.Err()
        case result, ok := <-pool.Results():
            if !ok {
                return results, fmt.Errorf("pool closed unexpectedly")
            }
            results = append(results, result)
            remaining--
        }
    }

    // Check for partial failures
    var errs []error
    for _, r := range results {
        if !r.Success {
            errs = append(errs, fmt.Errorf("task %s: %s", r.TaskID, r.Error))
        }
    }
    if len(errs) > 0 {
        return results, fmt.Errorf("%d/%d tasks failed: %v", len(errs), len(tasks), errs[0])
    }

    return results, nil
}

// CancelAll stops all workers and cleans up.
func (o *Orchestrator) CancelAll() {
    o.mu.Lock()
    defer o.mu.Unlock()
    if o.pool != nil {
        o.pool.Stop()
        o.pool = nil
        o.started = false
    }
}
