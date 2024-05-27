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
var prevSelected = 0

type Counters struct {
	InsertCounter       api.Float64Counter
	FailedCounter       api.Float64Counter
	InsertDiffCounter   api.Float64Counter
	SelectDiffCounter   api.Float64Counter
	SelectCounter       api.Float64Counter
	SelectFailedCounter api.Float64Counter
	InsertUp            api.Float64Gauge
	SelectUp            api.Float64Gauge
	Ctx                 context.Context
	Opt                 api.MeasurementOption
}

func main() {
	psqlInfo := getPsqlInfo()
	if os.Getenv("READONLY") != "true" {
		createTable(psqlInfo)
	}

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
	insertDiffCounter, err := meter.Float64Counter("dataloss_checker_insert_diff", api.WithDescription("insert diff request counter"))
	if err != nil {
		log.Fatal(err)
	}
	selectDiffCounter, err := meter.Float64Counter("dataloss_checker_select_diff", api.WithDescription("select diff request counter"))
	if err != nil {
		log.Fatal(err)
	}
	selectCounter, err := meter.Float64Counter("dataloss_checker_select_total", api.WithDescription("select request counter"))
	if err != nil {
		log.Fatal(err)
	}
	selectFailedCounter, err := meter.Float64Counter("dataloss_checker_select_failed", api.WithDescription("select failed request counter"))
	if err != nil {
		log.Fatal(err)
	}
	selectUp, err := meter.Float64Gauge("dataloss_checker_select_success", api.WithDescription("select working"))
	if err != nil {
		log.Fatal(err)
	}
	insertUp, err := meter.Float64Gauge("dataloss_checker_insert_success", api.WithDescription("insert working"))
	if err != nil {
		log.Fatal(err)
	}

	insertCounter.Add(ctx, 0.0, opt)
	failedCounter.Add(ctx, 0.0, opt)
	insertDiffCounter.Add(ctx, 0.0, opt)
	selectDiffCounter.Add(ctx, 0.0, opt)
	selectCounter.Add(ctx, 0, opt)
	selectFailedCounter.Add(ctx, 0, opt)
	insertUp.Record(ctx, 0, opt)
	selectUp.Record(ctx, 0, opt)

	return Counters{insertCounter, failedCounter, insertDiffCounter, selectDiffCounter, selectCounter, selectFailedCounter, insertUp, selectUp, ctx, opt}
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
			counters.SelectUp.Record(counters.Ctx, float64(0), counters.Opt)
			counters.InsertUp.Record(counters.Ctx, float64(0), counters.Opt)

			log.Println(err)
			time.Sleep(sleepMillis * time.Millisecond)
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
		counters.SelectUp.Record(counters.Ctx, float64(0), counters.Opt)
		counters.InsertUp.Record(counters.Ctx, float64(0), counters.Opt)

		log.Println(err)
		time.Sleep(sleepMillis * time.Millisecond)
		return
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
	counters.SelectCounter.Add(counters.Ctx, float64(1), counters.Opt)
	if err != nil {
		counters.SelectFailedCounter.Add(counters.Ctx, float64(1), counters.Opt)
		counters.SelectUp.Record(counters.Ctx, float64(0), counters.Opt)
		counters.InsertUp.Record(counters.Ctx, float64(0), counters.Opt)
		fmt.Println(err)
		return
	} else {
		counters.SelectUp.Record(counters.Ctx, float64(1), counters.Opt)
	}
	if lastWrittenCount > 0 && lastWritten != lastWrittenCount { // data loss
		counters.InsertDiffCounter.Add(counters.Ctx, float64(lastWrittenCount-lastWritten), counters.Opt)
	}
	if prevSelected > lastWritten { // data loss on replica
		counters.SelectDiffCounter.Add(counters.Ctx, float64(prevSelected-lastWritten), counters.Opt)
	}
	prevSelected = lastWritten

	if os.Getenv("READONLY") != "true" {
		sqlStatement := `
				INSERT INTO ledger(myident, mynumber, app_insert_timestamp)
				VALUES ($1, $2, $3)
			`
		myCounter += 1
		err = db.QueryRow(sqlStatement, myIdent, myCounter, time.Now()).Scan()
		counters.InsertCounter.Add(counters.Ctx, float64(1), counters.Opt)
		if err != sql.ErrNoRows {
			counters.FailedCounter.Add(counters.Ctx, float64(1), counters.Opt)
			counters.InsertUp.Record(counters.Ctx, float64(0), counters.Opt)
			log.Println(err)
			return
		} else {
			lastWrittenCount = myCounter
			counters.InsertUp.Record(counters.Ctx, float64(1), counters.Opt)
		}
	}
}

func createTable(psqlInfo string) {
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Println(err)
		return
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
		log.Println(err)
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
