package flagd

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/go-logr/logr"
	"github.com/open-feature/flagd/core/pkg/sync"
	"github.com/open-feature/go-sdk-contrib/providers/flagd/internal/cache"
	"github.com/open-feature/go-sdk-contrib/providers/flagd/internal/logger"
	"google.golang.org/grpc"
)

type ResolverType string

// Naming and defaults must comply with flagd environment variables
const (
	defaultMaxCacheSize          int    = 1000
	defaultRpcPort               uint16 = 8013
	defaultInProcessPort         uint16 = 8015
	defaultMaxEventStreamRetries        = 5
	defaultTLS                   bool   = false
	defaultCache                        = cache.LRUValue
	defaultHost                         = "localhost"
	defaultResolver                     = rpc
	defaultStreamDeadlineMs             = 600000
	defaultDeadlineMs                   = 500
	defaultRetryBackoffMs               = 1000
	defaultRetryBackoffMaxMs            = 120000
	defaultRetryGracePeriod             = 5
	defaultOfflinePollIntervalMs        = 5000

	rpc       ResolverType = "rpc"
	inProcess ResolverType = "in-process"
	file      ResolverType = "file"

	flagdHostEnvironmentVariableName                  = "FLAGD_HOST"
	flagdPortEnvironmentVariableName                  = "FLAGD_PORT"
	flagdTLSEnvironmentVariableName                   = "FLAGD_TLS"
	flagdSocketPathEnvironmentVariableName            = "FLAGD_SOCKET_PATH"
	flagdServerCertPathEnvironmentVariableName        = "FLAGD_SERVER_CERT_PATH"
	flagdCacheEnvironmentVariableName                 = "FLAGD_CACHE"
	flagdMaxCacheSizeEnvironmentVariableName          = "FLAGD_MAX_CACHE_SIZE"
	flagdMaxEventStreamRetriesEnvironmentVariableName = "FLAGD_MAX_EVENT_STREAM_RETRIES"
	flagdResolverEnvironmentVariableName              = "FLAGD_RESOLVER"
	flagdSourceProviderIDEnvironmentVariableName      = "FLAGD_SOURCE_PROVIDER_ID"
	flagdSourceSelectorEnvironmentVariableName        = "FLAGD_SOURCE_SELECTOR"
	flagdOfflinePathEnvironmentVariableName           = "FLAGD_OFFLINE_FLAG_SOURCE_PATH"
	flagdTargetUriEnvironmentVariableName             = "FLAGD_TARGET_URI"
	flagdDeadlineMsEnvironmentVariableName            = "FLAGD_DEADLINE_MS"
	flagdStreamDeadlineMsEnvironmentVariableName      = "FLAGD_STREAM_DEADLINE_MS"
	flagdKeepAliveTimeEnvironmentVariableName         = "FLAGD_KEEP_ALIVE_TIME_MS"
	flagdRetryBackoffMsEnvironmentVariableName        = "FLAGD_RETRY_BACKOFF_MS"
	flagdRetryBackoffMaxMsEnvironmentVariableName     = "FLAGD_RETRY_BACKOFF_MAX_MS"
	flagdRetryGracePeriodEnvironmentVariableName      = "FLAGD_RETRY_GRACE_PERIOD"
	flagdOfflinePollIntervalMsEnvironmentVariableName = "FLAGD_OFFLINE_POLL_MS"
)

type ProviderConfiguration struct {
	CacheType                        cache.Type
	CertificatePath                  string
	EventStreamConnectionMaxAttempts int
	Host                             string
	MaxCacheSize                     int
	OfflineFlagSourcePath            string
	OtelIntercept                    bool
	Port                             uint16
	TargetUri                        string
	Resolver                         ResolverType
	ProviderID                       string
	Selector                         string
	SocketPath                       string
	TLSEnabled                       bool
	CustomSyncProvider               sync.ISync
	CustomSyncProviderUri            string
	GrpcDialOptionsOverride          []grpc.DialOption
	DeadlineMs                       int
	StreamDeadlineMs                 int
	KeepAliveTime                    int
	RetryBackoffMs                   int
	RetryBackoffMaxMs                int
	RetryGracePeriod                 int
	OfflinePollIntervalMs            int

	log logr.Logger
}

func newDefaultConfiguration(log logr.Logger) *ProviderConfiguration {
	p := &ProviderConfiguration{
		CacheType:                        defaultCache,
		EventStreamConnectionMaxAttempts: defaultMaxEventStreamRetries,
		Host:                             defaultHost,
		log:                              log,
		MaxCacheSize:                     defaultMaxCacheSize,
		Resolver:                         defaultResolver,
		TLSEnabled:                       defaultTLS,
		StreamDeadlineMs:                 defaultStreamDeadlineMs,
		DeadlineMs:                       defaultDeadlineMs,
		RetryBackoffMs:                   defaultRetryBackoffMs,
		RetryBackoffMaxMs:                defaultRetryBackoffMaxMs,
		RetryGracePeriod:                 defaultRetryGracePeriod,
		OfflinePollIntervalMs:            defaultOfflinePollIntervalMs,
	}

	p.updateFromEnvVar()
	return p
}

func NewProviderConfiguration(opts []ProviderOption) (*ProviderConfiguration, error) {

	log := logr.New(logger.Logger{})

	// initialize with default configurations
	providerConfiguration := newDefaultConfiguration(log)

	// explicitly declared options have the highest priority
	for _, opt := range opts {
		opt(providerConfiguration)
	}

	configureProviderConfiguration(providerConfiguration)
	err := validateProviderConfiguration(providerConfiguration)

	return providerConfiguration, err
}

func configureProviderConfiguration(p *ProviderConfiguration) {
	if len(p.OfflineFlagSourcePath) > 0 && p.Resolver == inProcess {
		p.Resolver = file
	}

	if p.Port == 0 {
		switch p.Resolver {
		case rpc:
			p.Port = defaultRpcPort
		case inProcess:
			p.Port = defaultInProcessPort
		}
	}
}

func validateProviderConfiguration(p *ProviderConfiguration) error {
	// We need a file path for file mode
	if len(p.OfflineFlagSourcePath) == 0 && p.Resolver == file {
		return errors.New("resolver Type 'file' requires a OfflineFlagSourcePath")
	}

	return nil
}

// updateFromEnvVar is a utility to update configurations based on current environment variables
func (cfg *ProviderConfiguration) updateFromEnvVar() {
	portS := os.Getenv(flagdPortEnvironmentVariableName)
	if portS != "" {
		port, err := strconv.Atoi(portS)
		if err != nil {
			cfg.log.Error(err,
				fmt.Sprintf(
					"invalid env config for %s provided, using default value: %d or %d depending on resolver",
					flagdPortEnvironmentVariableName, defaultRpcPort, defaultInProcessPort,
				))
		} else {
			cfg.Port = uint16(port)
		}
	}

	if host := os.Getenv(flagdHostEnvironmentVariableName); host != "" {
		cfg.Host = host
	}

	if socketPath := os.Getenv(flagdSocketPathEnvironmentVariableName); socketPath != "" {
		cfg.SocketPath = socketPath
	}

	if certificatePath := os.Getenv(flagdServerCertPathEnvironmentVariableName); certificatePath != "" {

		cfg.CertificatePath = certificatePath
	}

	if tlsEnabled := os.Getenv(flagdTLSEnvironmentVariableName); tlsEnabled != "" {

		cfg.TLSEnabled = strings.ToLower(tlsEnabled) == "true"
	}

	if maxCacheSizeS := os.Getenv(flagdMaxCacheSizeEnvironmentVariableName); maxCacheSizeS != "" {
		maxCacheSizeFromEnv, err := strconv.Atoi(maxCacheSizeS)
		if err != nil {
			cfg.log.Error(err,
				fmt.Sprintf("invalid env config for %s provided, using default value: %d",
					flagdMaxCacheSizeEnvironmentVariableName, defaultMaxCacheSize,
				))
		} else {
			cfg.MaxCacheSize = maxCacheSizeFromEnv
		}
	}

	if cacheValue := os.Getenv(flagdCacheEnvironmentVariableName); cacheValue != "" {
		switch cache.Type(cacheValue) {
		case cache.LRUValue:
			cfg.CacheType = cache.LRUValue
		case cache.InMemValue:
			cfg.CacheType = cache.InMemValue
		case cache.DisabledValue:
			cfg.CacheType = cache.DisabledValue
		default:
			cfg.log.Info("invalid cache type configured: %s, falling back to default: %s", cacheValue, defaultCache)
			cfg.CacheType = defaultCache
		}
	}

	if maxEventStreamRetriesS := os.Getenv(
		flagdMaxEventStreamRetriesEnvironmentVariableName); maxEventStreamRetriesS != "" {

		maxEventStreamRetries, err := strconv.Atoi(maxEventStreamRetriesS)
		if err != nil {
			cfg.log.Error(err,
				fmt.Sprintf("invalid env config for %s provided, using default value: %d",
					flagdMaxEventStreamRetriesEnvironmentVariableName, defaultMaxEventStreamRetries))
		} else {
			cfg.EventStreamConnectionMaxAttempts = maxEventStreamRetries
		}
	}

	if resolver := os.Getenv(flagdResolverEnvironmentVariableName); resolver != "" {
		switch strings.ToLower(resolver) {
		case "rpc":
			cfg.Resolver = rpc
		case "in-process":
			cfg.Resolver = inProcess
		case "file":
			cfg.Resolver = file
		default:
			cfg.log.Info("invalid resolver type: %s, falling back to default: %s", resolver, defaultResolver)
			cfg.Resolver = defaultResolver
		}
	}

	if offlinePath := os.Getenv(flagdOfflinePathEnvironmentVariableName); offlinePath != "" {
		cfg.OfflineFlagSourcePath = offlinePath
	}

	if providerID := os.Getenv(flagdSourceProviderIDEnvironmentVariableName); providerID != "" {
		cfg.ProviderID = providerID
	}

	if selector := os.Getenv(flagdSourceSelectorEnvironmentVariableName); selector != "" {
		cfg.Selector = selector
	}

	if targetUri := os.Getenv(flagdTargetUriEnvironmentVariableName); targetUri != "" {
		cfg.TargetUri = targetUri
	}

	if deadlineMsS := os.Getenv(flagdDeadlineMsEnvironmentVariableName); deadlineMsS != "" {
		deadlineMs, err := strconv.Atoi(deadlineMsS)
		if err != nil {
			cfg.log.Error(err,
				fmt.Sprintf(
					"invalid env config for %s provided, using default value: %d or %d depending on resolver",
					flagdPortEnvironmentVariableName, defaultRpcPort, defaultInProcessPort,
				))
		} else {
			cfg.DeadlineMs = deadlineMs
		}
	}

	if streamDeadlineMsS := os.Getenv(flagdStreamDeadlineMsEnvironmentVariableName); streamDeadlineMsS != "" {
		streamDeadlineMs, err := strconv.Atoi(streamDeadlineMsS)
		if err != nil {
			cfg.log.Error(err,
				fmt.Sprintf(
					"invalid env config for %s provided, using default value: %d or %d depending on resolver",
					flagdPortEnvironmentVariableName, defaultRpcPort, defaultInProcessPort,
				))
		} else {
			cfg.StreamDeadlineMs = streamDeadlineMs
		}
	}

	if keepAliveTimeS := os.Getenv(flagdKeepAliveTimeEnvironmentVariableName); keepAliveTimeS != "" {
		keepAliveTime, err := strconv.Atoi(keepAliveTimeS)
		if err != nil {
			cfg.log.Error(err,
				fmt.Sprintf(
					"invalid env config for %s provided, using default value: %d or %d depending on resolver",
					flagdPortEnvironmentVariableName, defaultRpcPort, defaultInProcessPort,
				))
		} else {
			cfg.KeepAliveTime = keepAliveTime
		}
	}

	if retryBackoffMsS := os.Getenv(flagdRetryBackoffMsEnvironmentVariableName); retryBackoffMsS != "" {
		retryBackoffMs, err := strconv.Atoi(retryBackoffMsS)
		if err != nil {
			cfg.log.Error(err,
				fmt.Sprintf(
					"invalid env config for %s provided, using default value: %d or %d depending on resolver",
					flagdPortEnvironmentVariableName, defaultRpcPort, defaultInProcessPort,
				))
		} else {
			cfg.RetryBackoffMs = retryBackoffMs
		}
	}

	if retryBackoffMaxMsS := os.Getenv(flagdRetryBackoffMaxMsEnvironmentVariableName); retryBackoffMaxMsS != "" {
		retryBackoffMaxMs, err := strconv.Atoi(retryBackoffMaxMsS)
		if err != nil {
			cfg.log.Error(err,
				fmt.Sprintf(
					"invalid env config for %s provided, using default value: %d or %d depending on resolver",
					flagdPortEnvironmentVariableName, defaultRpcPort, defaultInProcessPort,
				))
		} else {
			cfg.RetryBackoffMaxMs = retryBackoffMaxMs
		}
	}

	if retryGracePeriodS := os.Getenv(flagdRetryGracePeriodEnvironmentVariableName); retryGracePeriodS != "" {
		retryGracePeriod, err := strconv.Atoi(retryGracePeriodS)
		if err != nil {
			cfg.log.Error(err,
				fmt.Sprintf(
					"invalid env config for %s provided, using default value: %d or %d depending on resolver",
					flagdPortEnvironmentVariableName, defaultRpcPort, defaultInProcessPort,
				))
		} else {
			cfg.RetryGracePeriod = retryGracePeriod
		}
	}

	if offlinePollIntervalMsS := os.Getenv(flagdOfflinePollIntervalMsEnvironmentVariableName); offlinePollIntervalMsS != "" {
		offlinePollIntervalMs, err := strconv.Atoi(offlinePollIntervalMsS)
		if err != nil {
			cfg.log.Error(err,
				fmt.Sprintf(
					"invalid env config for %s provided, using default value: %d or %d depending on resolver",
					flagdPortEnvironmentVariableName, defaultRpcPort, defaultInProcessPort,
				))
		} else {
			cfg.OfflinePollIntervalMs = offlinePollIntervalMs
		}
	}
}
