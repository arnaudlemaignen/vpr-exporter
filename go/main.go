package main

import (
	"flag"
	"io"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	"vpr/pkg/types"
	"vpr/pkg/utils"

	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/robfig/cron/v3"

	log "github.com/sirupsen/logrus"
)

// BuildVersion is provided at build time
var (
	BuildVersion    string
	BuildTime       string
	AppName         = "VPR Exporter"
	debug           = flag.Bool("debug", false, "Debug mode default false")
	listenAddress   = flag.String("web.listen-address", ":9801", "Address to listen on for telemetry")
	metricsPath     = "/metrics"
	limitAliases, _ = utils.ReadLimitAliasCSVFile() //read the container limit aliases from a CSV file
)

// Ready Readiness message
func Ready() string {
	return AppName + " is ready to rock"
}

func manageLogger() {
	// Output to stdout instead of the default stderr
	// Can be any io.Writer, see below for File example
	log.SetOutput(os.Stdout)

	//activate debug mode otherwise Info as log level
	if *debug {
		log.SetLevel(log.DebugLevel)
	}

	err := godotenv.Load(".env")
	if err != nil {
		log.Warn(".env file absent, assume env variables are set.")
	}

	logDir := utils.GetStringEnv("LOG_DIR", "logs")
	//create needed dirs
	os.MkdirAll(logDir, os.ModePerm)

	// Only log the warning severity or above.
	// log.SetLevel(log.WarnLevel)
	logFile := logDir + "/" + time.Now().Format("200601021504") + "_vpr.log"
	f, err := os.OpenFile(logFile, os.O_WRONLY|os.O_CREATE, 0755)
	if err != nil {
		log.Warn("Issue with log file ", logFile)
	}
	mw := io.MultiWriter(os.Stdout, f)
	log.SetOutput(mw)
}

// Init func
func Init() {
	flag.Parse()
	manageLogger()
	err := os.MkdirAll(types.DataPath, os.ModePerm)
	if err != nil {
		log.Error(types.DataPath, " path was not created :", err)
	}

	if *debug {
		log.Info("DEV MODE : Debug logs active")
	}
}

func main() {
	Init()
	log.Info("BuildVersion ", BuildVersion, " BuildTime ", BuildTime)
	log.Info("Starting ", AppName)

	//http handler just for the readiness
	http.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
             <head><title>` + AppName + `"</title></head>
             <body>
             <h1>` + Ready() + `'</h1>
             </body>
             </html>`))
	})

	log.Info("Serving metrics on ", metricsPath)
	http.Handle(metricsPath, promhttp.Handler())

	//To catch SIGHUP signal for reloading conf
	//https://rossedman.io/blog/computers/hot-reload-sighup-with-go/
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGHUP)
	log.Info("Hot Reload enabled")

	go func() {
		log.Info("Listening on port " + *listenAddress)
		//go routine : get Data at Startup
		go getData()
		// Funcs are invoked in their own goroutine, asynchronously.
		c := cron.New()
		// Run every 10 minutes
		// c.AddFunc("@every 10m", getData)
		c.Start()

		log.Fatal(http.ListenAndServe(*listenAddress, nil))
		c.Stop()
	}()

	for range sigs {
		log.Warn("HOT RELOAD")
		// Reload the limit alias configuration
		limitAliases, _ = utils.ReadLimitAliasCSVFile() //read the container limit aliases from a CSV file
		log.Info("Yaml Config Reloaded for next round")
	}
}
