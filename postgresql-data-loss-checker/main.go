package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/prometheus"
	api "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/sdk/metric"
)

const (
	sleepMillis = 10
)

var myIdent = time.Now().UnixNano()
var myCounter = 0
var lastWrittenCount = 0

type Counters struct {
	InsertCounter     api.Float64Counter
	FailedCounter     api.Float64Counter
	SelectDiffCounter api.Float64Counter
	Ctx               context.Context
	Opt               api.MeasurementOption
}

func main() {
	psqlInfo := getPsqlInfo()
	createTable(psqlInfo)

	counters := getCounters()
	go serveMetrics()

	if os.Getenv("MODE") == "single" {
		ethernalConnection(psqlInfo, counters)
	} else {
		separateConnection(psqlInfo, counters)
	}
}

func getCounters() Counters {
	ctx, meter := otelContext()
	opt := api.WithAttributes(
		attribute.Key("myident").Int64(myIdent),
	)
	insertCounter, err := meter.Float64Counter("dataloss_checker_insert_total", api.WithDescription("request counter"))
	if err != nil {
		log.Fatal(err)
	}
	failedCounter, err := meter.Float64Counter("dataloss_checker_insert_failed", api.WithDescription("failed request counter"))
	if err != nil {
		log.Fatal(err)
	}
	selectDiffCounter, err := meter.Float64Counter("dataloss_checker_select_diff", api.WithDescription("select diff request counter"))
	if err != nil {
		log.Fatal(err)
	}

	insertCounter.Add(ctx, 0.0, opt)
	failedCounter.Add(ctx, 0.0, opt)
	selectDiffCounter.Add(ctx, 0.0, opt)

	return Counters{insertCounter, failedCounter, selectDiffCounter, ctx, opt}
}

func getPsqlInfo() string {
	host := os.Getenv("HOST")
	port, _ := strconv.Atoi(os.Getenv("PORT"))
	user := os.Getenv("USER")
	password := os.Getenv("PASSWORD")
	dbname := os.Getenv("DBNAME")
	sslmode := os.Getenv("SSLMODE")

	if sslmode != "" {
		sslmode = "sslmode=" + sslmode
	}

	result := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s %s", host, port, user, password, dbname, sslmode)
	log.Println("Connecting to " + result)
	return result
}

func otelContext() (context.Context, api.Meter) {
	ctx := context.Background()

	// The exporter embeds a default OpenTelemetry Reader and
	// implements prometheus.Collector, allowing it to be used as
	// both a Reader and Collector.
	exporter, err := prometheus.New()
	if err != nil {
		log.Fatal(err)
	}
	provider := metric.NewMeterProvider(metric.WithReader(exporter))
	meter := provider.Meter("data-loss-checker")

	return ctx, meter
}

func separateConnection(psqlInfo string, counters Counters) {
	for {
		db, err := sql.Open("postgres", psqlInfo)
		if err != nil {
			log.Println(err)
			continue
		}

		insertSelectIteration(db, counters)
		db.Close()
	}
}

func ethernalConnection(psqlInfo string, counters Counters) {
	for {
		singleConnection(psqlInfo, counters)
	}
}

func singleConnection(psqlInfo string, counters Counters) {
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	for {
		insertSelectIteration(db, counters)
	}
}

func insertSelectIteration(db *sql.DB, counters Counters) {
	time.Sleep(sleepMillis * time.Millisecond)

	lastWritten := 0
	err := db.QueryRow("SELECT COALESCE(MAX(mynumber), 0) FROM ledger WHERE myident = $1", myIdent).Scan(&lastWritten)
	if err != nil {
		fmt.Println(err)
		return
	}
	if lastWrittenCount > 0 && lastWritten != lastWrittenCount {
		counters.SelectDiffCounter.Add(counters.Ctx, float64(lastWrittenCount-lastWritten), counters.Opt)
	}

	sqlStatement := `
			INSERT INTO ledger(myident, mynumber, app_insert_timestamp)
			VALUES ($1, $2, $3)
		`
	myCounter += 1
	err = db.QueryRow(sqlStatement, myIdent, myCounter, time.Now()).Scan()
	counters.InsertCounter.Add(counters.Ctx, float64(1), counters.Opt)
	if err != sql.ErrNoRows {
		counters.FailedCounter.Add(counters.Ctx, float64(1), counters.Opt)
		log.Println(err)
		return
	} else {
		lastWrittenCount = myCounter
	}

}

func createTable(psqlInfo string) {
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	sqlStatement := `
		CREATE TABLE IF NOT EXISTS ledger(
			myident bigint not null,
			mynumber bigint not null,
			app_insert_timestamp timestamp not null,
			db_insert_timestamp timestamp default NOW(),
			primary key (myident, mynumber)
		);
	`
	err = db.QueryRow(sqlStatement).Scan()
	if err != sql.ErrNoRows {
		panic(err)
	}
}

func serveMetrics() {
	log.Printf("serving metrics at localhost:9090/metrics")

	http.Handle("/metrics", promhttp.Handler())
	err := http.ListenAndServe(":9090", nil) //nolint:gosec // Ignoring G114: Use of net/http serve function that has no support for setting timeouts.
	if err != nil {
		log.Printf("error serving http: %v", err)
		return
	}
}
