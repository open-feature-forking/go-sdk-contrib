package integration

import (
	"context"
	"errors"
	"fmt"
	"github.com/cucumber/godog"
	flagd "github.com/open-feature/go-sdk-contrib/providers/flagd/pkg"
	"strconv"
	"strings"
)

type providerOption struct {
	option    string
	valueType string
	value     string
}

type errorAwareProviderConfiguration struct {
	configuration *flagd.ProviderConfiguration
	error         error
}

type ctxProviderOptionsKey struct{}

type ctxErrorAwareProviderConfigurationKey struct{}

var setEnvVar func(key, value string)

type valueVerifier struct {
	configValueResolver   func(config *flagd.ProviderConfiguration) interface{}
	expectedValueProvider func(value string) interface{}
}

var verifiersMap = map[string]valueVerifier{
	"resolver": {
		configValueResolver:   func(config *flagd.ProviderConfiguration) interface{} { return config.Resolver },
		expectedValueProvider: func(value string) interface{} { return flagd.ResolverType(strings.ToLower(value)) },
	},
	"port": {
		configValueResolver:   func(config *flagd.ProviderConfiguration) interface{} { return config.Port },
		expectedValueProvider: func(value string) interface{} { return uint16(stringToInt(value)) },
	},
	"deadlineMs": {
		configValueResolver:   func(config *flagd.ProviderConfiguration) interface{} { return config.DeadlineMs },
		expectedValueProvider: func(value string) interface{} { return stringToInt(value) },
	},
	"host": {
		configValueResolver: func(config *flagd.ProviderConfiguration) interface{} { return config.Host },
	},
	"tls": {
		configValueResolver:   func(config *flagd.ProviderConfiguration) interface{} { return config.TLSEnabled },
		expectedValueProvider: func(value string) interface{} { return stringToBool(value) },
	},
	"targetUri": {
		configValueResolver: func(config *flagd.ProviderConfiguration) interface{} { return config.TargetUri },
	},
	"certPath": {
		configValueResolver: func(config *flagd.ProviderConfiguration) interface{} { return config.CertificatePath },
	},
	"socketPath": {
		configValueResolver: func(config *flagd.ProviderConfiguration) interface{} { return config.SocketPath },
	},
	"cache": {
		configValueResolver: func(config *flagd.ProviderConfiguration) interface{} {
			return fmt.Sprintf(
				"%s",
				config.CacheType,
			)
		},
	},
	"streamDeadlineMs": {
		configValueResolver:   func(config *flagd.ProviderConfiguration) interface{} { return config.StreamDeadlineMs },
		expectedValueProvider: func(value string) interface{} { return stringToInt(value) },
	},
	"keepAliveTime": {
		configValueResolver:   func(config *flagd.ProviderConfiguration) interface{} { return config.KeepAliveTime },
		expectedValueProvider: func(value string) interface{} { return stringToInt(value) },
	},
	"retryBackoffMs": {
		configValueResolver:   func(config *flagd.ProviderConfiguration) interface{} { return config.RetryBackoffMs },
		expectedValueProvider: func(value string) interface{} { return stringToInt(value) },
	},
	"retryBackoffMaxMs": {
		configValueResolver:   func(config *flagd.ProviderConfiguration) interface{} { return config.RetryBackoffMaxMs },
		expectedValueProvider: func(value string) interface{} { return stringToInt(value) },
	},
	"retryGracePeriod": {
		configValueResolver:   func(config *flagd.ProviderConfiguration) interface{} { return config.RetryGracePeriod },
		expectedValueProvider: func(value string) interface{} { return stringToInt(value) },
	},
	"selector": {
		configValueResolver: func(config *flagd.ProviderConfiguration) interface{} { return config.Selector },
	},
	"maxCacheSize": {
		configValueResolver:   func(config *flagd.ProviderConfiguration) interface{} { return config.MaxCacheSize },
		expectedValueProvider: func(value string) interface{} { return stringToInt(value) },
	},
	"offlineFlagSourcePath": {
		configValueResolver: func(config *flagd.ProviderConfiguration) interface{} {
			return config.OfflineFlagSourcePath
		},
	},
	"offlinePollIntervalMs": {
		configValueResolver: func(config *flagd.ProviderConfiguration) interface{} {
			return config.OfflinePollIntervalMs
		},
		expectedValueProvider: func(value string) interface{} { return stringToInt(value) },
	},
}

var optionSupplierMap = map[string]func(value string) (flagd.ProviderOption, error){
	"resolver": func(value string) (flagd.ProviderOption, error) {
		switch strings.ToLower(value) {
		case "rpc":
			return flagd.WithRPCResolver(), nil
		case "in-process":
			return flagd.WithInProcessResolver(), nil
		case "file":
			return flagd.WithFileResolver(), nil
		}
		return nil, fmt.Errorf("invalid resolver '%s'", value)
	},
	"offlineFlagSourcePath": func(value string) (flagd.ProviderOption, error) {
		return flagd.WithOfflineFilePath(value), nil
	},
	"deadlineMs": func(value string) (flagd.ProviderOption, error) {
		return flagd.WithDeadlineMs(stringToInt(value)), nil
	},
	"host": func(value string) (flagd.ProviderOption, error) {
		return flagd.WithHost(value), nil
	},
	"tls": func(value string) (flagd.ProviderOption, error) {
		return flagd.WithTLS(stringToBool(value)), nil
	},
	"port": func(value string) (flagd.ProviderOption, error) {
		return flagd.WithPort(uint16(stringToInt(value))), nil
	},
	"targetUri": func(value string) (flagd.ProviderOption, error) {
		return flagd.WithTargetUri(value), nil
	},
	"certPath": func(value string) (flagd.ProviderOption, error) {
		return flagd.WithCertificatePath(value), nil
	},
	"socketPath": func(value string) (flagd.ProviderOption, error) {
		return flagd.WithSocketPath(value), nil
	},
	"streamDeadlineMs": func(value string) (flagd.ProviderOption, error) {
		return flagd.WithStreamDeadlineMs(stringToInt(value)), nil
	},
	"keepAliveTime": func(value string) (flagd.ProviderOption, error) {
		return flagd.WithKeepAliveTime(stringToInt(value)), nil
	},
	"retryBackoffMs": func(value string) (flagd.ProviderOption, error) {
		return flagd.WithRetryBackoffMs(stringToInt(value)), nil
	},
	"retryBackoffMaxMs": func(value string) (flagd.ProviderOption, error) {
		return flagd.WithRetryBackoffMaxMs(stringToInt(value)), nil
	},
	"retryGracePeriod": func(value string) (flagd.ProviderOption, error) {
		return flagd.WithRetryGracePeriod(stringToInt(value)), nil
	},
	"selector": func(value string) (flagd.ProviderOption, error) {
		return flagd.WithSelector(value), nil
	},
	"cache": func(value string) (flagd.ProviderOption, error) {
		switch strings.ToLower(value) {
		case "lru":
			return flagd.WithLRUCache(2500), nil
		case "mem":
			return flagd.WithBasicInMemoryCache(), nil
		case "disabled":

			return flagd.WithoutCache(), nil
		}
		return nil, fmt.Errorf("invalid cache type '%s'", value)
	},
	"maxCacheSize": func(value string) (flagd.ProviderOption, error) {
		return flagd.WithLRUCache(stringToInt(value)), nil
	},
	"offlinePollIntervalMs": func(value string) (flagd.ProviderOption, error) {
		return flagd.WithOfflinePollIntervalMs(stringToInt(value)), nil
	},
}

// InitializeConfigTestSuite register provider supplier and register test steps
func InitializeConfigTestSuite(setEnvVarFunc func(key, value string)) func(*godog.TestSuiteContext) {
	setEnvVar = setEnvVarFunc

	return func(suiteContext *godog.TestSuiteContext) {}
}

// InitializeConfigScenario initializes the config test scenario
func InitializeConfigScenario(ctx *godog.ScenarioContext) {
	ctx.Step(`^a config was initialized$`, aConfigWasInitialized)
	ctx.Step(`^an environment variable "([^"]*)" with value "([^"]*)"$`, anEnvironmentVariableWithValue)
	ctx.Step(`^an option "([^"]*)" of type "([^"]*)" with value "([^"]*)"$`, anOptionOfTypeWithValue)
	ctx.Step(
		`^the option "([^"]*)" of type "([^"]*)" should have the value "([^"]*)"$`,
		theOptionOfTypeShouldHaveTheValue,
	)
	ctx.Step(`^we should have an error$`, weShouldHaveAnError)
}

func aConfigWasInitialized(ctx context.Context) (context.Context, error) {
	providerOptions, _ := ctx.Value(ctxProviderOptionsKey{}).([]providerOption)

	var opts []flagd.ProviderOption

	for _, providerOption := range providerOptions {

		optionSupplier, ok := optionSupplierMap[providerOption.option]

		if !ok {
			return ctx, fmt.Errorf(
				"config with option '%s' with type '%s' and value '%s'",
				providerOption.option,
				providerOption.valueType,
				providerOption.value,
			)
		}

		option, err := optionSupplier(providerOption.value)

		if err != nil {
			return ctx, err
		}

		opts = append(opts, option)
	}

	providerConfiguration, err := flagd.NewProviderConfiguration(opts)

	errorAwareProviderConfiguration := errorAwareProviderConfiguration{
		configuration: providerConfiguration,
		error:         err,
	}

	return context.WithValue(ctx, ctxErrorAwareProviderConfigurationKey{}, errorAwareProviderConfiguration), nil
}

func anEnvironmentVariableWithValue(key, value string) {
	setEnvVar(key, value)
}

func anOptionOfTypeWithValue(ctx context.Context, option, valueType, value string) context.Context {
	providerOptions, _ := ctx.Value(ctxProviderOptionsKey{}).([]providerOption)

	data := providerOption{
		option:    option,
		valueType: valueType,
		value:     value,
	}

	providerOptions = append(providerOptions, data)

	return context.WithValue(ctx, ctxProviderOptionsKey{}, providerOptions)
}

func theOptionOfTypeShouldHaveTheValue(
	ctx context.Context, option, valueType, expectedValueS string,
) (context.Context, error) {
	errorAwareConfiguration, ok := ctx.Value(ctxErrorAwareProviderConfigurationKey{}).(errorAwareProviderConfiguration)
	if !ok {
		return ctx, errors.New("no errorAwareProviderConfiguration available")
	}

	// gherkins null value needs to converted to an empty string
	if expectedValueS == "null" {
		expectedValueS = ""
	}

	config := errorAwareConfiguration.configuration

	verifier, ok := verifiersMap[option]

	if !ok {
		return ctx, fmt.Errorf(
			"invalid option '%s' with type '%s' and value '%s'",
			option,
			valueType,
			expectedValueS,
		)
	}

	currentValue := verifier.configValueResolver(config)

	var expectedValue interface{} = expectedValueS
	if verifier.expectedValueProvider != nil {
		expectedValue = verifier.expectedValueProvider(expectedValueS)
	}

	if currentValue != expectedValue {
		return ctx, fmt.Errorf(
			"expected response of type '%s' with value '%s', got '%s'",
			valueType,
			expectedValueS,
			currentValue,
		)
	}

	return ctx, nil
}

func weShouldHaveAnError(ctx context.Context) (context.Context, error) {
	errorAwareConfiguration, ok := ctx.Value(ctxErrorAwareProviderConfigurationKey{}).(errorAwareProviderConfiguration)
	if !ok {
		return ctx, errors.New("no ProviderConfiguration found")
	}

	if errorAwareConfiguration.error == nil {
		return ctx, errors.New("configuration check succeeded, but should not")
	} else {
		return ctx, nil
	}
}

func stringToInt(str string) int {
	i, err := strconv.Atoi(str)
	if err != nil {
		panic(err)
	}
	return i
}

func stringToBool(str string) bool {
	b, err := strconv.ParseBool(str)
	if err != nil {
		panic(err)
	}

	return b
}
