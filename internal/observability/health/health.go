package health

import (
	"context"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
)

type CheckFunc func(ctx context.Context) error

type Check struct {
	Name     string
	Check    CheckFunc
	Critical bool // true = derruba /ready
}

type HealthChecker struct {
	timeout time.Duration
	checks  []Check
}

func NewHealthChecker(timeout time.Duration, checks ...Check) *HealthChecker {
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	return &HealthChecker{timeout: timeout, checks: checks}
}

type checkResult struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

func (h *HealthChecker) LiveFiber(c *fiber.Ctx) error {
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status": "ok",
	})
}

func (h *HealthChecker) ReadyFiber(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.Context(), h.timeout)
	defer cancel()

	results := make([]checkResult, len(h.checks))
	var wg sync.WaitGroup
	var mu sync.Mutex
	ready := true

	for i, chk := range h.checks {
		wg.Add(1)
		go func(i int, chk Check) {
			defer wg.Done()
			r := checkResult{Name: chk.Name, Status: "ok"}
			if err := chk.Check(ctx); err != nil {
				r.Status = "fail"
				r.Error = err.Error()
				if chk.Critical {
					mu.Lock()
					ready = false
					mu.Unlock()
				}
			}
			results[i] = r
		}(i, chk)
	}
	wg.Wait()

	status := fiber.StatusOK
	state := "ready"
	if !ready {
		status = fiber.StatusServiceUnavailable
		state = "not_ready"
	}
	return c.Status(status).JSON(fiber.Map{
		"status": state,
		"checks": results,
	})
}
