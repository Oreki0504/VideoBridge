package bridge

import (
	"golang.org/x/net/publicsuffix"
	"net"
	"net/url"
	"regexp"
	"strings"
)

var dnsName = regexp.MustCompile(`^(?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z](?:[a-z0-9-]{0,61}[a-z0-9])?$`)

func siteRoot(host string) (string, error) { return publicsuffix.EffectiveTLDPlusOne(host) }
func publicWebHost(host string) bool {
	if len(host) > 253 || net.ParseIP(host) != nil || !dnsName.MatchString(host) {
		return false
	}
	for _, suffix := range []string{"localhost", "local", "internal", "lan", "home", "onion"} {
		if host == suffix || strings.HasSuffix(host, "."+suffix) {
			return false
		}
	}
	_, err := siteRoot(host)
	return err == nil
}
func cookieRoots(raw string) []string {
	u, err := url.Parse(raw)
	if err != nil {
		return nil
	}
	root, err := siteRoot(u.Hostname())
	if err != nil {
		return nil
	}
	if root == "x.com" || root == "twitter.com" {
		return []string{"x.com", "twitter.com"}
	}
	return []string{root}
}
func allowedCookieDomain(domain string, roots []string) bool {
	bare := strings.TrimPrefix(domain, ".")
	if !dnsName.MatchString(bare) {
		return false
	}
	for _, root := range roots {
		if bare == root || strings.HasSuffix(bare, "."+root) {
			return true
		}
	}
	return false
}
