package dns

import (
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/miekg/dns"
)

const (
	resolvConfFilePath = "/etc/resolv.conf"
	defaultDNSPort     = "53"
	dotChar            = "."
)

var errResourceRecordNotFound = errors.New("resource record not found")

func LookupAddressFromSRVRecord(name, server string) (string, error) {
	var serverAddr string
	switch len(server) == 0 {
	case true:
		conf, err := dns.ClientConfigFromFile(resolvConfFilePath)
		if err != nil {
			return "", fmt.Errorf("failed to create dns client config: %w", err)
		}

		serverAddr = net.JoinHostPort(conf.Servers[0], defaultDNSPort)
	case false:
		serverAddr = server
	}

	var (
		msg    dns.Msg
		client dns.Client
	)
	msg.RecursionDesired = true
	msg.SetQuestion(dns.Fqdn(name), dns.TypeSRV)

	reply, _, err := client.Exchange(&msg, serverAddr)
	if err != nil {
		return "", fmt.Errorf("failed to lookup dns record: %w", err)
	}

	for _, rr := range reply.Answer {
		switch v := rr.(type) {
		case *dns.SRV:
			return strings.TrimRight(v.Target, dotChar), nil
		default:
			continue
		}
	}

	return "", errResourceRecordNotFound
}
