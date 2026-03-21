package router

import (
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/dgmmarin/etiketai/services/api-gateway/internal/handlers"
	"github.com/dgmmarin/etiketai/services/api-gateway/internal/middleware"
	"github.com/dgmmarin/etiketai/services/api-gateway/internal/proxy"
)

// New builds the full HTTP router for the API gateway.
func New(clients *proxy.Clients, s3cfg handlers.S3Config, mwCfg middleware.Config, logger *zap.Logger) http.Handler {
	r := chi.NewRouter()

	// ─── Global middleware ────────────────────────────────────────────────────
	r.Use(chiMiddleware.RequestID)
	r.Use(chiMiddleware.RealIP)
	r.Use(chiMiddleware.Recoverer)
	r.Use(chiMiddleware.Timeout(120 * time.Second))
	r.Use(corsMiddleware())

	// ─── Handlers ─────────────────────────────────────────────────────────────
	authH        := handlers.NewAuthHandler(clients.Auth, clients.Workspace, logger)
	labelH       := handlers.NewLabelHandler(clients.Label, s3cfg, logger)
	workspaceH   := handlers.NewWorkspaceHandler(clients.Workspace, logger)
	adminH       := handlers.NewAdminHandler(clients.Agent, logger)
	superAdminH  := handlers.NewSuperAdminHandler(clients.Billing, logger)
	printH       := handlers.NewPrintHandler(clients.Print, clients.Label, logger)
	productH     := handlers.NewProductHandler(clients.Product, logger)
	billingH     := handlers.NewBillingHandler(clients.Billing, logger)

	// ─── Public routes ────────────────────────────────────────────────────────
	r.Get("/health", handlers.Health)

	r.Route("/v1", func(r chi.Router) {
		// Auth — public (no JWT required)
		r.Route("/auth", func(r chi.Router) {
			loginRL := middleware.RateLimit(
				newRedisFromEnv(),
				middleware.RateLimitConfig{Max: 5, Window: 15 * time.Minute},
				middleware.IPKey,
			)
			r.With(loginRL).Post("/login", authH.Login)
			r.Post("/register", authH.Register)
			r.Post("/refresh", authH.Refresh)
			r.Post("/logout", authH.Logout)
			r.Get("/verify-email", authH.VerifyEmail)
			r.Post("/oauth/google", authH.OAuthGoogle)
		})

		// ─── Protected routes (JWT required) ─────────────────────────────────
		r.Group(func(r chi.Router) {
			r.Use(middleware.Auth(mwCfg, logger))

			// Products (saved label library)
			r.Route("/products", func(r chi.Router) {
				r.Get("/", productH.List)
				r.Post("/", productH.Create)
				r.Get("/{id}", productH.Get)
				r.Patch("/{id}", productH.Update)
			})

			// Labels
			r.Route("/labels", func(r chi.Router) {
				r.Get("/", labelH.List)
				r.With(middleware.RequireRole("admin")).Get("/export", labelH.Export)
				r.Post("/upload", labelH.Upload)
				r.Get("/{id}/status", labelH.GetStatus)
				r.Get("/{id}/compliance", labelH.GetCompliance)
				r.Patch("/{id}/fields", labelH.UpdateFields)
				r.Post("/{id}/confirm", labelH.Confirm)
				r.Delete("/{id}", labelH.Delete)
				r.With(middleware.RequireRole("operator")).Post("/{id}/print/pdf", printH.CreatePrintJob)
				r.With(middleware.RequireRole("operator")).Get("/{id}/print/pdf/{job_id}", printH.GetPrintJob)
				r.With(middleware.RequireRole("operator")).Get("/{id}/print/pdf/{job_id}/url", printH.GetReprintURL)
			r.With(middleware.RequireRole("operator")).Get("/{id}/print/pdf/{job_id}/stream", printH.StreamPrintStatus)
			})

			// Workspace — profile + subscription (any authenticated user)
			r.Route("/workspace", func(r chi.Router) {
				r.Get("/", workspaceH.GetProfile)
				r.Put("/profile", workspaceH.UpdateProfile)
				r.Get("/subscription", workspaceH.GetSubscription)
				// Member management (admin only)
				r.With(middleware.RequireRole("admin")).Post("/invite", workspaceH.InviteMember)
				r.Get("/invite/{token}", workspaceH.AcceptInvitation)
				r.With(middleware.RequireRole("admin")).Get("/members", workspaceH.ListMembers)
				r.With(middleware.RequireRole("admin")).Delete("/members/{id}", workspaceH.RevokeMember)
			})

			// Billing (admin only)
			r.Route("/billing", func(r chi.Router) {
				r.With(middleware.RequireRole("admin")).Post("/create-checkout", billingH.CreateCheckout)
			})

			// Admin — agent config per tenant (admin only)
			r.With(middleware.RequireRole("admin")).Route("/admin", func(r chi.Router) {
				r.Get("/workspaces/{id}/agent-config", adminH.GetAgentConfig)
				r.Put("/workspaces/{id}/agent-config", adminH.UpdateAgentConfig)
				r.Post("/workspaces/{id}/agent-config/test", adminH.TestAgentConfig)
				r.Get("/workspaces/{id}/agent-logs", adminH.GetAgentLogs)
				r.Get("/workspaces/{id}/metrics", adminH.GetMetrics)
				r.Get("/workspaces/{id}/rate-limits", adminH.GetRateLimits)
				r.Put("/workspaces/{id}/rate-limits", adminH.SetRateLimits)
			})

			// Superadmin — platform-level access (is_superadmin JWT claim required)
			r.With(middleware.RequireSuperAdmin()).Route("/superadmin", func(r chi.Router) {
				r.Get("/workspaces", superAdminH.ListWorkspaces)
				r.Get("/workspaces/{id}", superAdminH.GetWorkspace)
			})
		})
	})

	return r
}

func corsMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Request-ID")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func newRedisFromEnv() *redis.Client {
	url := "redis://localhost:6379"
	if v := os.Getenv("REDIS_URL"); v != "" {
		url = v
	}
	opts, _ := redis.ParseURL(url)
	return redis.NewClient(opts)
}
