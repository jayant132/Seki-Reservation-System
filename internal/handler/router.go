package handler

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"

	"github.com/jayant132/seki/internal/auth"
	custommw "github.com/jayant132/seki/internal/middleware"
	"github.com/jayant132/seki/internal/repository"
	"github.com/jayant132/seki/internal/service"
)

type Dependencies struct {
	Pool   *pgxpool.Pool
	Redis  *redis.Client
	Issuer *auth.Issuer
	Logger *slog.Logger
}

func NewRouter(deps Dependencies) http.Handler {
	r := chi.NewRouter()

	r.Use(chimw.RequestID)
	r.Use(chimw.Recoverer)
	r.Use(chimw.Timeout(30 * time.Second))
	r.Use(custommw.Tracing("seki-api"))
	r.Use(custommw.RequestLogger(deps.Logger))

	// --- wire repositories -> services -> handlers ---
	userRepo := repository.NewUserRepository(deps.Pool)
	resourceRepo := repository.NewResourceRepository(deps.Pool)
	bookingRepo := repository.NewBookingRepository(deps.Pool)
	auditRepo := repository.NewAuditRepository(deps.Pool)

	authSvc := service.NewAuthService(userRepo, deps.Issuer)
	resourceSvc := service.NewResourceService(resourceRepo)
	bookingSvc := service.NewBookingService(bookingRepo, resourceRepo)

	authH := NewAuthHandler(authSvc, deps.Logger)
	resourceH := NewResourceHandler(resourceSvc)
	bookingH := NewBookingHandler(bookingSvc, deps.Logger)
	auditH := NewAuditHandler(auditRepo)
	statusH := NewSystemStatusHandler(deps.Pool, deps.Redis)

	// --- unauthenticated ---
	r.Get("/status/live", statusH.Live)
	r.Get("/status/ready", statusH.Ready)
	r.Handle("/metrics", promhttp.Handler())

	r.Route("/api/v1", func(api chi.Router) {
		api.Post("/auth/register", authH.Register)
		api.Post("/auth/login", authH.Login)

		// Public read of resources + availability so a front-end can show
		// a calendar before the user logs in.
		api.Get("/resources", resourceH.List)
		api.Get("/resources/{id}/availability", bookingH.Availability)
		api.Get("/audit/{id}", auditH.ListForEntity)
		api.Get("/audit/verify", auditH.Verify)

		// --- authenticated ---
		api.Group(func(auth chi.Router) {
			auth.Use(custommw.RequireAuth(deps.Issuer))

			auth.Post("/bookings", bookingH.Create)
			auth.Post("/bookings/{id}/cancel", bookingH.Cancel)
			auth.Get("/bookings/mine", bookingH.ListMine)

			// --- admin only ---
			auth.Group(func(admin chi.Router) {
				admin.Use(custommw.RequireRole("admin"))
				admin.Post("/resources", resourceH.Create)
			})
		})
	})

	return r
}
