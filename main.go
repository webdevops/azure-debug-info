package main

import (
	"context"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/profiles/latest/resources/mgmt/resources"
	"github.com/Azure/azure-sdk-for-go/profiles/latest/resources/mgmt/subscriptions"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/Azure/go-autorest/autorest/to"
	jwt "github.com/golang-jwt/jwt/v4"
	"github.com/jessevdk/go-flags"
	log "github.com/sirupsen/logrus"
	"github.com/webdevops/azure-debug-info/config"
	"os"
	"os/signal"
	"path"
	"runtime"
	"strings"
	"syscall"
)

const (
	Author = "webdevops.io"
)

var (
	argparser *flags.Parser
	opts      config.Opts

	AzureAuthorizer    autorest.Authorizer
	AzureEnvironment   azure.Environment
	AzureSubscriptions []subscriptions.Subscription

	// Git version information
	gitCommit = "<unknown>"
	gitTag    = "<unknown>"
)

func main() {
	initArgparser()

	log.Infof("starting azure-debug-info v%s (%s; %s; by %v)", gitTag, gitCommit, runtime.Version(), Author)
	log.Info(string(opts.GetJson()))

	log.Infof("init Azure connection")
	initAzureConnection()

	startAzureReport()

	waitForSignal()
}

// init argparser and parse/validate arguments
func initArgparser() {
	argparser = flags.NewParser(&opts, flags.Default)
	_, err := argparser.Parse()

	// check if there is an parse error
	if err != nil {
		if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
			os.Exit(0)
		} else {
			fmt.Println()
			argparser.WriteHelp(os.Stdout)
			os.Exit(1)
		}
	}

	// verbose level
	if opts.Logger.Verbose {
		log.SetLevel(log.DebugLevel)
	}

	// debug level
	if opts.Logger.Debug {
		log.SetReportCaller(true)
		log.SetLevel(log.TraceLevel)
		log.SetFormatter(&log.TextFormatter{
			CallerPrettyfier: func(f *runtime.Frame) (string, string) {
				s := strings.Split(f.Function, ".")
				funcName := s[len(s)-1]
				return funcName, fmt.Sprintf("%s:%d", path.Base(f.File), f.Line)
			},
		})
	}

	// json log format
	if opts.Logger.LogJson {
		log.SetReportCaller(true)
		log.SetFormatter(&log.JSONFormatter{
			DisableTimestamp: true,
			CallerPrettyfier: func(f *runtime.Frame) (string, string) {
				s := strings.Split(f.Function, ".")
				funcName := s[len(s)-1]
				return funcName, fmt.Sprintf("%s:%d", path.Base(f.File), f.Line)
			},
		})
	}
}

// Init and build Azure authorzier
func initAzureConnection() {
	var err error
	ctx := context.Background()

	// setup azure authorizer
	AzureAuthorizer, err = auth.NewAuthorizerFromEnvironment()
	if err != nil {
		log.Panic(err)
	}
	subscriptionsClient := subscriptions.NewClient()
	subscriptionsClient.Authorizer = AzureAuthorizer

	// auto lookup subscriptions
	listResult, err := subscriptionsClient.List(ctx)
	if err != nil {
		log.Panic(err)
	}
	AzureSubscriptions = listResult.Values()

	AzureEnvironment, err = azure.EnvironmentFromName(*opts.Azure.Environment)
	if err != nil {
		log.Panic(err)
	}
}

func startAzureReport() {
	ctx := context.Background()
	log.Infof("starting access report")
	log.Infof("running in Azure environment \"%v\"", AzureEnvironment.Name)

	log.Infof("searching for ServicePrincipal information")
	authSettings, _ := auth.GetSettingsFromEnvironment()

	// spn detection
	if spnToken, err := authSettings.GetMSI().ServicePrincipalToken(); err == nil {
		// msi detected
		if err := spnToken.EnsureFresh(); err != nil {
			log.Panic(err)
		}

		jwtToken, _ := jwt.Parse(spnToken.Token().AccessToken, func(token *jwt.Token) (interface{}, error) {
			return []byte{}, nil
		})

		// we ignore jwt parsing issues here,
		// we don't care about the signature
		if jwtToken != nil {
			if claims, ok := jwtToken.Claims.(jwt.MapClaims); ok {
				spnInfo := log.Fields{}
				if val, ok := claims["oid"]; ok {
					spnInfo["objectid"] = val.(string)
				}

				if val, ok := claims["appid"]; ok {
					spnInfo["appid"] = val.(string)
				}

				if val, ok := claims["tid"]; ok {
					spnInfo["tenantid"] = val.(string)
				}

				log.WithFields(spnInfo).Info("found MSI ServicePrincipal in auth token")
			}
		}
	} else {
		// env settings
		spnInfo := log.Fields{}

		if val := os.Getenv("AZURE_CLIENT_ID"); val != "" {
			spnInfo["clientid"] = val
		}

		if val := os.Getenv("AZURE_TENANT_ID"); val != "" {
			spnInfo["tenantid"] = val
		}

		if len(spnInfo) > 0 {
			log.WithFields(spnInfo).Infof("using ServicePrincipal in ENV vars")
		} else {
			log.WithFields(spnInfo).Infof("unable to detect ServicePrincipal")
		}
	}

	log.Infof("starting Azure access report")
	for _, subscription := range AzureSubscriptions {
		contextLogger := log.WithField("subscription", to.String(subscription.SubscriptionID))
		contextLogger.Infof("found subscription \"%s\"", to.String(subscription.DisplayName))

		client := resources.NewGroupsClientWithBaseURI(AzureEnvironment.ResourceManagerEndpoint, *subscription.SubscriptionID)
		client.Authorizer = AzureAuthorizer

		resourceGroupResult, err := client.ListComplete(ctx, "", nil)
		if err != nil {
			contextLogger.Error(err)
			continue
		}

		for _, item := range *resourceGroupResult.Response().Value {
			contextLogger.WithField("resouceGroup", to.String(item.Name)).Info("found resouceGroup")
		}
	}

	log.Infof("report finished")
}

func waitForSignal() {
	sigs := make(chan os.Signal, 1)
	done := make(chan bool, 1)

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigs
		fmt.Println()
		fmt.Println(sig)
		done <- true
	}()

	<-done
}
