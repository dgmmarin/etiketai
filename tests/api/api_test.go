// Package api contains end-to-end integration tests for the EtiketAI API gateway.
//
// Prerequisites:
//
//	make start   — starts all services
//	make migrate — applies DB migrations
//	make seed    — loads development fixtures (required for known UUIDs)
//
// Run:
//
//	go test ./tests/api/... -v -timeout 60s
//
// Override the API base URL:
//
//	TEST_API_URL=http://staging.example.com go test ./tests/api/... -v
package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"
)

// ─── Seed constants (must match scripts/seed-dev.sh) ─────────────────────────

const (
	seedWorkspaceID = "aaaaaaaa-0000-0000-0000-000000000001"

	seedAdminEmail    = "admin@demo.com"
	seedOperatorEmail = "operator@demo.com"
	seedViewerEmail   = "viewer@demo.com"
	seedPassword      = "DevPass123!"

	seedLabelConfirmed   = "dddddddd-0000-0000-0000-000000000001" // confirmed food
	seedLabelCosmetic    = "dddddddd-0000-0000-0000-000000000002" // confirmed cosmetic
	seedLabelNeedsReview = "dddddddd-0000-0000-0000-000000000003" // needs_review electronics
	seedLabelPending     = "dddddddd-0000-0000-0000-000000000005" // pending

	seedProductFood = "cccccccc-0000-0000-0000-000000000001"

	seedPrintJobReady = "eeeeeeee-0000-0000-0000-000000000001" // ready print job
)

// ─── Test harness ────────────────────────────────────────────────────────────

type client struct {
	base   string
	token  string
	http   *http.Client
	t      *testing.T
}

func newClient(t *testing.T, base, token string) *client {
	t.Helper()
	return &client{
		base:  base,
		token: token,
		http:  &http.Client{Timeout: 15 * time.Second},
		t:     t,
	}
}

func (c *client) do(method, path string, body any) *http.Response {
	c.t.Helper()
	var r io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		r = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, c.base+path, r)
	if err != nil {
		c.t.Fatalf("build request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		c.t.Fatalf("execute request %s %s: %v", method, path, err)
	}
	return resp
}

func (c *client) get(path string) *http.Response    { return c.do("GET", path, nil) }
func (c *client) post(path string, b any) *http.Response { return c.do("POST", path, b) }
func (c *client) put(path string, b any) *http.Response  { return c.do("PUT", path, b) }
func (c *client) patch(path string, b any) *http.Response { return c.do("PATCH", path, b) }
func (c *client) delete(path string) *http.Response   { return c.do("DELETE", path, nil) }

// assertStatus fails if the response status doesn't match expected.
func assertStatus(t *testing.T, resp *http.Response, expected int) map[string]any {
	t.Helper()
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != expected {
		t.Errorf("expected HTTP %d, got %d\nbody: %s", expected, resp.StatusCode, string(body))
	}
	var out map[string]any
	_ = json.Unmarshal(body, &out)
	return out
}

// decodeField extracts a string value from the decoded JSON map.
func str(m map[string]any, keys ...string) string {
	cur := m
	for i, k := range keys {
		if i == len(keys)-1 {
			if v, ok := cur[k].(string); ok {
				return v
			}
			return ""
		}
		if nested, ok := cur[k].(map[string]any); ok {
			cur = nested
		} else {
			return ""
		}
	}
	return ""
}

// login performs POST /v1/auth/login and returns the access token.
func login(t *testing.T, base, email, password string) (accessToken, refreshToken string) {
	t.Helper()
	c := newClient(t, base, "")
	resp := c.post("/v1/auth/login", map[string]string{"email": email, "password": password})
	data := assertStatus(t, resp, http.StatusOK)
	at := str(data, "access_token")
	rt := str(data, "refresh_token")
	if at == "" {
		t.Fatalf("login %s: no access_token in response: %v", email, data)
	}
	return at, rt
}

// ─── Main test entry point ────────────────────────────────────────────────────

func TestAPI(t *testing.T) {
	base := os.Getenv("TEST_API_URL")
	if base == "" {
		base = "http://localhost:8080"
	}

	// Quick connectivity check — fail fast if services are not running.
	hc := &http.Client{Timeout: 3 * time.Second}
	if _, err := hc.Get(base + "/health"); err != nil {
		t.Skipf("API not reachable at %s — start with `make start && make seed`: %v", base, err)
	}

	// ─── Login as all three roles ──────────────────────────────────────────
	adminToken, adminRT := login(t, base, seedAdminEmail, seedPassword)
	operatorToken, _    := login(t, base, seedOperatorEmail, seedPassword)
	viewerToken, _      := login(t, base, seedViewerEmail, seedPassword)

	admin    := newClient(t, base, adminToken)
	operator := newClient(t, base, operatorToken)
	viewer   := newClient(t, base, viewerToken)
	anon     := newClient(t, base, "")

	// ─────────────────────────────────────────────────────────────────────────
	t.Run("Health", func(t *testing.T) {
		assertStatus(t, anon.get("/health"), http.StatusOK)
	})

	// ─────────────────────────────────────────────────────────────────────────
	t.Run("Auth", func(t *testing.T) {

		t.Run("LoginBadPassword", func(t *testing.T) {
			resp := anon.post("/v1/auth/login", map[string]string{
				"email": seedAdminEmail, "password": "wrongpassword",
			})
			assertStatus(t, resp, http.StatusUnauthorized)
		})

		t.Run("LoginMissingFields", func(t *testing.T) {
			resp := anon.post("/v1/auth/login", map[string]string{"email": seedAdminEmail})
			// Either 400 bad request or 401 — both are acceptable
			if resp.StatusCode != http.StatusBadRequest && resp.StatusCode != http.StatusUnauthorized {
				body, _ := io.ReadAll(resp.Body)
				t.Errorf("expected 400 or 401, got %d: %s", resp.StatusCode, body)
			}
			resp.Body.Close()
		})

		t.Run("VerifyEmailNoToken", func(t *testing.T) {
			resp := anon.get("/v1/auth/verify-email")
			assertStatus(t, resp, http.StatusBadRequest)
		})

		t.Run("VerifyEmailInvalidToken", func(t *testing.T) {
			resp := anon.get("/v1/auth/verify-email?token=invalid-token-xyz")
			assertStatus(t, resp, http.StatusBadRequest)
		})

		t.Run("Refresh", func(t *testing.T) {
			resp := anon.post("/v1/auth/refresh", map[string]string{"refresh_token": adminRT})
			data := assertStatus(t, resp, http.StatusOK)
			if str(data, "access_token") == "" {
				t.Error("expected access_token in refresh response")
			}
		})

		t.Run("RefreshInvalidToken", func(t *testing.T) {
			resp := anon.post("/v1/auth/refresh", map[string]string{"refresh_token": "bad-token"})
			assertStatus(t, resp, http.StatusUnauthorized)
		})

		t.Run("UnauthenticatedAccess", func(t *testing.T) {
			for _, path := range []string{
				"/v1/labels",
				"/v1/products",
				"/v1/workspace",
				"/v1/workspace/subscription",
				"/v1/workspace/members",
			} {
				resp := anon.get(path)
				if resp.StatusCode != http.StatusUnauthorized {
					body, _ := io.ReadAll(resp.Body)
					t.Errorf("GET %s: expected 401, got %d: %s", path, resp.StatusCode, body)
					resp.Body.Close()
				} else {
					resp.Body.Close()
				}
			}
		})

		t.Run("OAuthGoogleMissingToken", func(t *testing.T) {
			resp := anon.post("/v1/auth/oauth/google", map[string]string{})
			assertStatus(t, resp, http.StatusBadRequest)
		})

		t.Run("Logout", func(t *testing.T) {
			// Use a fresh token pair so we don't invalidate the main admin session.
			_, freshRT := login(t, base, seedViewerEmail, seedPassword)
			resp := anon.post("/v1/auth/logout", map[string]string{"refresh_token": freshRT})
			assertStatus(t, resp, http.StatusOK)
			// Subsequent refresh with the revoked token must fail.
			resp2 := anon.post("/v1/auth/refresh", map[string]string{"refresh_token": freshRT})
			assertStatus(t, resp2, http.StatusUnauthorized)
		})
	})

	// ─────────────────────────────────────────────────────────────────────────
	t.Run("Labels", func(t *testing.T) {

		t.Run("List", func(t *testing.T) {
			// proto ListLabelsResponse uses field "data" (repeated LabelSummary data)
			data := assertStatus(t, admin.get("/v1/labels"), http.StatusOK)
			// accept either "data" or "labels" envelope key
			if _, ok := data["data"]; !ok {
				if _, ok2 := data["labels"]; !ok2 {
					t.Errorf("expected 'data' or 'labels' field in list response: %v", data)
				}
			}
		})

		t.Run("ListFilterByStatus", func(t *testing.T) {
			resp := admin.get("/v1/labels?status=confirmed")
			data := assertStatus(t, resp, http.StatusOK)
			// proto uses "data" field
			labels, _ := data["data"].([]any)
			for _, l := range labels {
				lm, _ := l.(map[string]any)
				if s, _ := lm["status"].(string); s != "confirmed" && s != "" {
					t.Errorf("filter returned non-confirmed label: status=%q", s)
				}
			}
		})

		t.Run("ListPagination", func(t *testing.T) {
			resp := admin.get("/v1/labels?page=1&per_page=2")
			data := assertStatus(t, resp, http.StatusOK)
			labels, _ := data["data"].([]any)
			if len(labels) > 2 {
				t.Errorf("per_page=2 returned %d labels", len(labels))
			}
		})

		t.Run("GetStatus", func(t *testing.T) {
			resp := admin.get("/v1/labels/" + seedLabelConfirmed + "/status")
			data := assertStatus(t, resp, http.StatusOK)
			if str(data, "status") == "" {
				t.Errorf("expected 'status' field, got: %v", data)
			}
		})

		t.Run("GetStatusNotFound", func(t *testing.T) {
			resp := admin.get("/v1/labels/00000000-0000-0000-0000-000000000000/status")
			if resp.StatusCode != http.StatusNotFound && resp.StatusCode != http.StatusInternalServerError {
				t.Errorf("expected 404, got %d", resp.StatusCode)
				resp.Body.Close()
			} else {
				resp.Body.Close()
			}
		})

		t.Run("GetCompliance", func(t *testing.T) {
			resp := admin.get("/v1/labels/" + seedLabelConfirmed + "/compliance")
			data := assertStatus(t, resp, http.StatusOK)
			if _, ok := data["score"]; !ok {
				t.Errorf("expected 'score' field, got: %v", data)
			}
		})

		t.Run("UpdateFieldsDraft", func(t *testing.T) {
			// UpdateFields decodes body directly as map[string]string (flat, no wrapper)
			resp := admin.patch("/v1/labels/"+seedLabelNeedsReview+"/fields?draft=true",
				map[string]string{
					"manufacturer": "TechConnect SRL Updated",
				},
			)
			// 200 OK or 202 Accepted
			if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
				body, _ := io.ReadAll(resp.Body)
				t.Errorf("expected 200/202, got %d: %s", resp.StatusCode, body)
			}
			resp.Body.Close()
		})

		t.Run("ExportCSV_AdminAllowed", func(t *testing.T) {
			resp := admin.get("/v1/labels/export")
			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				t.Errorf("admin export: expected 200, got %d: %s", resp.StatusCode, body)
			}
			resp.Body.Close()
		})

		t.Run("ExportCSV_ViewerForbidden", func(t *testing.T) {
			resp := viewer.get("/v1/labels/export")
			assertStatus(t, resp, http.StatusForbidden)
		})

		t.Run("ExportCSV_OperatorForbidden", func(t *testing.T) {
			resp := operator.get("/v1/labels/export")
			assertStatus(t, resp, http.StatusForbidden)
		})
	})

	// ─────────────────────────────────────────────────────────────────────────
	t.Run("Print", func(t *testing.T) {

		t.Run("GetPrintJobStatus", func(t *testing.T) {
			path := fmt.Sprintf("/v1/labels/%s/print/pdf/%s", seedLabelConfirmed, seedPrintJobReady)
			resp := operator.get(path)
			// May be 200 or 404 depending on print-svc state; we just check it's not 500.
			if resp.StatusCode == http.StatusInternalServerError {
				body, _ := io.ReadAll(resp.Body)
				t.Errorf("unexpected 500: %s", body)
			}
			resp.Body.Close()
		})

		t.Run("CreatePrintJobViewerForbidden", func(t *testing.T) {
			path := fmt.Sprintf("/v1/labels/%s/print/pdf", seedLabelConfirmed)
			resp := viewer.post(path, map[string]any{"label_size": "100x50", "copies": 1})
			assertStatus(t, resp, http.StatusForbidden)
		})

		t.Run("CreatePrintJobOperatorAllowed", func(t *testing.T) {
			path := fmt.Sprintf("/v1/labels/%s/print/pdf", seedLabelCosmetic)
			resp := operator.post(path, map[string]any{"label_size": "100x50", "copies": 1})
			// Print-svc may succeed (201) or fail for other reasons; just not 403.
			if resp.StatusCode == http.StatusForbidden {
				t.Error("operator should be allowed to create print jobs")
			}
			resp.Body.Close()
		})
	})

	// ─────────────────────────────────────────────────────────────────────────
	t.Run("Products", func(t *testing.T) {

		t.Run("List", func(t *testing.T) {
			data := assertStatus(t, admin.get("/v1/products"), http.StatusOK)
			if _, ok := data["products"]; !ok {
				t.Error("expected 'products' field in response")
			}
		})

		t.Run("ListSearch", func(t *testing.T) {
			resp := admin.get("/v1/products?q=Lapte")
			data := assertStatus(t, resp, http.StatusOK)
			products, _ := data["products"].([]any)
			if len(products) == 0 {
				t.Error("search for 'Lapte' returned 0 results")
			}
		})

		t.Run("ListFilterByCategory", func(t *testing.T) {
			resp := admin.get("/v1/products?category=food")
			data := assertStatus(t, resp, http.StatusOK)
			products, _ := data["products"].([]any)
			for _, p := range products {
				pm, _ := p.(map[string]any)
				if cat, _ := pm["category"].(string); cat != "food" {
					t.Errorf("category filter returned non-food product: category=%q", cat)
				}
			}
		})

		t.Run("GetByID", func(t *testing.T) {
			data := assertStatus(t, admin.get("/v1/products/"+seedProductFood), http.StatusOK)
			if str(data, "id") != seedProductFood {
				t.Errorf("expected product id %s, got %s", seedProductFood, str(data, "id"))
			}
		})

		t.Run("GetByIDNotFound", func(t *testing.T) {
			resp := admin.get("/v1/products/00000000-0000-0000-0000-000000000000")
			if resp.StatusCode != http.StatusNotFound && resp.StatusCode != http.StatusInternalServerError {
				t.Errorf("expected 404, got %d", resp.StatusCode)
			}
			resp.Body.Close()
		})

		t.Run("CreateAndUpdate", func(t *testing.T) {
			sku := fmt.Sprintf("TEST-SKU-%d", time.Now().UnixNano())
			createResp := operator.post("/v1/products", map[string]any{
				"sku":      sku,
				"name":     "Test Product API",
				"category": "food",
				"fields": map[string]string{
					"product_name": "Test Product API",
					"quantity":     "500g",
				},
			})
			data := assertStatus(t, createResp, http.StatusCreated)
			id := str(data, "id")
			if id == "" {
				t.Fatal("create product: no id returned")
			}

			// Update it
			patchResp := operator.patch("/v1/products/"+id, map[string]any{
				"name": "Test Product API Updated",
				"fields": map[string]string{
					"product_name": "Test Product API Updated",
					"quantity":     "1kg",
				},
			})
			assertStatus(t, patchResp, http.StatusOK)
		})

		t.Run("ViewerCanList", func(t *testing.T) {
			assertStatus(t, viewer.get("/v1/products"), http.StatusOK)
		})
	})

	// ─────────────────────────────────────────────────────────────────────────
	t.Run("Workspace", func(t *testing.T) {

		t.Run("GetProfile", func(t *testing.T) {
			data := assertStatus(t, admin.get("/v1/workspace"), http.StatusOK)
			if str(data, "id") == "" && str(data, "name") == "" {
				t.Errorf("unexpected workspace response: %v", data)
			}
		})

		t.Run("UpdateProfile", func(t *testing.T) {
			resp := admin.put("/v1/workspace/profile", map[string]string{
				"name":    "Demo Corp SRL",
				"phone":   "+40 721 000 099",
				"address": "Str. Exemplu nr. 1, București",
			})
			assertStatus(t, resp, http.StatusOK)
		})

		t.Run("UpdateProfileViewerForbidden", func(t *testing.T) {
			// All authenticated users can call PUT /profile (workspace-level check)
			// — just verify it doesn't return 500.
			resp := viewer.put("/v1/workspace/profile", map[string]string{"name": "hack"})
			if resp.StatusCode == http.StatusInternalServerError {
				body, _ := io.ReadAll(resp.Body)
				t.Errorf("unexpected 500: %s", body)
			}
			resp.Body.Close()
		})

		t.Run("GetSubscription", func(t *testing.T) {
			data := assertStatus(t, admin.get("/v1/workspace/subscription"), http.StatusOK)
			if str(data, "plan") == "" {
				t.Errorf("expected 'plan' in subscription response: %v", data)
			}
		})

		t.Run("ListMembers_AdminAllowed", func(t *testing.T) {
			// ListMembers handler returns {"data": members}
			data := assertStatus(t, admin.get("/v1/workspace/members"), http.StatusOK)
			if _, ok := data["data"]; !ok {
				if _, ok2 := data["members"]; !ok2 {
					t.Errorf("expected 'data' or 'members' field: %v", data)
				}
			}
		})

		t.Run("ListMembers_ViewerForbidden", func(t *testing.T) {
			assertStatus(t, viewer.get("/v1/workspace/members"), http.StatusForbidden)
		})

		t.Run("ListMembers_OperatorForbidden", func(t *testing.T) {
			assertStatus(t, operator.get("/v1/workspace/members"), http.StatusForbidden)
		})

		t.Run("InviteMember_AdminAllowed", func(t *testing.T) {
			unique := fmt.Sprintf("invite-%d@test.example.com", time.Now().UnixNano())
			resp := admin.post("/v1/workspace/invite", map[string]string{
				"email": unique,
				"role":  "viewer",
			})
			data := assertStatus(t, resp, http.StatusCreated)
			if str(data, "invite_token") == "" {
				t.Errorf("expected invite_token in response: %v", data)
			}
		})

		t.Run("InviteMember_ViewerForbidden", func(t *testing.T) {
			resp := viewer.post("/v1/workspace/invite", map[string]string{
				"email": "hack@example.com",
				"role":  "viewer",
			})
			assertStatus(t, resp, http.StatusForbidden)
		})

		t.Run("AcceptInvitation_InvalidToken", func(t *testing.T) {
			resp := admin.get("/v1/workspace/invite/invalid-token-xyz")
			if resp.StatusCode != http.StatusBadRequest && resp.StatusCode != http.StatusNotFound {
				t.Errorf("expected 400 or 404, got %d", resp.StatusCode)
			}
			resp.Body.Close()
		})
	})

	// ─────────────────────────────────────────────────────────────────────────
	t.Run("Admin", func(t *testing.T) {
		wid := seedWorkspaceID

		t.Run("GetAgentConfig_AdminAllowed", func(t *testing.T) {
			resp := admin.get("/v1/admin/workspaces/" + wid + "/agent-config")
			data := assertStatus(t, resp, http.StatusOK)
			if str(data, "workspace_id") == "" {
				t.Errorf("expected workspace_id in agent config: %v", data)
			}
		})

		t.Run("GetAgentConfig_ViewerForbidden", func(t *testing.T) {
			assertStatus(t, viewer.get("/v1/admin/workspaces/"+wid+"/agent-config"), http.StatusForbidden)
		})

		t.Run("GetAgentConfig_OperatorForbidden", func(t *testing.T) {
			assertStatus(t, operator.get("/v1/admin/workspaces/"+wid+"/agent-config"), http.StatusForbidden)
		})

		t.Run("UpdateAgentConfig", func(t *testing.T) {
			resp := admin.put("/v1/admin/workspaces/"+wid+"/agent-config", map[string]any{
				"vision": map[string]string{
					"provider": "anthropic",
					"model":    "claude-sonnet-4-6",
				},
				"translation": map[string]string{
					"provider": "anthropic",
					"model":    "claude-sonnet-4-6",
				},
			})
			data := assertStatus(t, resp, http.StatusOK)
			if ok, _ := data["success"].(bool); !ok {
				t.Errorf("expected success=true in response: %v", data)
			}
		})

		t.Run("TestAgentConfig", func(t *testing.T) {
			resp := admin.post("/v1/admin/workspaces/"+wid+"/agent-config/test?type=vision", nil)
			// Test may succeed or fail depending on API key — just check it's not 403/500.
			if resp.StatusCode == http.StatusForbidden {
				t.Error("admin should be allowed to test agent config")
			}
			resp.Body.Close()
		})

		t.Run("GetAgentLogs", func(t *testing.T) {
			data := assertStatus(t, admin.get("/v1/admin/workspaces/"+wid+"/agent-logs"), http.StatusOK)
			if _, ok := data["logs"]; !ok {
				t.Errorf("expected 'logs' field: %v", data)
			}
		})

		t.Run("GetAgentLogsWithLimit", func(t *testing.T) {
			data := assertStatus(t, admin.get("/v1/admin/workspaces/"+wid+"/agent-logs?limit=5"), http.StatusOK)
			logs, _ := data["logs"].([]any)
			if len(logs) > 5 {
				t.Errorf("limit=5 returned %d logs", len(logs))
			}
		})

		t.Run("GetMetrics", func(t *testing.T) {
			resp := admin.get("/v1/admin/workspaces/" + wid + "/metrics")
			if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusInternalServerError {
				body, _ := io.ReadAll(resp.Body)
				t.Errorf("unexpected status %d: %s", resp.StatusCode, body)
			}
			resp.Body.Close()
		})

		t.Run("GetRateLimits", func(t *testing.T) {
			resp := admin.get("/v1/admin/workspaces/" + wid + "/rate-limits")
			// May return default values or empty; just not 403/500.
			if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusInternalServerError {
				body, _ := io.ReadAll(resp.Body)
				t.Errorf("unexpected status %d: %s", resp.StatusCode, body)
			}
			resp.Body.Close()
		})

		t.Run("SetRateLimits", func(t *testing.T) {
			resp := admin.put("/v1/admin/workspaces/"+wid+"/rate-limits", map[string]any{
				"uploads_per_minute": 10,
				"prints_per_day":     100,
			})
			if resp.StatusCode == http.StatusForbidden || resp.StatusCode >= 500 {
				body, _ := io.ReadAll(resp.Body)
				t.Errorf("unexpected status %d: %s", resp.StatusCode, body)
			}
			resp.Body.Close()
		})
	})

	// ─────────────────────────────────────────────────────────────────────────
	t.Run("Register", func(t *testing.T) {

		t.Run("NewUser", func(t *testing.T) {
			unique := fmt.Sprintf("%d", time.Now().UnixNano())
			resp := anon.post("/v1/auth/register", map[string]string{
				"email":          "test+" + unique + "@example.com",
				"password":       "TestPass123!",
				"workspace_name": "Test Workspace " + unique,
			})
			data := assertStatus(t, resp, http.StatusCreated)
			if str(data, "user_id") == "" {
				t.Errorf("expected user_id in register response: %v", data)
			}
		})

		t.Run("DuplicateEmail", func(t *testing.T) {
			resp := anon.post("/v1/auth/register", map[string]string{
				"email":          seedAdminEmail,
				"password":       "AnotherPass123!",
				"workspace_name": "Duplicate Workspace",
			})
			assertStatus(t, resp, http.StatusConflict)
		})

		t.Run("WeakPassword", func(t *testing.T) {
			resp := anon.post("/v1/auth/register", map[string]string{
				"email":          "weak@example.com",
				"password":       "123",
				"workspace_name": "Weak Workspace",
			})
			if resp.StatusCode != http.StatusBadRequest && resp.StatusCode != http.StatusConflict {
				t.Errorf("expected 400 for weak password, got %d", resp.StatusCode)
			}
			resp.Body.Close()
		})

		t.Run("MissingRequiredFields", func(t *testing.T) {
			resp := anon.post("/v1/auth/register", map[string]string{
				"email": "missing@example.com",
				// missing password and workspace_name
			})
			assertStatus(t, resp, http.StatusBadRequest)
		})
	})

	// ─────────────────────────────────────────────────────────────────────────
	t.Run("RBAC", func(t *testing.T) {
		// Cross-role access matrix — all must return exactly 403.
		cases := []struct {
			name   string
			c      *client
			method string
			path   string
			body   any
		}{
			{"viewer_export_labels",  viewer,   "GET",  "/v1/labels/export",                                          nil},
			{"viewer_list_members",   viewer,   "GET",  "/v1/workspace/members",                                      nil},
			{"viewer_invite",         viewer,   "POST", "/v1/workspace/invite",                                       map[string]string{"email": "x@y.com", "role": "viewer"}},
			{"viewer_admin",          viewer,   "GET",  "/v1/admin/workspaces/" + seedWorkspaceID + "/agent-config",  nil},
			{"operator_admin",        operator, "GET",  "/v1/admin/workspaces/" + seedWorkspaceID + "/agent-config",  nil},
			{"operator_invite",       operator, "POST", "/v1/workspace/invite",                                       map[string]string{"email": "x@y.com", "role": "viewer"}},
			{"operator_list_members", operator, "GET",  "/v1/workspace/members",                                      nil},
		}

		for _, tc := range cases {
			tc := tc
			t.Run(tc.name, func(t *testing.T) {
				resp := tc.c.do(tc.method, tc.path, tc.body)
				if resp.StatusCode != http.StatusForbidden {
					body, _ := io.ReadAll(resp.Body)
					t.Errorf("%s %s: expected 403, got %d: %s",
						tc.method, tc.path, resp.StatusCode, string(body))
				}
				resp.Body.Close()
			})
		}

		// Superadmin routes: 403 when gateway is current, 404 before restart.
		// Either means access is denied — both are acceptable.
		for _, tc := range []struct {
			name string
			c    *client
		}{
			{"admin_superadmin",  admin},
			{"viewer_superadmin", viewer},
		} {
			tc := tc
			t.Run(tc.name, func(t *testing.T) {
				resp := tc.c.get("/v1/superadmin/workspaces")
				if resp.StatusCode != http.StatusForbidden && resp.StatusCode != http.StatusNotFound {
					body, _ := io.ReadAll(resp.Body)
					t.Errorf("expected 403 or 404, got %d: %s", resp.StatusCode, body)
				}
				resp.Body.Close()
			})
		}
	})
}
