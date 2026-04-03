package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const defaultBaseURL = "http://127.0.0.1:8080"

type cli struct {
	baseURL    string
	username   string
	password   string
	outputJSON bool
	client     *http.Client
	loggedIn   bool
}

func main() {
	baseURL := envOrDefault("WAF_CLI_BASE_URL", defaultBaseURL)
	username := envOrDefault("WAF_CLI_USERNAME", envOrDefault("CONTROL_PLANE_BOOTSTRAP_ADMIN_USERNAME", "admin"))
	password := envOrDefault("WAF_CLI_PASSWORD", envOrDefault("CONTROL_PLANE_BOOTSTRAP_ADMIN_PASSWORD", "admin"))
	insecure := false
	noAuth := false
	outputJSON := false

	global := flag.NewFlagSet("waf-cli", flag.ExitOnError)
	global.StringVar(&baseURL, "base-url", baseURL, "control-plane base URL")
	global.StringVar(&username, "username", username, "control-plane username")
	global.StringVar(&password, "password", password, "control-plane password")
	global.BoolVar(&insecure, "insecure", false, "skip TLS verification")
	global.BoolVar(&noAuth, "no-auth", false, "skip auth for command")
	global.BoolVar(&outputJSON, "json", false, "print raw JSON responses")
	global.Usage = usage
	global.Parse(os.Args[1:])

	args := global.Args()
	if len(args) == 0 {
		usage()
		os.Exit(1)
	}

	httpClient, err := newHTTPClient(insecure)
	if err != nil {
		fatalf("init http client: %v", err)
	}
	tool := &cli{
		baseURL:    strings.TrimRight(baseURL, "/"),
		username:   username,
		password:   password,
		outputJSON: outputJSON,
		client:     httpClient,
	}

	if err := dispatch(tool, args, noAuth); err != nil {
		fatalf("%v", err)
	}
}

func dispatch(c *cli, args []string, noAuth bool) error {
	switch args[0] {
	case "health":
		return cmdHealth(c)
	case "setup":
		return cmdSetup(c, args[1:], noAuth)
	case "me":
		return cmdAuthMe(c, noAuth)
	case "sites":
		return cmdSites(c, args[1:], noAuth)
	case "upstreams":
		return cmdList(c, "upstreams", "/api/upstreams", args[1:], noAuth)
	case "tls":
		return cmdList(c, "tls configs", "/api/tls-configs", args[1:], noAuth)
	case "certificates":
		return cmdList(c, "certificates", "/api/certificates", args[1:], noAuth)
	case "events":
		return cmdEvents(c, args[1:], noAuth)
	case "audit":
		return cmdAudit(c, args[1:], noAuth)
	case "access-policies":
		return cmdList(c, "access policies", "/api/access-policies", args[1:], noAuth)
	case "waf-policies":
		return cmdList(c, "waf policies", "/api/waf-policies", args[1:], noAuth)
	case "rate-limit-policies":
		return cmdList(c, "rate-limit policies", "/api/rate-limit-policies", args[1:], noAuth)
	case "easy":
		return cmdEasy(c, args[1:], noAuth)
	case "antiddos":
		return cmdAntiDDoS(c, args[1:], noAuth)
	case "ban":
		return cmdBanLike(c, "ban", args[1:], noAuth)
	case "unban":
		return cmdBanLike(c, "unban", args[1:], noAuth)
	case "bans":
		return cmdBans(c, args[1:], noAuth)
	case "revisions":
		return cmdRevisions(c, args[1:], noAuth)
	case "reports":
		return cmdReports(c, args[1:], noAuth)
	case "compile":
		return cmdRevisions(c, []string{"compile"}, noAuth)
	case "apply":
		return cmdRevisions(c, append([]string{"apply"}, args[1:]...), noAuth)
	case "api":
		return cmdAPI(c, args[1:], noAuth)
	default:
		return fmt.Errorf("unknown command: %s", args[0])
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "waf-cli: container CLI for Berkut Solutions - TARINIO control-plane")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Usage:")
	fmt.Fprintln(os.Stderr, "  waf-cli [global flags] <command>")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Global flags:")
	fmt.Fprintln(os.Stderr, "  --base-url http://127.0.0.1:8080")
	fmt.Fprintln(os.Stderr, "  --username admin")
	fmt.Fprintln(os.Stderr, "  --password admin")
	fmt.Fprintln(os.Stderr, "  --insecure")
	fmt.Fprintln(os.Stderr, "  --no-auth")
	fmt.Fprintln(os.Stderr, "  --json")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Core commands:")
	fmt.Fprintln(os.Stderr, "  health")
	fmt.Fprintln(os.Stderr, "  setup status")
	fmt.Fprintln(os.Stderr, "  me")
	fmt.Fprintln(os.Stderr, "  sites list | sites delete <id>")
	fmt.Fprintln(os.Stderr, "  upstreams list")
	fmt.Fprintln(os.Stderr, "  tls list")
	fmt.Fprintln(os.Stderr, "  certificates list")
	fmt.Fprintln(os.Stderr, "  events [--limit N]")
	fmt.Fprintln(os.Stderr, "  audit [--action A --site-id S --status ST --limit N --offset N]")
	fmt.Fprintln(os.Stderr, "  ban <ip> [--site control-plane-access]")
	fmt.Fprintln(os.Stderr, "  unban <ip> [--site control-plane-access]")
	fmt.Fprintln(os.Stderr, "  bans list [--site control-plane-access]")
	fmt.Fprintln(os.Stderr, "  access-policies list")
	fmt.Fprintln(os.Stderr, "  waf-policies list")
	fmt.Fprintln(os.Stderr, "  rate-limit-policies list")
	fmt.Fprintln(os.Stderr, "  easy get <site-id> | easy upsert <site-id> --file profile.json")
	fmt.Fprintln(os.Stderr, "  antiddos get | antiddos upsert --file antiddos.json")
	fmt.Fprintln(os.Stderr, "  revisions compile | revisions apply <rev-id>")
	fmt.Fprintln(os.Stderr, "  reports revisions")
	fmt.Fprintln(os.Stderr, "  api <GET|POST|PUT|DELETE> <path> [--file body.json]")
}

func cmdHealth(c *cli) error {
	status, body, err := c.rawRequest(http.MethodGet, "/healthz", nil, false)
	if err != nil {
		return err
	}
	if c.outputJSON {
		var payload any
		if json.Unmarshal(body, &payload) == nil {
			return printJSON(payload)
		}
		fmt.Printf("{\"status\":%d,\"body\":%q}\n", status, strings.TrimSpace(string(body)))
		return nil
	}
	fmt.Printf("Health: HTTP %d\n", status)
	fmt.Printf("Response: %s\n", strings.TrimSpace(string(body)))
	return nil
}

func cmdSetup(c *cli, args []string, noAuth bool) error {
	if len(args) < 1 || args[0] != "status" {
		return fmt.Errorf("usage: waf-cli setup status")
	}
	value, err := c.requestJSON(http.MethodGet, "/api/setup/status", nil, !noAuth)
	if err != nil {
		return err
	}
	if c.outputJSON {
		return printJSON(value)
	}
	item, ok := value.(map[string]any)
	if !ok {
		return printJSON(value)
	}
	fmt.Println("Setup status:")
	printKV(
		"needs_bootstrap", stringify(item["needs_bootstrap"]),
		"bootstrap_allowed", stringify(item["bootstrap_allowed"]),
		"users_count", stringify(item["users_count"]),
		"needs_2fa", stringify(item["needs_2fa"]),
		"has_active_revision", stringify(item["has_active_revision"]),
	)
	return nil
}

func cmdAuthMe(c *cli, noAuth bool) error {
	value, err := c.requestJSON(http.MethodGet, "/api/auth/me", nil, !noAuth)
	if err != nil {
		return err
	}
	if c.outputJSON {
		return printJSON(value)
	}
	item, ok := value.(map[string]any)
	if !ok {
		return printJSON(value)
	}
	fmt.Println("Current user:")
	printKV(
		"id", stringify(item["id"]),
		"username", stringify(item["username"]),
		"email", stringify(item["email"]),
		"role", stringify(item["role"]),
		"totp_enabled", stringify(item["totp_enabled"]),
	)
	return nil
}

func cmdSites(c *cli, args []string, noAuth bool) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: waf-cli sites list | sites delete <id>")
	}
	switch args[0] {
	case "list":
		value, err := c.requestJSON(http.MethodGet, "/api/sites", nil, !noAuth)
		if err != nil {
			return err
		}
		return printSites(c, value)
	case "delete":
		if len(args) < 2 {
			return fmt.Errorf("usage: waf-cli sites delete <id>")
		}
		id := strings.TrimSpace(args[1])
		if id == "" {
			return fmt.Errorf("site id is required")
		}
		if err := c.requestNoContent(http.MethodDelete, "/api/sites/"+url.PathEscape(id), nil, !noAuth); err != nil {
			return err
		}
		fmt.Printf("Site deleted: %s\n", id)
		return nil
	default:
		return fmt.Errorf("usage: waf-cli sites list | sites delete <id>")
	}
}

func cmdList(c *cli, title, path string, args []string, noAuth bool) error {
	if len(args) < 1 || args[0] != "list" {
		return fmt.Errorf("usage: waf-cli %s list", strings.ReplaceAll(title, " ", "-"))
	}
	value, err := c.requestJSON(http.MethodGet, path, nil, !noAuth)
	if err != nil {
		return err
	}
	if c.outputJSON {
		return printJSON(value)
	}
	items := asList(value)
	fmt.Printf("%s: %d item(s)\n", strings.Title(title), len(items))
	if len(items) == 0 {
		return nil
	}
	return printTableFromMaps(items, []string{"id", "site_id", "enabled", "updated_at"})
}

func cmdEvents(c *cli, args []string, noAuth bool) error {
	limit := 20
	eventType := ""
	siteID := ""
	severity := ""

	fs := flag.NewFlagSet("events", flag.ExitOnError)
	fs.IntVar(&limit, "limit", 20, "max events")
	fs.StringVar(&eventType, "type", "", "filter by event type")
	fs.StringVar(&siteID, "site-id", "", "filter by site id")
	fs.StringVar(&severity, "severity", "", "filter by severity")
	fs.Parse(args)

	value, err := c.requestJSON(http.MethodGet, "/api/events", nil, !noAuth)
	if err != nil {
		return err
	}
	root, ok := value.(map[string]any)
	if !ok {
		return printJSON(value)
	}
	items := asList(root["events"])
	filtered := make([]map[string]any, 0, len(items))
	for _, item := range items {
		if eventType != "" && !strings.EqualFold(stringify(item["type"]), eventType) {
			continue
		}
		if siteID != "" && !strings.EqualFold(stringify(item["site_id"]), siteID) {
			continue
		}
		if severity != "" && !strings.EqualFold(stringify(item["severity"]), severity) {
			continue
		}
		filtered = append(filtered, item)
	}
	if limit > 0 && len(filtered) > limit {
		filtered = filtered[:limit]
	}
	if c.outputJSON {
		return printJSON(filtered)
	}
	fmt.Printf("Events: %d item(s)\n", len(filtered))
	if len(filtered) == 0 {
		return nil
	}
	return printTableFromMaps(filtered, []string{"occurred_at", "type", "severity", "site_id", "summary"})
}

func cmdAudit(c *cli, args []string, noAuth bool) error {
	fs := flag.NewFlagSet("audit", flag.ExitOnError)
	action := fs.String("action", "", "audit action")
	siteID := fs.String("site-id", "", "site id")
	status := fs.String("status", "", "status")
	limit := fs.Int("limit", 50, "limit")
	offset := fs.Int("offset", 0, "offset")
	_ = fs.Parse(args)

	query := url.Values{}
	if strings.TrimSpace(*action) != "" {
		query.Set("action", strings.TrimSpace(*action))
	}
	if strings.TrimSpace(*siteID) != "" {
		query.Set("site_id", strings.TrimSpace(*siteID))
	}
	if strings.TrimSpace(*status) != "" {
		query.Set("status", strings.TrimSpace(*status))
	}
	query.Set("limit", fmt.Sprintf("%d", *limit))
	query.Set("offset", fmt.Sprintf("%d", *offset))

	path := "/api/audit"
	if encoded := query.Encode(); encoded != "" {
		path += "?" + encoded
	}
	value, err := c.requestJSON(http.MethodGet, path, nil, !noAuth)
	if err != nil {
		return err
	}
	if c.outputJSON {
		return printJSON(value)
	}
	root, ok := value.(map[string]any)
	if !ok {
		return printJSON(value)
	}
	items := asList(root["items"])
	total := stringify(root["total"])
	fmt.Printf("Audit: %d item(s), total=%s\n", len(items), total)
	if len(items) == 0 {
		return nil
	}
	return printTableFromMaps(items, []string{"occurred_at", "action", "site_id", "status", "actor_ip"})
}

func cmdBanLike(c *cli, action string, args []string, noAuth bool) error {
	defaultSite := envOrDefault("WAF_CLI_DEFAULT_SITE", envOrDefault("CONTROL_PLANE_DEV_FAST_START_MANAGEMENT_SITE_ID", "control-plane-access"))
	siteID := defaultSite
	fs := flag.NewFlagSet(action, flag.ExitOnError)
	fs.StringVar(&siteID, "site", defaultSite, "site id")
	_ = fs.Parse(args)
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: waf-cli %s <ip> [--site %s]", action, defaultSite)
	}
	ip := strings.TrimSpace(fs.Arg(0))
	if ip == "" {
		return fmt.Errorf("ip is required")
	}
	path := "/api/sites/" + url.PathEscape(siteID) + "/" + action
	body := map[string]string{"ip": ip}
	value, err := c.requestJSON(http.MethodPost, path, body, !noAuth)
	if err != nil {
		return err
	}
	if c.outputJSON {
		return printJSON(value)
	}
	fmt.Printf("IP %s %sed for site %s\n", ip, action, siteID)
	if item, ok := value.(map[string]any); ok {
		deny := asStringSlice(item["denylist"])
		fmt.Printf("Active denylist size: %d\n", len(deny))
	}
	return nil
}

func cmdBans(c *cli, args []string, noAuth bool) error {
	if len(args) < 1 || args[0] != "list" {
		return fmt.Errorf("usage: waf-cli bans list [--site <id>]")
	}
	siteID := ""
	fs := flag.NewFlagSet("bans list", flag.ExitOnError)
	fs.StringVar(&siteID, "site", "", "site id")
	_ = fs.Parse(args[1:])

	value, err := c.requestJSON(http.MethodGet, "/api/access-policies", nil, !noAuth)
	if err != nil {
		return err
	}
	policies := asList(value)
	rows := make([]map[string]any, 0)
	for _, item := range policies {
		if siteID != "" && !strings.EqualFold(stringify(item["site_id"]), siteID) {
			continue
		}
		deny := asStringSlice(item["denylist"])
		for _, ip := range deny {
			rows = append(rows, map[string]any{
				"site_id":    stringify(item["site_id"]),
				"policy_id":  stringify(item["id"]),
				"ip":         ip,
				"updated_at": stringify(item["updated_at"]),
			})
		}
	}
	if c.outputJSON {
		return printJSON(rows)
	}
	fmt.Printf("Bans: %d item(s)\n", len(rows))
	if len(rows) == 0 {
		return nil
	}
	sort.Slice(rows, func(i, j int) bool {
		return stringify(rows[i]["site_id"])+stringify(rows[i]["ip"]) < stringify(rows[j]["site_id"])+stringify(rows[j]["ip"])
	})
	return printTableFromMaps(rows, []string{"site_id", "ip", "policy_id", "updated_at"})
}

func cmdEasy(c *cli, args []string, noAuth bool) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: waf-cli easy get <site-id> | easy upsert <site-id> --file profile.json")
	}
	switch args[0] {
	case "get":
		if len(args) < 2 {
			return fmt.Errorf("usage: waf-cli easy get <site-id>")
		}
		siteID := strings.TrimSpace(args[1])
		value, err := c.requestJSON(http.MethodGet, "/api/easy-site-profiles/"+url.PathEscape(siteID), nil, !noAuth)
		if err != nil {
			return err
		}
		return printJSON(value)
	case "upsert":
		fs := flag.NewFlagSet("easy upsert", flag.ExitOnError)
		file := fs.String("file", "", "profile json file")
		_ = fs.Parse(args[1:])
		if fs.NArg() < 1 {
			return fmt.Errorf("usage: waf-cli easy upsert <site-id> --file profile.json")
		}
		siteID := strings.TrimSpace(fs.Arg(0))
		if strings.TrimSpace(*file) == "" {
			return fmt.Errorf("--file is required")
		}
		payload, err := readJSONFile(*file)
		if err != nil {
			return err
		}
		value, err := c.requestJSON(http.MethodPut, "/api/easy-site-profiles/"+url.PathEscape(siteID), payload, !noAuth)
		if err != nil {
			return err
		}
		if c.outputJSON {
			return printJSON(value)
		}
		fmt.Printf("Easy profile upserted for site %s from %s\n", siteID, filepath.Clean(*file))
		return nil
	default:
		return fmt.Errorf("usage: waf-cli easy get <site-id> | easy upsert <site-id> --file profile.json")
	}
}

func cmdAntiDDoS(c *cli, args []string, noAuth bool) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: waf-cli antiddos get | antiddos upsert --file antiddos.json")
	}
	switch args[0] {
	case "get":
		value, err := c.requestJSON(http.MethodGet, "/api/anti-ddos/settings", nil, !noAuth)
		if err != nil {
			return err
		}
		return printJSON(value)
	case "upsert":
		fs := flag.NewFlagSet("antiddos upsert", flag.ExitOnError)
		file := fs.String("file", "", "settings json file")
		_ = fs.Parse(args[1:])
		if strings.TrimSpace(*file) == "" {
			return fmt.Errorf("--file is required")
		}
		payload, err := readJSONFile(*file)
		if err != nil {
			return err
		}
		value, err := c.requestJSON(http.MethodPut, "/api/anti-ddos/settings", payload, !noAuth)
		if err != nil {
			return err
		}
		if c.outputJSON {
			return printJSON(value)
		}
		fmt.Printf("Anti-DDoS settings upserted from %s\n", filepath.Clean(*file))
		return nil
	default:
		return fmt.Errorf("usage: waf-cli antiddos get | antiddos upsert --file antiddos.json")
	}
}

func cmdRevisions(c *cli, args []string, noAuth bool) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: waf-cli revisions compile | revisions apply <rev-id>")
	}
	switch args[0] {
	case "compile":
		value, err := c.requestJSON(http.MethodPost, "/api/revisions/compile", map[string]any{}, !noAuth)
		if err != nil {
			return err
		}
		if c.outputJSON {
			return printJSON(value)
		}
		root, ok := value.(map[string]any)
		if !ok {
			return printJSON(value)
		}
		revision := asMap(root["revision"])
		job := asMap(root["job"])
		fmt.Printf("Compiled revision: %s\n", stringify(revision["id"]))
		fmt.Printf("Compile job: %s (%s)\n", stringify(job["id"]), stringify(job["status"]))
		return nil
	case "apply":
		if len(args) < 2 {
			return fmt.Errorf("usage: waf-cli revisions apply <rev-id>")
		}
		revID := strings.TrimSpace(args[1])
		if revID == "" {
			return fmt.Errorf("revision id is required")
		}
		value, err := c.requestJSON(http.MethodPost, "/api/revisions/"+url.PathEscape(revID)+"/apply", map[string]any{}, !noAuth)
		if err != nil {
			return err
		}
		if c.outputJSON {
			return printJSON(value)
		}
		job := asMap(value)
		fmt.Printf("Apply requested for revision %s\n", revID)
		fmt.Printf("Apply job: %s (%s)\n", stringify(job["id"]), stringify(job["status"]))
		return nil
	default:
		return fmt.Errorf("usage: waf-cli revisions compile | revisions apply <rev-id>")
	}
}

func cmdReports(c *cli, args []string, noAuth bool) error {
	if len(args) < 1 || args[0] != "revisions" {
		return fmt.Errorf("usage: waf-cli reports revisions")
	}
	value, err := c.requestJSON(http.MethodGet, "/api/reports/revisions", nil, !noAuth)
	if err != nil {
		return err
	}
	if c.outputJSON {
		return printJSON(value)
	}
	root := asMap(value)
	fmt.Println("Revision report summary:")
	keys := make([]string, 0, len(root))
	for key := range root {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		fmt.Printf("  %s: %s\n", key, stringify(root[key]))
	}
	return nil
}

func cmdAPI(c *cli, args []string, noAuth bool) error {
	fs := flag.NewFlagSet("api", flag.ExitOnError)
	file := fs.String("file", "", "json body file")
	_ = fs.Parse(args)
	if fs.NArg() < 2 {
		return fmt.Errorf("usage: waf-cli api <GET|POST|PUT|DELETE> <path> [--file body.json]")
	}
	method := strings.ToUpper(strings.TrimSpace(fs.Arg(0)))
	path := strings.TrimSpace(fs.Arg(1))
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	var payload any
	var err error
	if strings.TrimSpace(*file) != "" {
		payload, err = readJSONFile(*file)
		if err != nil {
			return err
		}
	}
	value, err := c.requestJSON(method, path, payload, !noAuth)
	if err != nil {
		if strings.Contains(err.Error(), "HTTP 204") {
			fmt.Printf("%s %s -> 204 No Content\n", method, path)
			return nil
		}
		return err
	}
	return printJSON(value)
}

func printSites(c *cli, value any) error {
	if c.outputJSON {
		return printJSON(value)
	}
	items := asList(value)
	fmt.Printf("Sites: %d item(s)\n", len(items))
	if len(items) == 0 {
		return nil
	}
	return printTableFromMaps(items, []string{"id", "primary_host", "enabled", "updated_at"})
}

func printTableFromMaps(items []map[string]any, columns []string) error {
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		row := make([]string, 0, len(columns))
		for _, column := range columns {
			row = append(row, stringify(item[column]))
		}
		rows = append(rows, row)
	}
	headers := make([]string, len(columns))
	for i := range columns {
		headers[i] = columns[i]
	}
	renderTable(headers, rows)
	return nil
}

func renderTable(headers []string, rows [][]string) {
	if len(headers) == 0 {
		return
	}
	width := make([]int, len(headers))
	for i := range headers {
		width[i] = len(headers[i])
	}
	for _, row := range rows {
		for i := 0; i < len(headers) && i < len(row); i++ {
			if len(row[i]) > width[i] {
				width[i] = len(row[i])
			}
		}
	}

	printRow := func(cols []string) {
		for i := 0; i < len(headers); i++ {
			value := ""
			if i < len(cols) {
				value = cols[i]
			}
			fmt.Printf("%-*s", width[i], value)
			if i != len(headers)-1 {
				fmt.Print("  ")
			}
		}
		fmt.Println()
	}
	printSep := func() {
		for i := range headers {
			fmt.Print(strings.Repeat("-", width[i]))
			if i != len(headers)-1 {
				fmt.Print("  ")
			}
		}
		fmt.Println()
	}

	printRow(headers)
	printSep()
	for _, row := range rows {
		printRow(row)
	}
}

func printKV(pairs ...string) {
	for i := 0; i+1 < len(pairs); i += 2 {
		fmt.Printf("  %s: %s\n", pairs[i], pairs[i+1])
	}
}

func newHTTPClient(insecure bool) (*http.Client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	transport := &http.Transport{}
	if insecure {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	return &http.Client{
		Timeout:   20 * time.Second,
		Jar:       jar,
		Transport: transport,
	}, nil
}

func (c *cli) ensureLogin() error {
	if c.loggedIn {
		return nil
	}
	if strings.TrimSpace(c.username) == "" || strings.TrimSpace(c.password) == "" {
		return fmt.Errorf("username/password are required for auth commands")
	}
	payload := map[string]string{
		"username": c.username,
		"password": c.password,
	}
	status, body, err := c.rawRequest(http.MethodPost, "/api/auth/login", payload, false)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("login failed (%d): %s", status, extractErr(body))
	}
	c.loggedIn = true
	return nil
}

func (c *cli) requestJSON(method, path string, payload any, auth bool) (any, error) {
	if auth {
		if err := c.ensureLogin(); err != nil {
			return nil, err
		}
	}
	status, body, err := c.rawRequest(method, path, payload, false)
	if err != nil {
		return nil, err
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("%s %s failed (HTTP %d): %s", method, path, status, extractErr(body))
	}
	if status == http.StatusNoContent {
		return nil, fmt.Errorf("HTTP 204")
	}
	if len(body) == 0 {
		return map[string]any{}, nil
	}
	var out any
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return out, nil
}

func (c *cli) requestNoContent(method, path string, payload any, auth bool) error {
	if auth {
		if err := c.ensureLogin(); err != nil {
			return err
		}
	}
	status, body, err := c.rawRequest(method, path, payload, false)
	if err != nil {
		return err
	}
	if status < 200 || status >= 300 {
		return fmt.Errorf("%s %s failed (HTTP %d): %s", method, path, status, extractErr(body))
	}
	return nil
}

func (c *cli) rawRequest(method, path string, payload any, auth bool) (int, []byte, error) {
	if auth {
		if err := c.ensureLogin(); err != nil {
			return 0, nil, err
		}
	}
	var bodyReader io.Reader
	if payload != nil {
		raw, err := json.Marshal(payload)
		if err != nil {
			return 0, nil, err
		}
		bodyReader = bytes.NewReader(raw)
	}
	req, err := http.NewRequest(method, c.baseURL+path, bodyReader)
	if err != nil {
		return 0, nil, err
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, nil, err
	}
	return resp.StatusCode, body, nil
}

func readJSONFile(path string) (map[string]any, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file %s: %w", path, err)
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("decode json %s: %w", path, err)
	}
	return payload, nil
}

func extractErr(body []byte) string {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err == nil {
		if msg, ok := payload["error"].(string); ok && strings.TrimSpace(msg) != "" {
			return msg
		}
	}
	text := strings.TrimSpace(string(body))
	if text == "" {
		return "(empty response)"
	}
	return text
}

func asMap(value any) map[string]any {
	item, ok := value.(map[string]any)
	if !ok {
		return map[string]any{}
	}
	return item
}

func asList(value any) []map[string]any {
	raw, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		if mapped, ok := item.(map[string]any); ok {
			out = append(out, mapped)
		}
	}
	return out
}

func asStringSlice(value any) []string {
	raw, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		text := strings.TrimSpace(stringify(item))
		if text == "" {
			continue
		}
		out = append(out, text)
	}
	return out
}

func stringify(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return v
	case bool:
		if v {
			return "true"
		}
		return "false"
	case float64:
		if float64(int64(v)) == v {
			return fmt.Sprintf("%d", int64(v))
		}
		return fmt.Sprintf("%.3f", v)
	default:
		raw, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("%v", v)
		}
		return string(raw)
	}
}

func printJSON(value any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(value)
}

func envOrDefault(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "waf-cli: "+format+"\n", args...)
	os.Exit(1)
}
