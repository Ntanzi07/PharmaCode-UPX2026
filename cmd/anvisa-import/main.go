// Command anvisa-import loads the Anvisa open data into the database.
//
// It reads two files published by Anvisa and applies them with the same upsert
// the panel's spreadsheet import uses, so running it again updates what is
// already there instead of duplicating:
//
//	DADOS_ABERTOS_MEDICAMENTOS.csv  -> drugs + active ingredients
//	TA_PRECO_MEDICAMENTO.csv        -> packages + barcodes (CMED price list)
//
// It does NOT bring leaflets: Anvisa publishes those as one PDF per product,
// and in this project the simplified leaflet is written by the team and
// reviewed by a pharmacist. This command only fills in the boring part, so the
// app can resolve a scanned barcode into a drug.
//
// Usage:
//
//	go run ./cmd/anvisa-import --dry-run
//	go run ./cmd/anvisa-import --medicamentos tmp/anvisa/DADOS_ABERTOS_MEDICAMENTOS.csv \
//	                           --precos       tmp/anvisa/TA_PRECO_MEDICAMENTO.csv
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"regexp"
	"sort"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Ntanzi07/PharmaCode-UPX2026/internal/anvisa"
	"github.com/Ntanzi07/PharmaCode-UPX2026/internal/config"
	"github.com/Ntanzi07/PharmaCode-UPX2026/internal/importer"
	"github.com/Ntanzi07/PharmaCode-UPX2026/internal/service"
)

// Where the files live when no path is given. They can be a local path or a
// URL; Anvisa may be unreachable from a corporate network, in which case
// download them in the browser and pass the paths.
const (
	defaultMedicamentos = "https://dados.anvisa.gov.br/dados/DADOS_ABERTOS_MEDICAMENTOS.csv"
	defaultPrecos       = "https://dados.anvisa.gov.br/dados/TA_PRECO_MEDICAMENTO.csv"
)

// maxReportedErrors keeps a badly formed file from printing thousands of lines.
const maxReportedErrors = 20

func main() {
	var (
		medicamentos = flag.String("medicamentos", defaultMedicamentos, "DADOS_ABERTOS_MEDICAMENTOS.csv: local path or URL")
		precos       = flag.String("precos", defaultPrecos, "CMED price list (.csv or .xlsx): local path or URL")
		dryRun       = flag.Bool("dry-run", false, "read and count everything, then roll back: nothing is written")
		onlyValid    = flag.Bool("only-valid", false, "skip registrations Anvisa does not mark as valid")
		chunkSize    = flag.Int("chunk", anvisa.DefaultChunkSize, "how many drugs per transaction")
		skipPackages = flag.Bool("skip-packages", false, "load only the drugs, not the packages and barcodes")
	)
	flag.Parse()

	if err := run(*medicamentos, *precos, *dryRun, *onlyValid, *skipPackages, *chunkSize); err != nil {
		log.Fatalf("anvisa-import: %v", err)
	}
}

func run(medicamentos, precos string, dryRun, onlyValid, skipPackages bool, chunkSize int) error {
	// Ctrl+C stops between chunks: the one in flight is rolled back, and what
	// was already committed stays. Running the command again carries on.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	drugs, drugErrors, err := readDrugs(medicamentos, onlyValid)
	if err != nil {
		return err
	}
	log.Printf("%s: %d drugs read, %d with an active registration",
		anvisa.MedicamentosFile, len(drugs.Drugs), len(drugs.Active))
	report(drugErrors)

	var packages []importer.PackageRow
	if !skipPackages {
		var packageErrors []importer.RowError
		packages, packageErrors, err = readPackages(precos)
		if err != nil {
			return err
		}
		// The same barcode is printed on an old and a new registration of the
		// same product; the active one keeps it.
		var conflicts []importer.RowError
		packages, conflicts = anvisa.ResolveBarcodes(packages, drugs.Active)
		log.Printf("%s: %d packages read", anvisa.PrecosFile, len(packages))
		report(append(packageErrors, conflicts...))
	}

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("could not reach the database: %w", err)
	}

	files := anvisa.Chunks(drugs.Drugs, packages, chunkSize)
	if dryRun {
		log.Printf("dry run: every chunk is rolled back, nothing is written")
	}
	return apply(ctx, service.NewImportService(pool), files, dryRun)
}

// apply writes the chunks one at a time, printing progress: the whole base
// takes a few minutes and a silent command looks hung.
func apply(ctx context.Context, svc *service.ImportService, files []*importer.File, dryRun bool) error {
	started := time.Now()
	var total service.ImportResult
	var failures []importer.RowError

	for i, file := range files {
		if ctx.Err() != nil {
			log.Printf("interrupted: %d of %d chunks applied", i, len(files))
			break
		}

		result, err := svc.Run(ctx, file, dryRun)
		if err != nil {
			return fmt.Errorf("chunk %d/%d: %w", i+1, len(files), err)
		}
		add(&total, result)
		failures = append(failures, result.Errors...)

		if (i+1)%10 == 0 || i+1 == len(files) {
			log.Printf("chunk %d/%d - %d drugs, %d packages, %d barcodes (%s)",
				i+1, len(files),
				total.Drugs.Created+total.Drugs.Updated,
				total.Packages.Created+total.Packages.Updated,
				total.Eans.Created+total.Eans.Updated,
				time.Since(started).Round(time.Second))
		}
	}

	log.Printf("done in %s", time.Since(started).Round(time.Second))
	line("drugs", total.Drugs)
	line("packages", total.Packages)
	line("barcodes", total.Eans)
	report(failures)
	if dryRun {
		log.Printf("dry run: nothing was written")
	}
	return nil
}

func readDrugs(source string, onlyValid bool) (anvisa.Medicamentos, []importer.RowError, error) {
	r, err := open(source)
	if err != nil {
		return anvisa.Medicamentos{}, nil, err
	}
	defer r.Close()
	return anvisa.ParseMedicamentos(r, onlyValid)
}

func readPackages(source string) ([]importer.PackageRow, []importer.RowError, error) {
	r, err := open(source)
	if err != nil {
		return nil, nil, err
	}
	defer r.Close()
	return anvisa.ParsePrecos(r)
}

func open(source string) (io.ReadCloser, error) {
	log.Printf("reading %s", source)
	r, err := anvisa.Open(source)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", source, err)
	}
	return r, nil
}

func add(total *service.ImportResult, one service.ImportResult) {
	total.Drugs.Created += one.Drugs.Created
	total.Drugs.Updated += one.Drugs.Updated
	total.Packages.Created += one.Packages.Created
	total.Packages.Updated += one.Packages.Updated
	total.Eans.Created += one.Eans.Created
	total.Eans.Updated += one.Eans.Updated
}

func line(what string, counts service.ImportCounts) {
	log.Printf("%-10s %6d new, %6d updated", what, counts.Created, counts.Updated)
}

// digitsInMessage blanks the numbers of a message so rows that differ only by a
// line number are counted together.
var digitsInMessage = regexp.MustCompile(`[0-9]+`)

// report prints the rows that could not be used. The Anvisa files have tens of
// thousands of lines and the same problem repeats on thousands of them, so the
// rows are grouped by reason with one example line, instead of scrolling past.
func report(errs []importer.RowError) {
	if len(errs) == 0 {
		return
	}

	type group struct {
		count   int
		example importer.RowError
	}
	groups := map[string]*group{}
	var order []string
	for _, e := range errs {
		// Line numbers inside the message (which row claimed a barcode first)
		// would make every row its own group, so they are stripped from the key.
		key := e.Sheet + "|" + e.Column + "|" + digitsInMessage.ReplaceAllString(e.Message, "N")
		g, seen := groups[key]
		if !seen {
			g = &group{example: e}
			groups[key] = g
			order = append(order, key)
		}
		g.count++
	}

	sort.Slice(order, func(i, j int) bool { return groups[order[i]].count > groups[order[j]].count })

	log.Printf("%d rows ignored:", len(errs))
	for i, key := range order {
		if i == maxReportedErrors {
			log.Printf("  ... and %d other reasons", len(order)-maxReportedErrors)
			break
		}
		g := groups[key]
		log.Printf("  %6d x %s (%s), e.g. line %d", g.count, g.example.Message, g.example.Column, g.example.Line)
	}
}
