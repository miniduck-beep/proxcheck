package main

import (
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"projectx/proxytestlib/checker"
	"projectx/proxytestlib/config"
	"projectx/proxytestlib/metrics"
	"projectx/proxytestlib/models"
	"projectx/proxytestlib/runner"
	"projectx/proxytestlib/xray"
)

func main() {
	// Initialize config.CLIConfig (mimicking original xray-checker behavior)
	// We need to set default values for Xray.StartPort, Proxy.IpCheckUrl, etc.
	config.CLIConfig.Xray.StartPort = 10000
	config.CLIConfig.Xray.LogLevel = "debug"
	config.CLIConfig.Proxy.CheckMethod = "ip"
	config.CLIConfig.Proxy.IpCheckUrl = "https://api.ipify.org?format=text"
	config.CLIConfig.Proxy.Timeout = 60
	config.CLIConfig.Proxy.SimulateLatency = false // Disable for accurate readings
	config.CLIConfig.Metrics.Port = "2112"
	config.CLIConfig.Metrics.Instance = "proxy-tester-instance"


	// Initialize metrics
	metrics.InitMetrics(config.CLIConfig.Metrics.Instance)
	registry := prometheus.NewRegistry()
	registry.MustRegister(metrics.GetProxyStatusMetric())
	registry.MustRegister(metrics.GetProxyLatencyMetric())

	// Hardcoded proxy configs for demonstration
	proxyConfigs := []*models.ProxyConfig{
		{
			Protocol: "vless",
			Name:     "Proxy1",
			Server:   "84.246.85.62",
			Port:     8880,
			UUID:     "df0680ca-e43c-498d-ed86-8e196eedd012",
			Security: "none",
			Type:     "grpc",
		},
		{
			Protocol: "vless",
			Name:     "Proxy2",
			Server:   "a-lf-0.okamii.ir",
			Port:     2095,
			UUID:     "19b7d4d8-de95-41cf-b270-a310cc21f151",
			Security: "none",
			Type:     "tcp",
		},
		{
			Protocol: "vless",
			Name:     "NonWorking",
			Server:   "1.2.3.4",
			Port:     12345,
			UUID:     "non-existent-uuid",
			Security: "none",
			Type:     "tcp",
		},
	}

	if len(proxyConfigs) == 0 {
		log.Fatalf("No proxy configurations to check.")
	}

	// Prepare proxy configs (assign Index and StableID)
	xray.PrepareProxyConfigs(proxyConfigs)

	// Generate and save Xray config
	configFile := "xray_config.json"
	if err := xray.GenerateAndSaveConfig(proxyConfigs, config.CLIConfig.Xray.StartPort, configFile, config.CLIConfig.Xray.LogLevel); err != nil {
		log.Fatalf("Error generating Xray config: %v", err)
	}

	// Initialize Xray runner
	xrayRunner := runner.NewXrayRunner(configFile)
	if err := xrayRunner.Start(); err != nil {
		log.Fatalf("Error starting Xray runner: %v", err)
	}
	defer xrayRunner.Stop()

	// Initialize proxy checker
	proxyChecker := checker.NewProxyChecker(
		proxyConfigs,
		config.CLIConfig.Xray.StartPort,
		config.CLIConfig.Proxy.IpCheckUrl,
		config.CLIConfig.Proxy.Timeout,
		config.CLIConfig.Proxy.StatusCheckUrl,
		config.CLIConfig.Proxy.DownloadUrl,
		config.CLIConfig.Proxy.DownloadTimeout,
		config.CLIConfig.Proxy.DownloadMinSize,
		config.CLIConfig.Proxy.CheckMethod,
		config.CLIConfig.Metrics.Instance,
	)

	// Run checks
	log.Println("Starting proxy check iteration...")
	proxyChecker.CheckAllProxies()
	log.Println("Proxy check iteration completed.")

	// Expose metrics via HTTP
	http.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
	log.Printf("Metrics server listening on :%s", config.CLIConfig.Metrics.Port)
	log.Fatal(http.ListenAndServe(":"+config.CLIConfig.Metrics.Port, nil))
}