package runner

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/agentos/agentos/internal/model"
)

type Output struct {
	ExitCode int
	Lines    []string
	Usage    model.Usage
}

type Runner interface {
	Run(context.Context, model.Process, ToolHandler, UsageHandler, func(string, any)) (Output, error)
	Cancel(context.Context, string) error
}

type ToolHandler func(context.Context, model.Process, model.ToolRequest) model.ToolResult
type UsageHandler func(context.Context, string, model.Usage) error

type Docker struct {
	Binary string
}

func NewDocker() *Docker {
	return &Docker{Binary: "docker"}
}

func (d *Docker) Run(
	ctx context.Context,
	process model.Process,
	handleTool ToolHandler,
	handleUsage UsageHandler,
	emit func(string, any),
) (Output, error) {
	image, err := d.resolveImage(ctx, process.Manifest.Image)
	if err != nil {
		return Output{}, err
	}
	args := []string{
		"run", "--rm", "-i", "--name", containerName(process.ID),
		"--read-only", "--cap-drop=ALL", "--security-opt=no-new-privileges",
		"--pids-limit=256", "--memory=1g", "--cpus=2",
		"--user", "65532:65532",
		"--network=none",
		"--tmpfs", "/tmp:rw,noexec,nosuid,size=128m",
		"-e", "AGENTOS_PROCESS_ID=" + process.ID,
		"-e", "AGENTOS_TASK=" + process.Manifest.Task,
		"-e", "AGENTOS_DECLARED_TOOLS=" + strings.Join(process.Manifest.Capabilities.Tools, ","),
	}
	cleanupNetwork := func() {}
	if len(process.Manifest.Capabilities.NetworkDestinations) > 0 {
		network, proxy, cleanup, networkErr := d.startEgressProxy(ctx, process)
		if networkErr != nil {
			return Output{}, networkErr
		}
		cleanupNetwork = cleanup
		defer cleanupNetwork()
		args = replaceNetwork(args, network)
		proxyURL := "http://" + proxy + ":8080"
		args = append(args, "-e", "HTTPS_PROXY="+proxyURL, "-e", "HTTP_PROXY="+proxyURL, "-e", "NO_PROXY=")
	}
	for _, mount := range process.Manifest.Mounts {
		source, err := secureHostPath(mount.Source, os.Getenv("AGENTOS_WORKSPACE_ROOT"))
		if err != nil {
			return Output{}, err
		}
		if !mountCovered(mount, process.Manifest.Capabilities) {
			return Output{}, fmt.Errorf("mount target %q is not covered by filesystem capabilities", mount.Target)
		}
		mode := "rw"
		if mount.ReadOnly {
			mode = "ro"
		}
		args = append(args, "--mount", fmt.Sprintf("type=bind,src=%s,dst=%s,%s", source, mount.Target, mode))
	}
	for name, value := range process.Manifest.Implementation.Env {
		if containsSecret(process.Manifest.Capabilities.Secrets, name) {
			continue
		}
		args = append(args, "-e", name+"="+value)
	}
	if process.Manifest.Model != "" {
		args = append(args,
			"-e", "OPENAI_MODEL="+process.Manifest.Model,
			"-e", "OPENAI_DEFAULT_MODEL="+process.Manifest.Model,
		)
	}
	for _, secretName := range process.Manifest.Capabilities.Secrets {
		if _, ok := os.LookupEnv(secretName); !ok {
			return Output{}, fmt.Errorf("declared secret %q is unavailable in daemon environment", secretName)
		}
		args = append(args, "-e", secretName)
	}
	args = append(args, image)
	args = append(args, process.Manifest.Implementation.Command...)

	cmd := exec.CommandContext(ctx, d.Binary, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return Output{}, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return Output{}, err
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return Output{}, err
	}
	if err = cmd.Start(); err != nil {
		return Output{}, err
	}
	taskContext := map[string]any{
		"process_id": process.ID,
		"model":      process.Manifest.Model,
		"budget":     process.Manifest.Budget,
	}
	if len(process.Checkpoint) > 0 {
		taskContext["checkpoint"] = process.Checkpoint
	}
	taskMessage, err := json.Marshal(map[string]any{
		"protocol": "agent-process-os/v1",
		"type":     "task",
		"id":       process.ID,
		"input":    process.Manifest.Task,
		"context":  taskContext,
	})
	if err != nil {
		return Output{}, err
	}
	if _, err = stdin.Write(append(taskMessage, '\n')); err != nil {
		return Output{}, err
	}
	interactiveTools := process.Manifest.Implementation.Adapter == "jsonlines-tools" ||
		process.Manifest.Implementation.Env["AGENTOS_RESULT_ARTIFACT"] != ""
	if !interactiveTools {
		if err = stdin.Close(); err != nil {
			return Output{}, err
		}
	}

	var lines []string
	var protocolError string
	var protocolResult bool
	var usage model.Usage
	var stdinMu sync.Mutex
	var stdinClose sync.Once
	closeStdin := func() {
		stdinClose.Do(func() { _ = stdin.Close() })
	}
	secretValues := collectSecretValues(process.Manifest.Capabilities.Secrets)
	var outputMu sync.Mutex
	done := make(chan struct{}, 2)
	read := func(stream string, r io.Reader) {
		defer func() { done <- struct{}{} }()
		scanner := bufio.NewScanner(r)
		scanner.Buffer(make([]byte, 64*1024), 16*1024*1024)
		for scanner.Scan() {
			line := redactSecrets(scanner.Text(), secretValues)
			outputMu.Lock()
			lines = append(lines, line)
			var structured map[string]any
			if json.Unmarshal([]byte(line), &structured) == nil {
				emit("worker."+stream, workerEventMetadata(structured))
				if stream == "stdout" {
					protocol, _ := structured["protocol"].(string)
					taskID, _ := structured["task_id"].(string)
					messageType, _ := structured["type"].(string)
					control := messageType == "result" || messageType == "error" ||
						messageType == "usage" || messageType == "tool_request" ||
						messageType == "checkpoint"
					if protocol != "agent-process-os/v1" || (control && taskID != process.ID) {
						outputMu.Unlock()
						continue
					}
					switch structured["type"] {
					case "checkpoint":
						emit("process.checkpoint", structured["checkpoint"])
					case "result":
						protocolResult = true
						if interactiveTools {
							closeStdin()
						}
					case "error":
						protocolError = fmt.Sprint(structured["error"])
						if interactiveTools {
							closeStdin()
						}
					case "tool_request":
						if !interactiveTools || handleTool == nil {
							protocolError = "worker requested a brokered tool on a non-tool adapter"
							go func() { _ = d.Cancel(context.Background(), process.ID) }()
							break
						}
						rawPayload, marshalErr := json.Marshal(structured["payload"])
						if marshalErr != nil {
							protocolError = "invalid tool request payload"
							break
						}
						request := model.ToolRequest{
							IdempotencyKey: fmt.Sprint(structured["idempotency_key"]),
							Action:         fmt.Sprint(structured["action"]),
							Resource:       fmt.Sprint(structured["resource"]),
							Payload:        rawPayload,
						}
						go func() {
							result := handleTool(ctx, process, request)
							response, marshalErr := json.Marshal(map[string]any{
								"protocol":        "agent-process-os/v1",
								"type":            "tool_result",
								"task_id":         process.ID,
								"idempotency_key": request.IdempotencyKey,
								"status":          result.Status,
								"output":          result.Output,
								"error":           result.Error,
							})
							if marshalErr != nil {
								return
							}
							stdinMu.Lock()
							defer stdinMu.Unlock()
							_, _ = stdin.Write(append(response, '\n'))
						}()
					case "usage":
						usage.Tokens = numberAsInt64(structured["tokens"])
						inputTokens := numberAsInt64(structured["input_tokens"])
						outputTokens := numberAsInt64(structured["output_tokens"])
						usage.CostUSD =
							(float64(inputTokens)*process.Manifest.Pricing.InputUSDPerMillion +
								float64(outputTokens)*process.Manifest.Pricing.OutputUSDPerMillion) / 1_000_000
						if handleUsage != nil {
							if usageErr := handleUsage(context.Background(), process.ID, usage); usageErr != nil {
								protocolError = usageErr.Error()
								go func() { _ = d.Cancel(context.Background(), process.ID) }()
							}
						}
						if process.Manifest.Budget.MaxTokens > 0 && usage.Tokens > process.Manifest.Budget.MaxTokens {
							protocolError = "token budget exceeded"
							go func() { _ = d.Cancel(context.Background(), process.ID) }()
						}
						if process.Manifest.Budget.MaxCostUSD > 0 && usage.CostUSD > process.Manifest.Budget.MaxCostUSD {
							protocolError = "cost budget exceeded"
							go func() { _ = d.Cancel(context.Background(), process.ID) }()
						}
					}
				}
			} else {
				emit("worker."+stream, map[string]any{"redacted": true, "bytes": len(line)})
			}
			outputMu.Unlock()
		}
		if scanErr := scanner.Err(); scanErr != nil {
			outputMu.Lock()
			if protocolError == "" {
				protocolError = stream + " framing error: " + scanErr.Error()
			}
			outputMu.Unlock()
			go func() { _ = d.Cancel(context.Background(), process.ID) }()
		}
	}
	go read("stdout", stdout)
	go read("stderr", stderr)
	waitErr := cmd.Wait()
	closeStdin()
	<-done
	<-done

	output := Output{Lines: lines, Usage: usage}
	if protocolError != "" {
		return output, fmt.Errorf("agent protocol error: %s", protocolError)
	}
	if waitErr == nil {
		if process.Manifest.Implementation.Adapter != "process" && !protocolResult {
			return output, errors.New("agent adapter exited without a result message")
		}
		return output, nil
	}
	var exitErr *exec.ExitError
	if errors.As(waitErr, &exitErr) {
		output.ExitCode = exitErr.ExitCode()
		return output, fmt.Errorf("container exited with code %d", output.ExitCode)
	}
	return output, waitErr
}

func workerEventMetadata(frame map[string]any) map[string]any {
	result := map[string]any{
		"type":     frame["type"],
		"protocol": frame["protocol"],
		"task_id":  frame["task_id"],
	}
	switch frame["type"] {
	case "ready":
		result["capabilities"] = frame["capabilities"]
	case "usage":
		result["tokens"] = frame["tokens"]
		result["input_tokens"] = frame["input_tokens"]
		result["output_tokens"] = frame["output_tokens"]
		result["requests"] = frame["requests"]
	case "tool_event":
		result["event"] = frame["event"]
		result["tool"] = frame["tool"]
		result["call_id"] = frame["call_id"]
	default:
		result["redacted"] = true
	}
	return result
}

func (d *Docker) Cancel(ctx context.Context, processID string) error {
	cmd := exec.CommandContext(ctx, d.Binary, "rm", "-f", containerName(processID))
	output, err := cmd.CombinedOutput()
	if err != nil && strings.Contains(strings.ToLower(string(output)), "no such container") {
		err = nil
	}
	proxyErr := d.removeIfExists(ctx, proxyName(processID))
	networkErr := exec.CommandContext(ctx, d.Binary, "network", "rm", networkName(processID)).Run()
	if err != nil {
		return err
	}
	if proxyErr != nil {
		return proxyErr
	}
	if networkErr != nil {
		output, inspectErr := exec.CommandContext(ctx, d.Binary, "network", "inspect", networkName(processID)).CombinedOutput()
		message := strings.ToLower(string(output))
		missing := strings.Contains(message, "no such network") || strings.Contains(message, "not found")
		if inspectErr == nil || !missing {
			return networkErr
		}
	}
	return nil
}

func (d *Docker) resolveImage(ctx context.Context, image string) (string, error) {
	if strings.Contains(image, "@sha256:") {
		return image, nil
	}
	cmd := exec.CommandContext(ctx, d.Binary, "image", "inspect", "--format", "{{json .RepoDigests}}|{{.Id}}", image)
	raw, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("resolve immutable image digest for %q: %w", image, err)
	}
	parts := strings.SplitN(strings.TrimSpace(string(raw)), "|", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("unexpected image inspection result for %q", image)
	}
	var digests []string
	_ = json.Unmarshal([]byte(parts[0]), &digests)
	if len(digests) > 0 && strings.Contains(digests[0], "@sha256:") {
		return digests[0], nil
	}
	if strings.HasPrefix(parts[1], "sha256:") {
		return parts[1], nil
	}
	return "", fmt.Errorf("image %q has no immutable digest or image ID", image)
}

func secureHostPath(path, workspaceRoot string) (string, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	info, err := os.Lstat(absolute)
	if err != nil {
		return "", err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("mount source %q is a symbolic link", path)
	}
	if workspaceRoot == "" {
		workspaceRoot, err = os.Getwd()
		if err != nil {
			return "", err
		}
	}
	root, err := filepath.Abs(workspaceRoot)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(root, absolute)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("mount source %q is outside AGENTOS_WORKSPACE_ROOT", path)
	}
	for current := absolute; ; current = filepath.Dir(current) {
		part, statErr := os.Lstat(current)
		if statErr != nil {
			return "", statErr
		}
		if part.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("mount source %q traverses a symbolic link", path)
		}
		if filepath.Clean(current) == filepath.Clean(root) {
			break
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", fmt.Errorf("mount source %q is not beneath workspace root", path)
		}
	}
	lower := strings.ToLower(absolute)
	if strings.Contains(lower, "docker.sock") || strings.Contains(lower, "docker_engine") {
		return "", errors.New("container runtime sockets cannot be mounted")
	}
	if runtime.GOOS == "windows" {
		if strings.HasPrefix(absolute, `\\`) || strings.Contains(absolute, `\\.\`) ||
			strings.Contains(absolute, `\\?\`) || strings.Contains(filepath.Base(absolute), ":") {
			return "", fmt.Errorf("mount source %q uses a disallowed Windows path form", path)
		}
	}
	return absolute, nil
}

func containerName(id string) string {
	return "agentos-" + strings.ToLower(strings.ReplaceAll(id, "_", "-"))
}

func networkName(id string) string { return containerName(id) + "-net" }
func proxyName(id string) string   { return containerName(id) + "-proxy" }

func containsSecret(values []string, name string) bool {
	for _, value := range values {
		if value == name {
			return true
		}
	}
	return false
}

func truncate(value string, max int) string {
	if len(value) <= max {
		return value
	}
	return value[:max]
}

func replaceNetwork(args []string, network string) []string {
	for i := 0; i < len(args); i++ {
		if strings.HasPrefix(args[i], "--network=") {
			args[i] = "--network=" + network
			return args
		}
		if args[i] == "--network" {
			if i+1 == len(args) {
				return append(args, network)
			}
			args[i+1] = network
			return args
		}
	}
	return append(args, "--network", network)
}

func mountCovered(mount model.Mount, capabilities model.Capabilities) bool {
	roots := capabilities.FilesystemRead
	if !mount.ReadOnly {
		roots = capabilities.FilesystemWrite
	}
	target := filepath.Clean(mount.Target)
	for _, root := range roots {
		cleanRoot := filepath.Clean(root)
		rel, err := filepath.Rel(cleanRoot, target)
		if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

func collectSecretValues(names []string) []string {
	var values []string
	for _, name := range names {
		if value := os.Getenv(name); value != "" {
			values = append(values, value)
		}
	}
	return values
}

func redactSecrets(value string, secrets []string) string {
	for _, secret := range secrets {
		value = strings.ReplaceAll(value, secret, "[REDACTED]")
	}
	return value
}

func numberAsInt64(value any) int64 {
	switch typed := value.(type) {
	case float64:
		return int64(typed)
	case int64:
		return typed
	case int:
		return int64(typed)
	default:
		return 0
	}
}

func numberAsFloat64(value any) float64 {
	switch typed := value.(type) {
	case float64:
		return typed
	case int64:
		return float64(typed)
	case int:
		return float64(typed)
	default:
		return 0
	}
}

func (d *Docker) startEgressProxy(ctx context.Context, process model.Process) (string, string, func(), error) {
	network := networkName(process.ID)
	proxy := proxyName(process.ID)
	if output, err := exec.CommandContext(ctx, d.Binary, "network", "create", "--internal", network).CombinedOutput(); err != nil {
		return "", "", nil, fmt.Errorf("create isolated egress network: %s: %w", strings.TrimSpace(string(output)), err)
	}
	cleanup := func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		_ = d.removeIfExists(cleanupCtx, proxy)
		_ = exec.CommandContext(cleanupCtx, d.Binary, "network", "rm", network).Run()
	}
	proxyImageName := os.Getenv("AGENTOS_PROXY_IMAGE")
	if proxyImageName == "" {
		proxyImageName = "python:3.12-alpine@sha256:236173eb74001afe2f60862de935b74fcbd00adfca247b2c27051a70a6a39a2d"
	}
	proxyImage, err := d.resolveImage(ctx, proxyImageName)
	if err != nil {
		if output, pullErr := exec.CommandContext(ctx, d.Binary, "pull", proxyImageName).CombinedOutput(); pullErr != nil {
			cleanup()
			return "", "", nil, fmt.Errorf("pull egress proxy image: %s: %w", strings.TrimSpace(string(output)), pullErr)
		}
		proxyImage, err = d.resolveImage(ctx, proxyImageName)
		if err != nil {
			cleanup()
			return "", "", nil, fmt.Errorf("resolve egress proxy image after pull: %w", err)
		}
	}
	allowed := strings.Join(process.Manifest.Capabilities.NetworkDestinations, ",")
	run := exec.CommandContext(ctx, d.Binary,
		"run", "-d", "--rm", "--name", proxy, "--network", network,
		"--read-only", "--cap-drop=ALL", "--security-opt=no-new-privileges",
		"--pids-limit=128", "--memory=128m", "--cpus=0.5",
		"--user", "65532:65532", "--tmpfs", "/tmp:rw,noexec,nosuid,size=16m",
		"-e", "ALLOWED_HOSTS="+allowed,
		proxyImage, "python", "-u", "-c", egressProxyPython,
	)
	if output, runErr := run.CombinedOutput(); runErr != nil {
		cleanup()
		return "", "", nil, fmt.Errorf("start egress proxy: %s: %w", strings.TrimSpace(string(output)), runErr)
	}
	if output, connectErr := exec.CommandContext(ctx, d.Binary, "network", "connect", "bridge", proxy).CombinedOutput(); connectErr != nil {
		cleanup()
		return "", "", nil, fmt.Errorf("connect egress proxy outbound: %s: %w", strings.TrimSpace(string(output)), connectErr)
	}
	for attempt := 0; attempt < 30; attempt++ {
		ready := exec.CommandContext(ctx, d.Binary, "exec", proxy, "python", "-c",
			"import socket; s=socket.create_connection(('127.0.0.1',8080),1); s.close()")
		if ready.Run() == nil {
			return network, proxy, cleanup, nil
		}
		select {
		case <-ctx.Done():
			cleanup()
			return "", "", nil, ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
	}
	cleanup()
	return "", "", nil, fmt.Errorf("egress proxy %s did not become ready", proxy)
}

func (d *Docker) removeIfExists(ctx context.Context, name string) error {
	output, err := exec.CommandContext(ctx, d.Binary, "rm", "-f", name).CombinedOutput()
	if err != nil && strings.Contains(strings.ToLower(string(output)), "no such container") {
		return nil
	}
	return err
}

const egressProxyPython = `
import ipaddress, os, select, socket, socketserver

ALLOWED = {h.strip().lower().rstrip(".") for h in os.environ.get("ALLOWED_HOSTS", "").split(",") if h.strip()}

def allowed_target(host, port):
    host = host.lower().rstrip(".")
    if host not in ALLOWED or port != 443:
        return None
    try:
        infos = socket.getaddrinfo(host, port, type=socket.SOCK_STREAM)
    except OSError:
        return None
    for info in infos:
        ip = ipaddress.ip_address(info[4][0])
        if not ip.is_global:
            continue
        return str(ip)
    return None

def read_exact(connection, length):
    chunks = []
    remaining = length
    while remaining:
        chunk = connection.recv(remaining)
        if not chunk:
            raise OSError("unexpected EOF")
        chunks.append(chunk)
        remaining -= len(chunk)
    return b"".join(chunks)

def client_hello_sni(record):
    if len(record) < 9 or record[0] != 22 or record[5] != 1:
        return None
    body = memoryview(record)[9:]
    offset = 2 + 32
    if offset >= len(body):
        return None
    session_length = body[offset]
    offset += 1 + session_length
    if offset + 2 > len(body):
        return None
    cipher_length = int.from_bytes(body[offset:offset + 2], "big")
    offset += 2 + cipher_length
    if offset >= len(body):
        return None
    compression_length = body[offset]
    offset += 1 + compression_length
    if offset + 2 > len(body):
        return None
    extension_length = int.from_bytes(body[offset:offset + 2], "big")
    offset += 2
    end = min(len(body), offset + extension_length)
    while offset + 4 <= end:
        extension_type = int.from_bytes(body[offset:offset + 2], "big")
        size = int.from_bytes(body[offset + 2:offset + 4], "big")
        offset += 4
        value = body[offset:offset + size]
        offset += size
        if extension_type != 0 or len(value) < 5:
            continue
        name_length = int.from_bytes(value[3:5], "big")
        return bytes(value[5:5 + name_length]).decode("ascii", "strict").lower().rstrip(".")
    return None

class Handler(socketserver.StreamRequestHandler):
    def handle(self):
        line = self.rfile.readline(8192).decode("ascii", "replace").strip()
        parts = line.split()
        if len(parts) != 3 or parts[0].upper() != "CONNECT":
            self.wfile.write(b"HTTP/1.1 405 Method Not Allowed\r\nConnection: close\r\n\r\n")
            return
        target = parts[1].rsplit(":", 1)
        if len(target) != 2:
            self.wfile.write(b"HTTP/1.1 400 Bad Request\r\nConnection: close\r\n\r\n")
            return
        host = target[0].strip("[]")
        try:
            port = int(target[1])
        except ValueError:
            return
        while True:
            header = self.rfile.readline(8192)
            if header in (b"\r\n", b"\n", b""):
                break
        address = allowed_target(host, port)
        if address is None:
            self.wfile.write(b"HTTP/1.1 403 Forbidden\r\nConnection: close\r\n\r\n")
            return
        self.wfile.write(b"HTTP/1.1 200 Connection Established\r\n\r\n")
        self.wfile.flush()
        try:
            header = read_exact(self.connection, 5)
            record_length = int.from_bytes(header[3:5], "big")
            first_record = header + read_exact(self.connection, record_length)
            if client_hello_sni(first_record) != host.lower().rstrip("."):
                return
            upstream = socket.create_connection((address, port), timeout=10)
            upstream.sendall(first_record)
        except OSError:
            return
        sockets = [self.connection, upstream]
        try:
            while True:
                readable, _, _ = select.select(sockets, [], [], 30)
                if not readable:
                    break
                for source in readable:
                    data = source.recv(65536)
                    if not data:
                        return
                    (upstream if source is self.connection else self.connection).sendall(data)
        finally:
            upstream.close()

class Server(socketserver.ThreadingTCPServer):
    allow_reuse_address = True
    daemon_threads = True

Server(("0.0.0.0", 8080), Handler).serve_forever()
`
