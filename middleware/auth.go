package middleware

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"

	"ctpcharts/config"
	"github.com/gofiber/fiber/v3"
)

type sessionResponse struct {
	CID   string   `json:"cid"`
	Roles []string `json:"roles"`
}

type sessionError struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

func validateSession(c fiber.Ctx) (*sessionResponse, error) {
	sessionCookie := c.Cookies("session_id")
	if sessionCookie == "" {
		return nil, fmt.Errorf("no session_id cookie")
	}

	req, err := http.NewRequest(http.MethodGet, config.C.SSOBaseURL+"/internal/session/validate", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("X-Internal-Key", config.C.InternalAPIKey)
	req.Header.Set("Cookie", "session_id="+sessionCookie)
	req.Header.Set("User-Agent", c.Get("User-Agent"))

	if xff := c.Get("X-Forwarded-For"); xff != "" {
		req.Header.Set("X-Forwarded-For", xff)
	} else {
		req.Header.Set("X-Forwarded-For", c.IP())
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		var ssoErr sessionError
		json.Unmarshal(body, &ssoErr)
		return nil, fmt.Errorf("session invalid: %s", ssoErr.Error)
	}

	var session sessionResponse
	if err := json.Unmarshal(body, &session); err != nil {
		return nil, err
	}

	return &session, nil
}

func RequireRole(allowed ...string) fiber.Handler {
	return func(c fiber.Ctx) error {
		rolesVal := c.Locals("roles")
		roles, _ := rolesVal.([]string)
		for _, r := range allowed {
			if slices.Contains(roles, r) {
				return c.Next()
			}
		}
		return c.Status(fiber.StatusForbidden).SendString("403 Forbidden")
	}
}

func RequireAuth(c fiber.Ctx) error {
	session, err := validateSession(c)
	if err != nil {
		return c.Redirect().To(config.C.SSOLoginURL)
	}

	requiredRole := config.C.ChartsRequiredRole
	if requiredRole != "" {
		if !slices.Contains(session.Roles, requiredRole) {
			return c.Status(fiber.StatusForbidden).SendString("403 Forbidden")
		}
	}

	c.Locals("cid", session.CID)
	c.Locals("roles", session.Roles)

	return c.Next()
}
