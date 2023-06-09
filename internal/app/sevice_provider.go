package app

import (
	"context"

	accessV1 "github.com/Arkosh744/auth-service-api/internal/api/access_v1"
	authV1 "github.com/Arkosh744/auth-service-api/internal/api/auth_v1"
	userV1 "github.com/Arkosh744/auth-service-api/internal/api/user_v1"
	"github.com/Arkosh744/auth-service-api/internal/client/pg"
	"github.com/Arkosh744/auth-service-api/internal/closer"
	"github.com/Arkosh744/auth-service-api/internal/config"
	"github.com/Arkosh744/auth-service-api/internal/log"
	"github.com/Arkosh744/auth-service-api/internal/rate_limiter"
	accessRepo "github.com/Arkosh744/auth-service-api/internal/repo/access"
	userRepo "github.com/Arkosh744/auth-service-api/internal/repo/user"
	accessService "github.com/Arkosh744/auth-service-api/internal/service/access"
	authService "github.com/Arkosh744/auth-service-api/internal/service/auth"
	userService "github.com/Arkosh744/auth-service-api/internal/service/user"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/sony/gobreaker"
	"go.uber.org/zap"
)

type serviceProvider struct {
	pgConfig        config.PGConfig
	grpcConfig      config.GRPCConfig
	httpConfig      config.HTTPConfig
	swaggerConfig   config.SwaggerConfig
	promConfig      config.PromConfig
	rateLimitConfig config.RateLimitConfig
	breakerConfig   config.BreakerConfig
	authConfig      config.AuthConfig

	pgClient       pg.Client
	rateLimiter    *rate_limiter.TokenBucketLimiter
	circuitBreaker *gobreaker.CircuitBreaker

	accessRepository accessRepo.Repository
	userRepository   userRepo.Repository

	accessService accessService.Service
	authService   authService.Service
	userService   userService.Service

	accessImpl *accessV1.Implementation
	userImpl   *userV1.Implementation
	authImpl   *authV1.Implementation
}

func newServiceProvider() *serviceProvider {
	return &serviceProvider{}
}

func (s *serviceProvider) GetAuthConfig() config.AuthConfig {
	if s.authConfig == nil {
		cfg, err := config.NewAuthConfig()
		if err != nil {
			log.Fatalf("failed to get auth config: %s", err.Error())
		}

		s.authConfig = cfg
	}

	return s.authConfig
}

func (s *serviceProvider) GetRateLimitConfig() config.RateLimitConfig {
	if s.rateLimitConfig == nil {
		cfg, err := config.NewRateLimitConfig()
		if err != nil {
			log.Fatalf("failed to get rate limit config", zap.Error(err))
		}

		s.rateLimitConfig = cfg
	}

	return s.rateLimitConfig
}

func (s *serviceProvider) GetBreakerConfig() config.BreakerConfig {
	if s.breakerConfig == nil {
		cfg, err := config.NewBreakerConfig()
		if err != nil {
			log.Fatalf("failed to get circuit breaker config", zap.Error(err))
		}

		s.breakerConfig = cfg
	}

	return s.breakerConfig
}

func (s *serviceProvider) GetPGConfig() config.PGConfig {
	if s.pgConfig == nil {
		cfg, err := config.NewPGConfig()
		if err != nil {
			log.Fatalf("failed to get pg config", zap.Error(err))
		}

		s.pgConfig = cfg
	}

	return s.pgConfig
}

func (s *serviceProvider) GetGRPCConfig() config.GRPCConfig {
	if s.grpcConfig == nil {
		cfg, err := config.NewGRPCConfig()
		if err != nil {
			log.Fatalf("failed to get grpc config", zap.Error(err))
		}

		s.grpcConfig = cfg
	}

	return s.grpcConfig
}

func (s *serviceProvider) GetHTTPConfig() config.HTTPConfig {
	if s.httpConfig == nil {
		cfg, err := config.NewHTTPConfig()
		if err != nil {
			log.Fatalf("failed to get http config", zap.Error(err))
		}

		s.httpConfig = cfg
	}

	return s.httpConfig
}

func (s *serviceProvider) GetPromConfig() config.PromConfig {
	if s.promConfig == nil {
		cfg, err := config.NewPromConfig()
		if err != nil {
			log.Fatalf("failed to get prom config", zap.Error(err))
		}

		s.promConfig = cfg
	}

	return s.promConfig
}

func (s *serviceProvider) GetSwaggerConfig() config.SwaggerConfig {
	if s.swaggerConfig == nil {
		cfg, err := config.NewSwaggerConfig()
		if err != nil {
			log.Fatalf("failed to get swagger config: %s", err.Error())
		}

		s.swaggerConfig = cfg
	}

	return s.swaggerConfig
}

func (s *serviceProvider) GetRateLimiter(ctx context.Context) *rate_limiter.TokenBucketLimiter {
	if s.rateLimiter == nil {
		s.rateLimiter = rate_limiter.NewTokenBucketLimiter(
			ctx,
			s.GetRateLimitConfig().Limit(),
			s.GetRateLimitConfig().Period())
	}

	return s.rateLimiter
}

func (s *serviceProvider) GetBreaker(_ context.Context) *gobreaker.CircuitBreaker {
	if s.circuitBreaker == nil {
		s.circuitBreaker = gobreaker.NewCircuitBreaker(gobreaker.Settings{
			Name:        "auth-service-api",
			MaxRequests: uint32(s.GetBreakerConfig().Requests()),
			Interval:    s.GetBreakerConfig().Interval(),
			Timeout:     s.GetBreakerConfig().Timeout(),
			ReadyToTrip: func(counts gobreaker.Counts) bool {
				// >60% of requests failed => open circuit (no new requests allowed)
				return float64(counts.TotalFailures)/float64(counts.Requests) > 0.6
			},
			OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
				log.Infof("grpc breaker state changed: %s %s -> %s", name, from, to)
			},
		})
	}

	return s.circuitBreaker
}

func (s *serviceProvider) GetPGClient(ctx context.Context) pg.Client {
	if s.pgClient == nil {
		pgCfg, err := pgxpool.ParseConfig(s.GetPGConfig().DSN())
		if err != nil {
			log.Fatalf("failed to parse pg config", zap.Error(err))
		}

		cl, err := pg.NewClient(ctx, pgCfg)
		if err != nil {
			log.Fatalf("failed to get pg client", zap.Error(err))
		}

		if cl.PG().Ping(ctx) != nil {
			log.Fatalf("failed to ping pg", zap.Error(err))
		}

		closer.Add(cl.Close)

		s.pgClient = cl
	}

	return s.pgClient
}

func (s *serviceProvider) GetUserRepo(ctx context.Context) userRepo.Repository {
	if s.userRepository == nil {
		s.userRepository = userRepo.NewRepository(s.GetPGClient(ctx))
	}

	return s.userRepository
}

func (s *serviceProvider) GetUserService(ctx context.Context) userService.Service {
	if s.userService == nil {
		s.userService = userService.NewService(s.GetUserRepo(ctx))
	}

	return s.userService
}

func (s *serviceProvider) GetUserImpl(ctx context.Context) *userV1.Implementation {
	if s.userImpl == nil {
		s.userImpl = userV1.NewImplementation(s.GetUserService(ctx))
	}

	return s.userImpl
}

func (s *serviceProvider) GetAuthService(ctx context.Context) authService.Service {
	if s.authService == nil {
		s.authService = authService.NewService(
			s.GetAuthConfig(),
			s.GetUserRepo(ctx),
		)
	}

	return s.authService
}

func (s *serviceProvider) GetAuthImpl(ctx context.Context) *authV1.Implementation {
	if s.authImpl == nil {
		s.authImpl = authV1.NewImplementation(s.GetAuthService(ctx))
	}

	return s.authImpl
}

func (s *serviceProvider) GetAccessRepository(ctx context.Context) accessRepo.Repository {
	if s.accessRepository == nil {
		s.accessRepository = accessRepo.NewRepository(s.GetPGClient(ctx))
	}

	return s.accessRepository
}

func (s *serviceProvider) GetAccessService(ctx context.Context) accessService.Service {
	if s.accessService == nil {
		s.accessService = accessService.NewService(s.GetAccessRepository(ctx), s.GetAuthConfig())
	}

	return s.accessService
}

func (s *serviceProvider) GetAccessImpl(ctx context.Context) *accessV1.Implementation {
	if s.accessImpl == nil {
		s.accessImpl = accessV1.NewImplementation(s.GetAccessService(ctx))
	}

	return s.accessImpl
}
