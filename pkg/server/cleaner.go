package server

import (
	"context"
	"time"

	raven "github.com/getsentry/raven-go"
	"github.com/gocardless/draupnir/pkg/exec"
	"github.com/gocardless/draupnir/pkg/models"
	"github.com/gocardless/draupnir/pkg/server/api/auth"
	"github.com/gocardless/draupnir/pkg/server/api/middleware"
	"github.com/gocardless/draupnir/pkg/store"
	"github.com/pkg/errors"
	"github.com/prometheus/common/log"
)

type InstanceCleaner struct {
	logger        log.Logger
	sentryClient  *raven.Client
	instanceStore store.InstanceStore
	executor      exec.Executor
	authenticator auth.Authenticator
}

func NewInstanceCleaner(logger log.Logger, sentryClient *raven.Client, instanceStore store.InstanceStore, executor exec.Executor, authenticator auth.Authenticator) *InstanceCleaner {
	return &InstanceCleaner{
		logger:        logger,
		sentryClient:  sentryClient,
		instanceStore: instanceStore,
		executor:      executor,
		authenticator: authenticator,
	}
}

func (ic *InstanceCleaner) Start(ctx context.Context, interval time.Duration) error {
	// We need to add a logger to the context, as the exec package depends on one
	// being present in order to log
	ctx = context.WithValue(ctx, middleware.LoggerKey, &ic.logger)
	for {
		select {
		case <-time.After(interval):
			ic.logger.Info("Cleaning old instances with invalid tokens")
			instances, err := ic.instanceStore.List()
			if err != nil {
				err = errors.Wrap(err, "cannot clean instances: unable to list instances")
				ic.logger.Error(err.Error())
				ic.sentryClient.CaptureError(err, map[string]string{})
			} else {
				for _, instance := range instances {
					if instance.RefreshToken != "" {
						valid, err, validityErr := ic.authenticator.IsRefreshTokenValid(instance.RefreshToken)
						if err != nil {
							err = errors.Wrap(err, "failed to validate token")
							ic.logger.With("instance", instance.ID).Error(err.Error())
							ic.sentryClient.CaptureError(err, map[string]string{})
						} else if !valid {
							logger := ic.logger.With("instance", instance.ID).With("user", instance.UserEmail)
							logger.Infof("Token for instance invalid: destroying instance: %s", validityErr.Error())
							err = ic.destroyInstance(ctx, instance)
							if err != nil {
								err = errors.Wrap(err, "failed to destroy instance")
								logger.Error(err.Error())
								ic.sentryClient.CaptureError(err, map[string]string{})
							}
						}
					}
				}
			}
		case <-ctx.Done():
			return nil
		}
	}
}

func (ic *InstanceCleaner) destroyInstance(ctx context.Context, instance models.Instance) error {
	err := ic.executor.DestroyInstance(ctx, instance.ID)
	if err == nil {
		err = ic.instanceStore.Destroy(instance)
	}
	return err
}
