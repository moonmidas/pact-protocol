package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"pact/internal/pairing"
	"pact/internal/relay"
	"pact/internal/store"
)

func runInitiate(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) < 1 || strings.HasPrefix(args[0], "-") {
		fmt.Fprintln(stderr, "Usage: pact initiate <name> --as <profile> [--relay https://wepact.online]")
		return 2
	}
	name := args[0]
	fs := flag.NewFlagSet("initiate", flag.ContinueOnError)
	fs.SetOutput(stderr)
	root := fs.String("root", ".pact-local", "storage root")
	as := fs.String("as", "", "profile initiating pact")
	relayURL := fs.String("relay", "https://wepact.online", "relay base URL")
	if err := fs.Parse(args[1:]); err != nil {
		return 2
	}
	if *as == "" {
		fmt.Fprintln(stderr, "--as is required")
		return 2
	}
	paths := store.NewPaths(*root)
	if err := ensureProfile(paths, *as); err != nil {
		fmt.Fprintf(stderr, "initiate: %v\n", err)
		return 1
	}
	invitePath, invite, err := pairing.CreateInvite(paths, *as)
	if err != nil {
		fmt.Fprintf(stderr, "initiate: %v\n", err)
		return 1
	}
	data, err := json.Marshal(invite)
	if err != nil {
		fmt.Fprintf(stderr, "initiate: %v\n", err)
		return 1
	}
	resp, err := http.Post(strings.TrimRight(*relayURL, "/")+"/api/start", "application/json", strings.NewReader(string(data)))
	if err != nil {
		fmt.Fprintf(stderr, "initiate: %v\n", err)
		return 1
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		fmt.Fprintf(stderr, "initiate: relay returned %s\n", resp.Status)
		return 1
	}
	var created relay.CreateResponse
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		fmt.Fprintf(stderr, "initiate: %v\n", err)
		return 1
	}
	startURL, err := resolveStartURL(*relayURL, created.URL)
	if err != nil {
		fmt.Fprintf(stderr, "initiate: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "created Pact start link for %s: %s\n", name, startURL)
	fmt.Fprintf(stdout, "Send this to %s. Their agent should run: pact accept %s --as <their-name>\n", name, startURL)
	fmt.Fprintf(stdout, "local invite backup: %s\n", invitePath)
	return 0
}

func runAccept(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) < 1 || strings.HasPrefix(args[0], "-") {
		fmt.Fprintln(stderr, "Usage: pact accept <start-url-or-invite-file> --as <profile> [--handle <name>]")
		return 2
	}
	source := args[0]
	fs := flag.NewFlagSet("accept", flag.ContinueOnError)
	fs.SetOutput(stderr)
	root := fs.String("root", ".pact-local", "storage root")
	as := fs.String("as", "", "profile accepting pact")
	handle := fs.String("handle", "", "local handle for sender")
	if err := fs.Parse(args[1:]); err != nil {
		return 2
	}
	if *as == "" {
		fmt.Fprintln(stderr, "--as is required")
		return 2
	}
	paths := store.NewPaths(*root)
	if err := ensureProfile(paths, *as); err != nil {
		fmt.Fprintf(stderr, "accept: %v\n", err)
		return 1
	}
	invitePath := source
	cleanup := func() {}
	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		tmp, err := fetchStartInvite(source)
		if err != nil {
			fmt.Fprintf(stderr, "accept: %v\n", err)
			return 1
		}
		invitePath = tmp
		cleanup = func() { _ = os.Remove(tmp) }
	}
	defer cleanup()
	contact, err := pairing.AcceptInvite(paths, *as, *handle, invitePath)
	if err != nil {
		fmt.Fprintf(stderr, "accept: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "accepted Pact with %s as @%s\n", contact.DisplayName, contact.Handle)
	fmt.Fprintf(stdout, "You can now receive Pact requests from %s and use: pact runner tick --as %s\n", contact.Handle, *as)
	return 0
}

func fetchStartInvite(startURL string) (string, error) {
	apiURL, err := startAPIURL(startURL)
	if err != nil {
		return "", err
	}
	resp, err := http.Get(apiURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("relay returned %s", resp.Status)
	}
	tmp, err := os.CreateTemp("", "pact-start-*.json")
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(tmp, resp.Body); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
		return "", err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmp.Name())
		return "", err
	}
	return tmp.Name(), nil
}

func startAPIURL(link string) (string, error) {
	parsed, err := parseAbsoluteURL(link)
	if err != nil {
		return "", fmt.Errorf("expected an absolute Pact start URL like https://wepact.online/start/<id>")
	}
	if !strings.HasPrefix(parsed.Path, "/start/") {
		return "", fmt.Errorf("expected a Pact start path like /start/<id>")
	}
	parsed.Path = strings.Replace(parsed.Path, "/start/", "/api/start/", 1)
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String(), nil
}

func resolveStartURL(relayURL string, startURL string) (string, error) {
	parsedStart, err := parseURL(startURL)
	if err != nil {
		return "", fmt.Errorf("relay returned invalid start URL: %w", err)
	}
	if parsedStart.IsAbs() {
		return parsedStart.String(), nil
	}
	parsedRelay, err := parseAbsoluteURL(relayURL)
	if err != nil {
		return "", fmt.Errorf("relay returned relative start URL and --relay is not an absolute URL")
	}
	return parsedRelay.ResolveReference(parsedStart).String(), nil
}

func parseAbsoluteURL(value string) (*url.URL, error) {
	parsed, err := parseURL(value)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("invalid absolute URL")
	}
	return parsed, nil
}

func parseURL(value string) (*url.URL, error) {
	return url.Parse(value)
}
