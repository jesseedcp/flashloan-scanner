package main

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli/v2"

	httpapi "github.com/cpchain-network/flashloan-scanner/api/http"
	"github.com/cpchain-network/flashloan-scanner/common/cliapp"
	"github.com/cpchain-network/flashloan-scanner/common/opio"
	"github.com/cpchain-network/flashloan-scanner/config"
	"github.com/cpchain-network/flashloan-scanner/database"
	"github.com/cpchain-network/flashloan-scanner/database/create_table"
	scannerapp "github.com/cpchain-network/flashloan-scanner/scanner/app"
	"github.com/cpchain-network/flashloan-scanner/scanner/fixture"
	scannerreport "github.com/cpchain-network/flashloan-scanner/scanner/report"
)

var (
	ConfigFlag = &cli.StringFlag{
		Name:    "config",
		Value:   "./flashloan-scanner.yaml",
		Aliases: []string{"c"},
		Usage:   "path to config file",
		EnvVars: []string{"FLASHLOAN_SCANNER_CONFIG"},
	}
	MigrationsFlag = &cli.StringFlag{
		Name:    "migrations-dir",
		Value:   "./migrations",
		Usage:   "path to migrations folder",
		EnvVars: []string{"FLASHLOAN_SCANNER_MIGRATIONS_DIR"},
	}
	ChainIDFlag = &cli.Uint64Flag{
		Name:    "chain-id",
		Usage:   "target chain id for scanner result queries",
		EnvVars: []string{"FLASHLOAN_SCANNER_CHAIN_ID"},
	}
	TxHashFlag = &cli.StringFlag{
		Name:    "tx-hash",
		Usage:   "specific transaction hash to inspect",
		EnvVars: []string{"FLASHLOAN_SCANNER_TX_HASH"},
	}
	LimitFlag = &cli.IntFlag{
		Name:    "limit",
		Value:   10,
		Usage:   "maximum number of flashloan transactions to print",
		EnvVars: []string{"FLASHLOAN_SCANNER_REPORT_LIMIT"},
	}
	StrictOnlyFlag = &cli.BoolFlag{
		Name:    "strict-only",
		Usage:   "show only transactions that contain verified strict interactions",
		EnvVars: []string{"FLASHLOAN_SCANNER_STRICT_ONLY"},
	}
	FormatFlag = &cli.StringFlag{
		Name:    "format",
		Value:   "text",
		Usage:   "output format: text, json, csv",
		EnvVars: []string{"FLASHLOAN_SCANNER_REPORT_FORMAT"},
	}
	OutputFlag = &cli.StringFlag{
		Name:    "output",
		Usage:   "optional output file path; defaults to stdout",
		EnvVars: []string{"FLASHLOAN_SCANNER_REPORT_OUTPUT"},
	}
)

func runFlashloanScanner(ctx *cli.Context, shutdown context.CancelCauseFunc) (cliapp.Lifecycle, error) {
	log.Info("running flashloan scanner...")
	cfg, err := config.New(ctx.String(ConfigFlag.Name))
	if err != nil {
		log.Error("failed to load config", "err", err)
		return nil, err
	}
	return scannerapp.NewFlashloanScannerService(ctx.Context, cfg, shutdown)
}

func runScanConsoleServer(ctx *cli.Context, _ context.CancelCauseFunc) (cliapp.Lifecycle, error) {
	log.Info("running scan console server...")
	cfg, err := config.New(ctx.String(ConfigFlag.Name))
	if err != nil {
		log.Error("failed to load config", "err", err)
		return nil, err
	}
	return httpapi.NewServer(ctx.Context, cfg)
}

func runMigrations(ctx *cli.Context) error {
	ctx.Context = opio.CancelOnInterrupt(ctx.Context)
	log.Info("running migrations...")
	cfg, err := config.New(ctx.String(ConfigFlag.Name))
	if err != nil {
		log.Error("failed to load config", "err", err)
		return err
	}
	db, err := database.NewDB(ctx.Context, cfg.MasterDb)
	if err != nil {
		log.Error("failed to connect to database", "err", err)
		return err
	}
	defer func(db *database.DB) {
		err := db.Close()
		if err != nil {
			return
		}
	}(db)
	err = db.ExecuteSQLMigration(ctx.String(MigrationsFlag.Name))
	if err != nil {
		return err
	}
	for i := range cfg.RPCs {
		log.Info("create chain table by chainId", "chainId", cfg.RPCs[i].ChainId)
		create_table.CreateTableFromTemplate(strconv.Itoa(int(cfg.RPCs[i].ChainId)), db)
	}
	log.Info("running migrations and create table from template success")
	return nil
}

func runFlashloanFixture(ctx *cli.Context) error {
	ctx.Context = opio.CancelOnInterrupt(ctx.Context)
	log.Info("loading flashloan fixture...")
	cfg, err := config.New(ctx.String(ConfigFlag.Name))
	if err != nil {
		log.Error("failed to load config", "err", err)
		return err
	}
	db, err := database.NewDB(ctx.Context, cfg.MasterDb)
	if err != nil {
		log.Error("failed to connect to database", "err", err)
		return err
	}
	defer func(db *database.DB) {
		_ = db.Close()
	}(db)

	switch cfg.Scanner.Protocol {
	case "aave_v3":
		if len(cfg.Scanner.Aave.Pools) == 0 {
			return fmt.Errorf("scanner.aave.pools must contain at least one pool address")
		}
		record := fixture.DefaultAaveV3FlashLoanSimpleFixture(
			cfg.Scanner.ChainID,
			common.HexToAddress(cfg.Scanner.Aave.Pools[0]),
			cfg.Scanner.StartBlock,
		)
		if err := fixture.LoadAaveV3FlashLoanSimpleFixture(db, record); err != nil {
			return err
		}
	case "balancer_v2":
		if len(cfg.Scanner.Balancer.Vaults) == 0 {
			return fmt.Errorf("scanner.balancer.vaults must contain at least one vault address")
		}
		record := fixture.DefaultBalancerV2FlashLoanFixture(
			cfg.Scanner.ChainID,
			common.HexToAddress(cfg.Scanner.Balancer.Vaults[0]),
			cfg.Scanner.StartBlock,
		)
		if err := fixture.LoadBalancerV2FlashLoanFixture(db, record); err != nil {
			return err
		}
	case "uniswap_v2":
		if len(cfg.Scanner.UniswapV2.Pairs) == 0 {
			return fmt.Errorf("scanner.uniswap_v2.pairs must contain at least one pair record")
		}
		pairCfg := cfg.Scanner.UniswapV2.Pairs[0]
		record := fixture.DefaultUniswapV2FlashSwapFixture(
			cfg.Scanner.ChainID,
			common.HexToAddress(pairCfg.FactoryAddress),
			common.HexToAddress(pairCfg.PairAddress),
			common.HexToAddress(pairCfg.Token0),
			common.HexToAddress(pairCfg.Token1),
			cfg.Scanner.StartBlock,
		)
		if err := fixture.LoadUniswapV2FlashSwapFixture(db, record); err != nil {
			return err
		}
	default:
		return fmt.Errorf("fixture loader currently supports only aave_v3, balancer_v2, and uniswap_v2, got %s", cfg.Scanner.Protocol)
	}

	log.Info("flashloan fixture loaded", "protocol", cfg.Scanner.Protocol, "chainId", cfg.Scanner.ChainID)
	return nil
}

func runFlashloanReport(ctx *cli.Context) error {
	ctx.Context = opio.CancelOnInterrupt(ctx.Context)
	cfg, err := config.New(ctx.String(ConfigFlag.Name))
	if err != nil {
		log.Error("failed to load config", "err", err)
		return err
	}
	db, err := database.NewDB(ctx.Context, cfg.MasterDb)
	if err != nil {
		log.Error("failed to connect to database", "err", err)
		return err
	}
	defer func(db *database.DB) {
		_ = db.Close()
	}(db)

	chainID := ctx.Uint64(ChainIDFlag.Name)
	if chainID == 0 {
		chainID = cfg.Scanner.ChainID
	}
	if chainID == 0 {
		return fmt.Errorf("chain id is required")
	}

	service := scannerreport.NewService(db)
	reports, err := service.LoadReports(scannerreport.Options{
		ChainID:    chainID,
		TxHash:     ctx.String(TxHashFlag.Name),
		Limit:      ctx.Int(LimitFlag.Name),
		StrictOnly: ctx.Bool(StrictOnlyFlag.Name),
	})
	if err != nil {
		return err
	}

	var output string
	switch ctx.String(FormatFlag.Name) {
	case "text":
		output = scannerreport.RenderText(reports)
	case "json":
		rendered, err := scannerreport.RenderJSON(reports)
		if err != nil {
			return err
		}
		output = rendered
	case "csv":
		return writeCSVOutput(ctx.String(OutputFlag.Name), reports)
	default:
		return fmt.Errorf("unsupported format: %s", ctx.String(FormatFlag.Name))
	}

	return writeTextOutput(ctx.String(OutputFlag.Name), output)
}

func writeTextOutput(outputPath, content string) error {
	if outputPath == "" {
		_, err := fmt.Fprintln(os.Stdout, content)
		return err
	}
	return os.WriteFile(outputPath, []byte(content+"\n"), 0o644)
}

func writeCSVOutput(outputPath string, reports []scannerreport.TransactionReport) error {
	if outputPath == "" {
		return scannerreport.WriteCSV(os.Stdout, reports)
	}
	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)
	return scannerreport.WriteCSV(file, reports)
}

func newCli(GitCommit string, GitDate string) *cli.App {
	flags := []cli.Flag{ConfigFlag}
	migrationFlags := []cli.Flag{MigrationsFlag, ConfigFlag}
	reportFlags := []cli.Flag{ConfigFlag, ChainIDFlag, TxHashFlag, LimitFlag, StrictOnlyFlag, FormatFlag, OutputFlag}
	return &cli.App{
		Version:              "v0.0.1",
		Description:          "A flash-loan scanning application",
		EnableBashCompletion: true,
		Commands: []*cli.Command{
			{
				Name:        "flashloan-scan",
				Flags:       flags,
				Description: "Runs the flash-loan scanner service",
				Action:      cliapp.LifecycleCmd(runFlashloanScanner),
			},
			{
				Name:        "scan-console",
				Flags:       flags,
				Description: "Runs the Gin + WebSocket scan console backend",
				Action:      cliapp.LifecycleCmd(runScanConsoleServer),
			},
			{
				Name:        "migrate",
				Flags:       migrationFlags,
				Description: "Runs the database migrations",
				Action:      runMigrations,
			},
			{
				Name:        "flashloan-fixture",
				Flags:       flags,
				Description: "Loads a local flash-loan scanner fixture dataset",
				Action:      runFlashloanFixture,
			},
			{
				Name:        "flashloan-report",
				Flags:       reportFlags,
				Description: "Prints flash-loan scanner results from the database",
				Action:      runFlashloanReport,
			},
			{
				Name:        "version",
				Description: "print version",
				Action: func(ctx *cli.Context) error {
					cli.ShowVersion(ctx)
					return nil
				},
			},
		},
	}
}
