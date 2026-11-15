package manparser

import (
	"context"
	"sync"
)

// Progress reports worker-pool completion for a single command.
type Progress struct {
	Command string
	Done    int
	Total   int
	Err     error
	Options []Option
}

// ParseMany runs ParseOptions for commands with a worker pool of size workers (4–8).
// Progress is sent on progress (optional); the call returns when all jobs finish.
func (p *Parser) ParseMany(ctx context.Context, commands []string, workers int, progress chan<- Progress) []Progress {
	if workers < 1 {
		workers = 4
	}
	if workers > 8 {
		workers = 8
	}
	total := len(commands)
	jobs := make(chan string)
	var (
		mu   sync.Mutex
		out  []Progress
		done int
		wg   sync.WaitGroup
	)
	worker := func() {
		defer wg.Done()
		for cmd := range jobs {
			opts, err := p.ParseOptions(ctx, cmd)
			mu.Lock()
			done++
			pr := Progress{Command: cmd, Done: done, Total: total, Err: err, Options: opts}
			out = append(out, pr)
			mu.Unlock()
			if progress != nil {
				select {
				case progress <- pr:
				case <-ctx.Done():
				}
			}
		}
	}
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go worker()
	}
	for _, c := range commands {
		select {
		case jobs <- c:
		case <-ctx.Done():
			close(jobs)
			wg.Wait()
			return out
		}
	}
	close(jobs)
	wg.Wait()
	return out
}

// ParseManyAsync starts ParseMany in a goroutine and returns a progress channel that
// is closed when finished — UI-safe non-blocking API (P1-12).
func (p *Parser) ParseManyAsync(ctx context.Context, commands []string, workers int) <-chan Progress {
	ch := make(chan Progress, len(commands)+1)
	go func() {
		defer close(ch)
		_ = p.ParseMany(ctx, commands, workers, ch)
	}()
	return ch
}
