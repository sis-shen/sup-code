package subagent

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/supcode/supcode/internal/agent"
	"github.com/supcode/supcode/internal/contextmgr"
	"github.com/supcode/supcode/pkg"
)

// WorkerPool manages a pool of worker goroutines for parallel subtask execution.
type WorkerPool struct {
	stopped  chan struct{}
	workers  []*Worker
	taskCh   chan pkg.SubTask
	resultCh chan pkg.SubTaskResult
	wg       sync.WaitGroup
	cancel   context.CancelFunc
	ctx      context.Context
}

// Worker executes a single subtask in its own goroutine with an independent LLM client.
type Worker struct {
	id       int
	pool     *WorkerPool
	llmCli   pkg.LLMClient
	toolReg  pkg.ToolRegistry
	wtMgr    pkg.WorktreeManager
	maxIters int
}

// NewWorkerPool creates a pool with the given number of workers.
func NewWorkerPool(numWorkers int, llmFactory func() (pkg.LLMClient, error), toolReg pkg.ToolRegistry, wtMgr pkg.WorktreeManager, maxIters int) (*WorkerPool, error) {
	if numWorkers <= 0 {
		numWorkers = 3
	}
	ctx, cancel := context.WithCancel(context.Background())
	wp := &WorkerPool{
		stopped:  make(chan struct{}),
		taskCh:   make(chan pkg.SubTask, numWorkers*2),
		resultCh: make(chan pkg.SubTaskResult, numWorkers*2),
		cancel:   cancel,
		ctx:      ctx,
	}
	for i := 0; i < numWorkers; i++ {
		llmCli, err := llmFactory()
		if err != nil {
			cancel()
			return nil, fmt.Errorf("create LLM client for worker %d: %w", i, err)
		}
		w := &Worker{
			id: i, pool: wp,
			llmCli: llmCli, toolReg: toolReg,
			wtMgr: wtMgr, maxIters: maxIters,
		}
		wp.workers = append(wp.workers, w)
		wp.wg.Add(1)
		go w.run()
	}
	return wp, nil
}

// Submit adds a task to the pool. Returns false if the pool is stopped.
func (wp *WorkerPool) Submit(task pkg.SubTask) bool {
	select {
	case <-wp.stopped:
		return false
	case wp.taskCh <- task:
		return true
	}
}

// Results returns the result channel.
func (wp *WorkerPool) Results() <-chan pkg.SubTaskResult {
	return wp.resultCh
}

// Stop cancels all workers and waits for them to finish.
func (wp *WorkerPool) Stop() {
	close(wp.stopped)
	close(wp.taskCh)
	wp.cancel()
	wp.wg.Wait()
	close(wp.resultCh)
}

// run is the main loop for a worker goroutine.
func (w *Worker) run() {
	defer w.pool.wg.Done()
	log.Printf("worker %d started", w.id)
	for {
		select {
		case <-w.pool.ctx.Done():
			return
		case task, ok := <-w.pool.taskCh:
			if !ok {
				return
			}
			result := w.executeTask(task)
			select {
			case w.pool.resultCh <- result:
			case <-w.pool.ctx.Done():
				return
			}
		}
	}
}

// executeTask runs a single subtask with its own context, worktree, and agent instance.
func (w *Worker) executeTask(task pkg.SubTask) pkg.SubTaskResult {
	worktreePath, err := w.wtMgr.Create(w.pool.ctx, task.ID, "main")
	if err != nil {
		return pkg.SubTaskResult{TaskID: task.ID, Success: false, Error: fmt.Sprintf("create worktree: %v", err)}
	}
	_ = worktreePath

	taskCtx, taskCancel := context.WithTimeout(w.pool.ctx, 10*time.Minute)
	defer taskCancel()

	ctxMgr := contextmgr.NewManager()
	sessMgr := &simpleSessionManager{}

	a, err := agent.NewAgent(agent.AgentConfig{
		LLMClient:      w.llmCli,
		ToolRegistry:   w.toolReg,
		ContextManager: ctxMgr,
		SessionManager: sessMgr,
		MaxIterations:  w.maxIters,
	})
	if err != nil {
		if abandonErr := w.wtMgr.Abandon(w.pool.ctx, task.ID); abandonErr != nil {
			log.Printf("worker %d: abandon worktree for task %s: %v", w.id, task.ID, abandonErr)
		}
		return pkg.SubTaskResult{TaskID: task.ID, Success: false, Error: fmt.Sprintf("create agent: %v", err)}
	}

	result, err := a.Run(taskCtx, task.ID, task.Context)
	if err != nil {
		if abandonErr := w.wtMgr.Abandon(w.pool.ctx, task.ID); abandonErr != nil {
			log.Printf("worker %d: abandon worktree for task %s: %v", w.id, task.ID, abandonErr)
		}
		if taskCtx.Err() == context.DeadlineExceeded {
			return pkg.SubTaskResult{TaskID: task.ID, Success: false, Error: "timeout"}
		}
		errMsg := result.Error
		if errMsg == "" {
			errMsg = err.Error()
		}
		return pkg.SubTaskResult{TaskID: task.ID, Success: false, Error: errMsg}
	}

	mergeErr := w.wtMgr.Merge(w.pool.ctx, task.ID)
	if mergeErr != nil {
		log.Printf("worker %d: merge worktree for task %s: %v", w.id, task.ID, mergeErr)
	}

	return pkg.SubTaskResult{
		TaskID:  task.ID,
		Success: true,
		Output:  result.Summary,
	}
}

// simpleSessionManager implements pkg.SessionManager with in-memory storage.
type simpleSessionManager struct {
	mu       sync.RWMutex
	sessions map[string]*pkg.Session
}

func (s *simpleSessionManager) Create(ctx context.Context, title string) (*pkg.Session, error) {
	return &pkg.Session{ID: "session-" + title, Title: title}, nil
}
func (s *simpleSessionManager) Get(ctx context.Context, sessionID string) (*pkg.Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.sessions == nil {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}
	if sess, ok := s.sessions[sessionID]; ok {
		return sess, nil
	}
	return nil, fmt.Errorf("session not found: %s", sessionID)
}
func (s *simpleSessionManager) List(ctx context.Context) ([]*pkg.Session, error) { return nil, nil }
func (s *simpleSessionManager) AppendMessage(ctx context.Context, sessionID string, msg pkg.Message) error {
	return nil
}
func (s *simpleSessionManager) SetState(ctx context.Context, sessionID string, state pkg.LoopState) error {
	return nil
}
func (s *simpleSessionManager) SetPlan(ctx context.Context, sessionID string, plan pkg.Plan) error {
	return nil
}
func (s *simpleSessionManager) Delete(ctx context.Context, sessionID string) error { return nil }
func (s *simpleSessionManager) Close(ctx context.Context, sessionID string) error  { return nil }
