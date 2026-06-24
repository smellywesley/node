package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"gopkg.in/yaml.v3"

	agentapi "github.com/agentos/agentos/internal/api"
	"github.com/agentos/agentos/internal/core"
	"github.com/agentos/agentos/internal/model"
	"github.com/agentos/agentos/internal/runner"
	"github.com/agentos/agentos/internal/state"
	"github.com/agentos/agentos/internal/store"
)

const defaultAddress = "127.0.0.1:7467"

type config struct {
	home    string
	address string
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "agentos:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return usage()
	}
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	switch args[0] {
	case "serve":
		return serve(cfg, args[1:])
	case "dashboard":
		return dashboard(cfg, args[1:])
	case "doctor":
		return doctor(cfg, args[1:])
	case "rotate-token":
		return rotateToken(cfg, args[1:])
	case "validate":
		return validateManifestCommand(args[1:])
	case "version":
		return versionCommand(args[1:])
	case "run":
		if len(args) != 2 {
			return errors.New("usage: agentos run <manifest.yaml>")
		}
		raw, err := os.ReadFile(args[1])
		if err != nil {
			return err
		}
		var manifest model.Manifest
		if err = yaml.Unmarshal(raw, &manifest); err != nil {
			return err
		}
		return request(cfg, http.MethodPost, "/v1/processes", manifest, true)
	case "ps":
		return request(cfg, http.MethodGet, "/v1/processes", nil, true)
	case "inspect":
		return oneID(cfg, args, http.MethodGet, "/v1/processes/%s")
	case "logs":
		return oneID(cfg, args, http.MethodGet, "/v1/processes/%s/events")
	case "replay":
		return oneID(cfg, args, http.MethodGet, "/v1/processes/%s/replay")
	case "audit":
		return audit(cfg, args)
	case "suspend", "resume", "cancel":
		if len(args) != 2 {
			return fmt.Errorf("usage: agentos %s <process-id>", args[0])
		}
		return request(cfg, http.MethodPost, fmt.Sprintf("/v1/processes/%s/%s", args[1], args[0]), map[string]any{}, true)
	case "approvals":
		return request(cfg, http.MethodGet, "/v1/approvals", nil, true)
	case "approve", "deny":
		return approval(cfg, args)
	default:
		return usage()
	}
}

func dashboard(cfg config, args []string) error {
	flags := flag.NewFlagSet("dashboard", flag.ContinueOnError)
	printURL := flags.Bool("print-url", false, "print the credential-bearing dashboard URL instead of opening it")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("usage: agentos dashboard [--print-url]")
	}
	if err := daemonHealthy(cfg); err != nil {
		return fmt.Errorf("dashboard requires a running daemon: %w", err)
	}
	tokenRaw, err := os.ReadFile(filepath.Join(cfg.home, "token"))
	if err != nil {
		return fmt.Errorf("read daemon token: %w", err)
	}
	target := dashboardURL(
		cfg.address,
		strings.TrimSpace(string(tokenRaw)),
		strings.TrimSpace(os.Getenv("AGENTOS_APPROVER_TOKEN")),
	)
	if *printURL {
		fmt.Println(target)
		return nil
	}
	if err = openBrowser(target); err != nil {
		return fmt.Errorf("open dashboard: %w; use `agentos dashboard --print-url` instead", err)
	}
	fmt.Printf("Opened AgentOS dashboard at http://%s/\n", cfg.address)
	return nil
}

func daemonHealthy(cfg config) error {
	client := &http.Client{Timeout: 3 * time.Second}
	response, err := client.Get("http://" + cfg.address + "/v1/health")
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("health endpoint returned %s", response.Status)
	}
	return nil
}

func rotateToken(cfg config, args []string) error {
	flags := flag.NewFlagSet("rotate-token", flag.ContinueOnError)
	force := flags.Bool("force", false, "rotate while daemon is running; restart daemon before using the new token")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("usage: agentos rotate-token [--force]")
	}
	if !*force {
		if err := daemonHealthy(cfg); err == nil {
			return errors.New("daemon is running; stop it before rotating the operator token, or use --force and restart the daemon before reconnecting")
		}
	}
	if err := state.EnsureDir(cfg.home); err != nil {
		return err
	}
	if _, err := agentapi.RotateToken(filepath.Join(cfg.home, "token")); err != nil {
		return err
	}
	fmt.Printf("operator token rotated at %s\n", filepath.Join(cfg.home, "token"))
	fmt.Println("restart any running daemon and reconnect dashboards before using the new token")
	return nil
}

func dashboardURL(address, token, approverToken string) string {
	target := url.URL{Scheme: "http", Host: address, Path: "/"}
	credentials := url.Values{"token": []string{token}}
	if approverToken != "" {
		credentials.Set("approver_token", approverToken)
	}
	target.Fragment = credentials.Encode()
	return target.String()
}

func openBrowser(target string) error {
	var command *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		command = exec.Command("rundll32.exe", "url.dll,FileProtocolHandler", target)
	case "darwin":
		command = exec.Command("open", target)
	default:
		command = exec.Command("xdg-open", target)
	}
	return command.Start()
}

func loadConfig() (config, error) {
	home := os.Getenv("AGENTOS_HOME")
	if home == "" {
		userHome, err := os.UserHomeDir()
		if err != nil {
			return config{}, err
		}
		home = filepath.Join(userHome, ".agentos")
	}
	address := os.Getenv("AGENTOS_ADDR")
	if address == "" {
		address = defaultAddress
	}
	return config{home: home, address: address}, nil
}

func validateStateHome(home string) error {
	return state.ValidateHome(home)
}

func serve(cfg config, args []string) error {
	flags := flag.NewFlagSet("serve", flag.ContinueOnError)
	address := flags.String("addr", cfg.address, "loopback listen address")
	concurrency := flags.Int("concurrency", 2, "maximum simultaneous agent containers")
	if err := flags.Parse(args); err != nil {
		return err
	}
	host, _, err := net.SplitHostPort(*address)
	if err != nil {
		return err
	}
	if host != "127.0.0.1" && host != "::1" && host != "localhost" {
		return errors.New("v1 daemon must bind to loopback")
	}
	if err = state.EnsureDir(cfg.home); err != nil {
		return err
	}
	token, err := agentapi.EnsureToken(filepath.Join(cfg.home, "token"))
	if err != nil {
		return err
	}
	approverToken, err := approverToken()
	if err != nil {
		return err
	}
	db, err := store.Open(filepath.Join(cfg.home, "agentos.db"))
	if err != nil {
		return err
	}
	defer db.Close()
	service := core.New(db, runner.NewDocker(), *concurrency)
	defer service.Close()
	if err = service.Recover(context.Background()); err != nil {
		return err
	}
	server := &http.Server{
		Addr:              *address,
		Handler:           agentapi.New(service, token, approverToken),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	errs := make(chan error, 1)
	go func() {
		fmt.Printf("AgentOS daemon listening on http://%s\n", *address)
		errs <- server.ListenAndServe()
	}()
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	select {
	case <-signals:
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return server.Shutdown(ctx)
	case err = <-errs:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

func request(cfg config, method, path string, body any, pretty bool) error {
	tokenFile := "token"
	if method == http.MethodPost && strings.HasPrefix(path, "/v1/approvals/") {
		token, tokenErr := approverToken()
		if tokenErr != nil {
			return tokenErr
		}
		return sendRequest(cfg, method, path, body, pretty, token)
	}
	tokenRaw, err := os.ReadFile(filepath.Join(cfg.home, tokenFile))
	if err != nil {
		return fmt.Errorf("read daemon token; is `agentos serve` running? %w", err)
	}
	return sendRequest(cfg, method, path, body, pretty, strings.TrimSpace(string(tokenRaw)))
}

func sendRequest(cfg config, method, path string, body any, pretty bool, token string) error {
	var payload io.Reader
	if body != nil {
		raw, marshalErr := json.Marshal(body)
		if marshalErr != nil {
			return marshalErr
		}
		payload = bytes.NewReader(raw)
	}
	req, err := http.NewRequest(method, "http://"+cfg.address+path, payload)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 35 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("%s: %s", resp.Status, strings.TrimSpace(string(raw)))
	}
	if !pretty {
		_, err = os.Stdout.Write(raw)
		return err
	}
	var decoded any
	if json.Unmarshal(raw, &decoded) == nil {
		formatted, _ := json.MarshalIndent(decoded, "", "  ")
		fmt.Println(string(formatted))
		return nil
	}
	fmt.Print(string(raw))
	return nil
}

func approverToken() (string, error) {
	token := strings.TrimSpace(os.Getenv("AGENTOS_APPROVER_TOKEN"))
	if len(token) < 32 {
		return "", errors.New("AGENTOS_APPROVER_TOKEN must be set to a secret of at least 32 characters")
	}
	return token, nil
}

func oneID(cfg config, args []string, method, template string) error {
	if len(args) != 2 {
		return fmt.Errorf("usage: agentos %s <process-id>", args[0])
	}
	return request(cfg, method, fmt.Sprintf(template, args[1]), nil, true)
}

func approval(cfg config, args []string) error {
	if len(args) < 2 || len(args) > 3 {
		return fmt.Errorf("usage: agentos %s <approval-id> [reason]", args[0])
	}
	decision := "approved"
	if args[0] == "deny" {
		decision = "denied"
	}
	reason := ""
	if len(args) == 3 {
		reason = args[2]
	}
	return request(cfg, http.MethodPost, fmt.Sprintf("/v1/approvals/%s/%s", args[1], decision),
		map[string]string{"reason": reason}, true)
}

func audit(cfg config, args []string) error {
	if len(args) < 2 || len(args) > 3 {
		return errors.New("usage: agentos audit <process-id> [output.json]")
	}
	if len(args) == 2 {
		return request(cfg, http.MethodGet, fmt.Sprintf("/v1/processes/%s/audit", args[1]), nil, true)
	}
	tokenRaw, err := os.ReadFile(filepath.Join(cfg.home, "token"))
	if err != nil {
		return err
	}
	req, _ := http.NewRequest(http.MethodGet, "http://"+cfg.address+fmt.Sprintf("/v1/processes/%s/audit", args[1]), nil)
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(string(tokenRaw)))
	resp, err := (&http.Client{Timeout: 35 * time.Second}).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("%s: %s", resp.Status, strings.TrimSpace(string(raw)))
	}
	var decoded any
	if err = json.Unmarshal(raw, &decoded); err != nil {
		return err
	}
	formatted, _ := json.MarshalIndent(decoded, "", "  ")
	return os.WriteFile(args[2], append(formatted, '\n'), 0o600)
}

func usage() error {
	return errors.New(`usage: agentos <command>

commands:
  serve [--addr 127.0.0.1:7467] [--concurrency 2]
  dashboard [--print-url]
  doctor
  rotate-token [--force]
  validate <manifest.yaml>
  version
  run <manifest.yaml>
  ps
  inspect <process-id>
  suspend|resume|cancel <process-id>
  approvals
  approve|deny <approval-id> [reason]
  logs <process-id>
  replay <process-id>
  audit <process-id> [output.json]`)
}
