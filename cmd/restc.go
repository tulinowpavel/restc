package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"log/slog"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"

	"github.com/tulinowpavel/restc"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	projectRootFlag := flag.String("path", cwd, "project root path")
	filePatternFlag := flag.String("pattern", `\.go$`, "pattern for files with controllers")
	outputFlag := flag.String("output", ".", "output path")
	useShellFlag := flag.Bool("shell", false, "invoke plugin via system shell")
	pluginFlag := flag.String("plugin", "", "generator plugin")

	flag.Parse()

	projectRoot, err := filepath.Abs(*projectRootFlag)
	if err != nil {
		logger.Error("cannot determine project root absolute path", "error", err)
		os.Exit(1)
	}

	patternRegex, err := regexp.Compile(*filePatternFlag)
	if err != nil {
		logger.Error("cannot compile controller file regex pattern", "error", err)
		os.Exit(1)
	}

	outputPath, err := filepath.Abs(*outputFlag)
	if err != nil {
		logger.Error("cannot determine output path", "error", err)
		os.Exit(1)
	}

	modRegex := regexp.MustCompile(`module (.+)`)
	modFile, err := os.ReadFile(path.Join(projectRoot, "go.mod"))
	if err != nil {
		logger.Error("cannot read go.mod file", "error", err)
		os.Exit(1)
	}

	subMatches := modRegex.FindStringSubmatch(string(modFile))
	if len(subMatches) != 2 {
		logger.Error("cannot find module name in go.mod file", "match", subMatches)
		os.Exit(1)
	}

	moduleName := string(subMatches[1])
	if moduleName == "" {
		logger.Error("empty module name in go.mod file")
		os.Exit(1)
	}

	rg := restc.NewRestCompilerAnalyzer(logger, moduleName, projectRoot, patternRegex)
	rg.Analyze()

	logger.Info("invoke generator plugin", "plugin", *pluginFlag)

	def, _ := json.Marshal(rg.Definitions)

	var cmd *exec.Cmd
	if *useShellFlag {
		// TODO: detect system shell
		cmd = exec.Command("zsh", "-c", *pluginFlag)
	} else {
		cmd = exec.Command("restc-" + *pluginFlag)
	}

	cmd.Env = append(
		os.Environ(),
		"RESTC_OUTPUT="+outputPath,
	)

	cmd.Stdin = bytes.NewBuffer(def)
	cmd.Stdout = os.Stdout

	if err := cmd.Run(); err != nil {
		logger.Error("invoke generator plugin error", "error", err)
		os.Exit(0)
	}

	// TODO: write output into file
}
