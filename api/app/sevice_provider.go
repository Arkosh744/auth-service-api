package app

import (
	"context"

	userV1 "github.com/Arkosh744/auth-grpc/internal/api/user_v1"
	"github.com/Arkosh744/auth-grpc/internal/client/pg"
	"github.com/Arkosh744/auth-grpc/internal/closer"
	"github.com/Arkosh744/auth-grpc/internal/config"
	userRepo "github.com/Arkosh744/auth-grpc/internal/repo/user"
	userService "github.com/Arkosh744/auth-grpc/internal/service/user"
	"github.com/jackc/pgx/v4/pgxpool"
	"go.uber.org/zap"
)

type serviceProvider struct {
	pgConfig   config.PGConfig
	grpcConfig config.GRPCConfig

	pgClient       pg.Client
	userRepository userRepo.Repository
	userService    userService.Service
	log            *zap.SugaredLogger

	userImpl *userV1.Implementation
}

func newServiceProvider(log *zap.SugaredLogger) *serviceProvider {
	return &serviceProvider{log: log}
}

func (s *serviceProvider) GetPGConfig() config.PGConfig {
	if s.pgConfig == nil {
		cfg, err := config.NewPGConfig()
		if err != nil {
			s.log.Fatal("failed to get pg config", zap.Error(err))
		}

		s.pgConfig = cfg
	}

	return s.pgConfig
}

func (s *serviceProvider) GetGRPCConfig() config.GRPCConfig {
	if s.grpcConfig == nil {
		cfg, err := config.NewGRPCConfig()
		if err != nil {
			s.log.Fatal("failed to get grpc config", zap.Error(err))
		}

		s.grpcConfig = cfg
	}

	return s.grpcConfig
}

func (s *serviceProvider) GetPGClient(ctx context.Context) pg.Client {
	if s.pgClient == nil {
		pgCfg, err := pgxpool.ParseConfig(s.GetPGConfig().DSN())
		if err != nil {
			s.log.Fatal("failed to parse pg config", zap.Error(err))
		}

		cl, err := pg.NewClient(ctx, pgCfg, s.log)
		if err != nil {
			s.log.Fatal("failed to get pg client", zap.Error(err))
		}

		if cl.PG().Ping(ctx) != nil {
			s.log.Fatal("failed to ping pg", zap.Error(err))
		}

		closer.Add(cl.Close)

		s.pgClient = cl
	}

	return s.pgClient
}

func (s *serviceProvider) GetUserRepo(ctx context.Context) userRepo.Repository {
	if s.userRepository == nil {
		s.userRepository = userRepo.NewRepository(s.GetPGClient(ctx), s.log)
	}

	return s.userRepository
}

func (s *serviceProvider) GetUserService(ctx context.Context) userService.Service {
	if s.userService == nil {
		s.userService = userService.NewService(s.GetUserRepo(ctx), s.log)
	}

	return s.userService
}

func (s *serviceProvider) GetUserImpl(ctx context.Context) *userV1.Implementation {
	if s.userImpl == nil {
		s.userImpl = userV1.NewImplementation(s.GetUserService(ctx), s.log)
	}

	return s.userImpl
}