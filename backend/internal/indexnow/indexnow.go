// Package indexnow tells search engines that support IndexNow (Bing, Yandex,
// Seznam, Naver, Yep) which pages on a project's website were added or
// changed. Google does not use IndexNow.
package indexnow

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	// MaxURLs is the protocol's limit for one request.
	MaxURLs = 10_000
	// maxHosts caps the hosts one submission reaches, so one call cannot fan
	// out into hundreds of outbound requests.
	maxHosts = 3
	// maxSkippedShown keeps the answer small when most URLs are off-site.
	maxSkippedShown = 20
	// requestTimeout bounds each key-file check and each engine request.
	requestTimeout  = 8 * time.Second
	maxKeyFileBytes = 1 << 10
	userAgent       = "Uplumo-IndexNow/1.0 (+https://uplumo.com)"
)

// Engines are tried in order. Every engine shares an accepted submission
// with the others, so one accepted request reaches all of them. Bing goes
// first as the largest participant.
var Engines = []string{
	"https://www.bing.com/indexnow",
	"https://yandex.com/indexnow",
	"https://indexnow.yep.com/indexnow",
}

var (
	// ErrProjectNotFound means the project is not in the organization or is
	// archived.
	ErrProjectNotFound = errors.New("project not found")
	// ErrNoDomain means the project has no website yet.
	ErrNoDomain = errors.New("project has no website")
	// ErrNotSetUp means the project has no IndexNow key yet.
	ErrNotSetUp = errors.New("indexnow is not set up")
	// ErrNotIncluded means the workspace's plan does not include IndexNow.
	ErrNotIncluded = errors.New("plan does not include indexnow")
	// ErrRateLimited means the workspace used its hourly IndexNow requests.
	ErrRateLimited = errors.New("indexnow rate limited")
)

// Store keeps each project's key.
type Store interface {
	// Domain returns the active project's website, nil when it has none, or
	// ErrProjectNotFound.
	Domain(ctx context.Context, organizationID, projectID string) (*string, error)
	// Key returns the project's key, or "" when it has none.
	Key(ctx context.Context, projectID string) (string, error)
	// CreateKey stores key unless the project already has one, and returns
	// the key the project ends up with.
	CreateKey(ctx context.Context, projectID, key string) (string, error)
}

// Plans reports whether an organization's plan includes IndexNow.
type Plans interface {
	HasPaidPlan(ctx context.Context, organizationID string) (bool, error)
}

// Limiter counts IndexNow requests per workspace.
type Limiter interface {
	Allow(ctx context.Context, organizationID string) (bool, error)
}

// Service sets up keys, checks key files and submits URLs.
type Service struct {
	Logger *slog.Logger
	Store  Store
	// Plans is nil when billing is not configured: IndexNow is then open.
	Plans   Plans
	Limiter Limiter
	// Sites fetches key files from customers' websites. It must not follow
	// redirects and must refuse private addresses (audit.Guard).
	Sites *http.Client
	// Engines posts to the IndexNow endpoints.
	Engines   *http.Client
	Endpoints []string
}

// Setup is a project's IndexNow state.
type Setup struct {
	Domain string `json:"domain"`
	// Key is "" until someone sets IndexNow up.
	Key string `json:"key"`
}

// HostResult is what happened to the URLs on one host.
type HostResult struct {
	Host     string `json:"host"`
	URLCount int    `json:"urlCount"`
	// Status is "submitted" or "failed".
	Status  string `json:"status"`
	Message string `json:"message"`
}

// SubmitResult is what a submission did.
type SubmitResult struct {
	Hosts []HostResult `json:"hosts"`
	// Skipped lists up to 20 URLs that were not sent; SkippedCount is all.
	Skipped      []string `json:"skipped"`
	SkippedCount int      `json:"skippedCount"`
}

// KeyFile is whether one host serves the key file.
type KeyFile struct {
	Host string `json:"host"`
	// Status is found, missing, wrong_content, redirects, moved (redirects
	// to the same file on another host), unreachable or unknown (a bot wall,
	// server error or timeout: the file may be fine).
	Status string `json:"status"`
}

// Included reports whether the organization's plan includes IndexNow.
func (s *Service) Included(ctx context.Context, organizationID string) (bool, error) {
	if s.Plans == nil {
		return true, nil
	}
	included, err := s.Plans.HasPaidPlan(ctx, organizationID)
	if err != nil {
		return false, fmt.Errorf("check indexnow plan: %w", err)
	}
	return included, nil
}

// Get returns the project's website and key.
func (s *Service) Get(ctx context.Context, organizationID, projectID string) (Setup, error) {
	domain, err := s.domain(ctx, organizationID, projectID)
	if err != nil && !errors.Is(err, ErrNoDomain) {
		return Setup{}, err
	}
	key, keyErr := s.Store.Key(ctx, projectID)
	if keyErr != nil {
		return Setup{}, keyErr
	}
	return Setup{Domain: domain, Key: key}, nil
}

// SetUp creates the project's key the first time. Concurrent first uses
// end up with the same key, so the site never hosts a key that was replaced.
func (s *Service) SetUp(ctx context.Context, organizationID, projectID string) (Setup, error) {
	if err := s.require(ctx, organizationID); err != nil {
		return Setup{}, err
	}
	domain, err := s.domain(ctx, organizationID, projectID)
	if err != nil {
		return Setup{}, err
	}
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return Setup{}, fmt.Errorf("generate indexnow key: %w", err)
	}
	key, err := s.Store.CreateKey(ctx, projectID, hex.EncodeToString(raw))
	if err != nil {
		return Setup{}, err
	}
	return Setup{Domain: domain, Key: key}, nil
}

// CheckKeyFiles reports whether the bare domain and its www host serve the
// key file. It counts against the workspace's hourly limit because it
// fetches from the customer's website.
func (s *Service) CheckKeyFiles(ctx context.Context, organizationID, projectID string) ([]KeyFile, error) {
	setup, err := s.ready(ctx, organizationID, projectID)
	if err != nil {
		return nil, err
	}
	hosts := []string{setup.Domain}
	if !strings.HasPrefix(setup.Domain, "www.") {
		hosts = append(hosts, "www."+setup.Domain)
	}
	files := make([]KeyFile, len(hosts))
	for i, host := range hosts {
		files[i] = KeyFile{Host: host, Status: s.checkKeyFile(ctx, host, setup.Key)}
	}
	return files, nil
}

// Submit sends URLs on the project's website to IndexNow, one request per
// host. URLs off the website, and hosts past the third, are skipped rather
// than failing the call.
func (s *Service) Submit(ctx context.Context, organizationID, projectID string, urls []string) (SubmitResult, error) {
	setup, err := s.ready(ctx, organizationID, projectID)
	if err != nil {
		return SubmitResult{}, err
	}
	var order []string
	byHost := map[string][]string{}
	seen := map[string]bool{}
	result := SubmitResult{Hosts: []HostResult{}, Skipped: []string{}}
	skip := func(raw string) {
		result.SkippedCount++
		if len(result.Skipped) < maxSkippedShown {
			result.Skipped = append(result.Skipped, raw)
		}
	}
	for _, raw := range urls {
		host, normalized, ok := onSite(raw, setup.Domain)
		if !ok {
			skip(raw)
			continue
		}
		if seen[normalized] {
			continue
		}
		if _, known := byHost[host]; !known {
			if len(order) == maxHosts {
				skip(raw)
				continue
			}
			order = append(order, host)
		}
		seen[normalized] = true
		byHost[host] = append(byHost[host], normalized)
	}
	for _, host := range order {
		result.Hosts = append(result.Hosts, s.submitHost(ctx, host, setup.Key, byHost[host]))
	}
	return result, nil
}

// SubmitPublished sends one page that was just published to the customer's
// site. It does nothing, and answers "", when the plan does not include
// IndexNow or the project has not set it up: setting up the key is how a
// customer turns automatic submission on. It never fails the publish.
func (s *Service) SubmitPublished(ctx context.Context, organizationID, projectID, pageURL string) (status, message string) {
	result, err := s.Submit(ctx, organizationID, projectID, []string{pageURL})
	switch {
	case errors.Is(err, ErrNotIncluded), errors.Is(err, ErrNotSetUp), errors.Is(err, ErrNoDomain), errors.Is(err, ErrProjectNotFound):
		return "", ""
	case errors.Is(err, ErrRateLimited):
		return "failed", "Not sent to IndexNow: this workspace used its submissions for this hour. Submit the page from Settings later."
	case err != nil:
		s.logger().WarnContext(ctx, "indexnow submit published page", "project", projectID, "err", err)
		return "failed", "Not sent to IndexNow because of an error on our side. Submit the page from Settings."
	case len(result.Hosts) == 0:
		return "failed", "Not sent to IndexNow: the page is not on this project's website."
	default:
		return result.Hosts[0].Status, result.Hosts[0].Message
	}
}

// ready checks the plan, the website, the key and the hourly limit, in that
// order, so a free workspace is told about the plan, not the limit.
func (s *Service) ready(ctx context.Context, organizationID, projectID string) (Setup, error) {
	if err := s.require(ctx, organizationID); err != nil {
		return Setup{}, err
	}
	domain, err := s.domain(ctx, organizationID, projectID)
	if err != nil {
		return Setup{}, err
	}
	key, err := s.Store.Key(ctx, projectID)
	if err != nil {
		return Setup{}, err
	}
	if key == "" {
		return Setup{}, ErrNotSetUp
	}
	allowed, err := s.Limiter.Allow(ctx, organizationID)
	if err != nil {
		return Setup{}, fmt.Errorf("limit indexnow requests: %w", err)
	}
	if !allowed {
		return Setup{}, ErrRateLimited
	}
	return Setup{Domain: domain, Key: key}, nil
}

func (s *Service) require(ctx context.Context, organizationID string) error {
	included, err := s.Included(ctx, organizationID)
	if err != nil {
		return err
	}
	if !included {
		return ErrNotIncluded
	}
	return nil
}

func (s *Service) domain(ctx context.Context, organizationID, projectID string) (string, error) {
	domain, err := s.Store.Domain(ctx, organizationID, projectID)
	if err != nil {
		return "", err
	}
	if domain == nil || *domain == "" {
		return "", ErrNoDomain
	}
	return strings.ToLower(*domain), nil
}

// onSite returns the URL's host and the URL without its fragment when it is
// an http(s) URL on domain or one of its subdomains.
func onSite(raw, domain string) (host, normalized string, ok bool) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") || parsed.User != nil || parsed.Port() != "" {
		return "", "", false
	}
	host = strings.ToLower(parsed.Hostname())
	if host != domain && !strings.HasSuffix(host, "."+domain) {
		return "", "", false
	}
	parsed.Host = host
	parsed.Fragment, parsed.RawFragment = "", ""
	return host, parsed.String(), true
}

func (s *Service) checkKeyFile(ctx context.Context, host, key string) string {
	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()
	fileURL := "https://" + host + "/" + key + ".txt"
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, fileURL, nil)
	if err != nil {
		return "unreachable"
	}
	request.Header.Set("User-Agent", userAgent)
	response, err := s.Sites.Do(request)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return "unknown"
		}
		return "unreachable"
	}
	defer func() { _ = response.Body.Close() }()
	switch {
	case response.StatusCode == http.StatusNotFound || response.StatusCode == http.StatusGone:
		return "missing"
	case response.StatusCode >= 300 && response.StatusCode < 400:
		target, err := response.Request.URL.Parse(response.Header.Get("Location"))
		if err == nil && !strings.EqualFold(target.Hostname(), host) && target.Path == "/"+key+".txt" {
			return "moved"
		}
		return "redirects"
	case response.StatusCode != http.StatusOK:
		return "unknown"
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, maxKeyFileBytes))
	if err != nil {
		return "unknown"
	}
	if strings.TrimSpace(string(body)) != key {
		return "wrong_content"
	}
	return "found"
}

// keyFileProblem catches the setup mistakes engines would reject later. An
// inconclusive check returns "" and the submission goes ahead.
func keyFileProblem(status, host string) string {
	switch status {
	case "missing":
		return fmt.Sprintf("No key file at the root of https://%s. Upload it there, then try again.", host)
	case "wrong_content":
		return fmt.Sprintf("The key file on https://%s must contain only the key.", host)
	case "redirects", "moved":
		return fmt.Sprintf("The key file on https://%s redirects. Upload it to https://%s itself.", host, host)
	default:
		return ""
	}
}

func (s *Service) submitHost(ctx context.Context, host, key string, urls []string) HostResult {
	result := HostResult{Host: host, URLCount: len(urls), Status: "failed"}
	if problem := keyFileProblem(s.checkKeyFile(ctx, host, key), host); problem != "" {
		result.Message = problem
		return result
	}
	body, err := json.Marshal(map[string]any{"host": host, "key": key, "urlList": urls})
	if err != nil {
		result.Message = "Could not prepare the request."
		return result
	}
	status := 0
	for _, endpoint := range s.Endpoints {
		status = s.post(ctx, endpoint, host, body)
		// A rejected key or bad request would be rejected by every engine.
		if status != 0 && status != http.StatusTooManyRequests && status < 500 {
			break
		}
	}
	result.Status, result.Message = describe(status, len(urls), host)
	return result
}

// post answers the engine's status code, or 0 when it did not respond.
func (s *Service) post(ctx context.Context, endpoint, host string, body []byte) int {
	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return 0
	}
	request.Header.Set("Content-Type", "application/json; charset=utf-8")
	request.Header.Set("User-Agent", userAgent)
	response, err := s.Engines.Do(request)
	if err != nil {
		s.logger().WarnContext(ctx, "indexnow engine did not respond", "endpoint", endpoint, "host", host, "err", err)
		return 0
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode >= 300 {
		detail, _ := io.ReadAll(io.LimitReader(response.Body, 200))
		s.logger().WarnContext(ctx, "indexnow engine refused", "endpoint", endpoint, "host", host, "status", response.StatusCode, "detail", string(detail))
	}
	return response.StatusCode
}

func describe(status, count int, host string) (string, string) {
	urls := fmt.Sprintf("%d URL", count)
	if count != 1 {
		urls += "s"
	}
	switch status {
	case 0:
		return "failed", "IndexNow did not respond. Try again later."
	case http.StatusOK:
		return "submitted", "Submitted " + urls + "."
	case http.StatusAccepted:
		return "submitted", fmt.Sprintf("Submitted %s. Search engines will check the key file on https://%s first.", urls, host)
	case http.StatusForbidden:
		return "failed", fmt.Sprintf("The key was not accepted. Check that the key file on https://%s contains only the key.", host)
	case http.StatusUnprocessableEntity:
		return "failed", "IndexNow refused these URLs: they must belong to this host."
	case http.StatusTooManyRequests:
		return "failed", "Search engines are refusing submissions right now. Try again later, and only submit pages that changed."
	default:
		return "failed", fmt.Sprintf("IndexNow answered HTTP %d.", status)
	}
}

func (s *Service) logger() *slog.Logger {
	if s.Logger == nil {
		return slog.Default()
	}
	return s.Logger
}
