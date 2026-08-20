package middleware

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
)

func RateLimitFiber(limiter *RateLimiter, extractor ...KeyExtractor) fiber.Handler {
	// usa extractor custom ou default
	keyFn := TenantKeyExtractor
	if len(extractor) > 0 && extractor[0] != nil {
		keyFn = extractor[0]
	}

	return func(c *fiber.Ctx) error {
		if c.Path() == "/health" || c.Path() == "/ready" || c.Path() == "/metrics" || c.Path() == "/status" {
			return c.Next()
		}
		key := keyFn(c)
		if ok, retryAfter := limiter.Allow(key); !ok {
			c.Set("Retry-After", strconv.FormatInt(retryAfter, 10))
			return c.Status(429).JSON(fiber.Map{
				"error":       "rate limit exceeded",
				"retry_after": retryAfter,
				"key":         key,
			})
		}
		return c.Next()
	}
}
