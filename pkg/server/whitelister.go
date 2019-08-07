package server

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/coreos/go-iptables/iptables"
	raven "github.com/getsentry/raven-go"
	"github.com/gocardless/draupnir/pkg/store"
	"github.com/pkg/errors"
	"github.com/prometheus/common/log"
)

const ChainName = "DRAUPNIR-WHITELIST"

var (
	ruleDestPortRegexp  = regexp.MustCompile(`dpt:(\d+)`)
	ruleUserEmailRegexp = regexp.MustCompile(`/\* user: (.+) \*/$`)
)

type RuleEntry struct {
	IPNet     string
	Port      uint16
	UserEmail string
}

type reconcileRequest struct {
	// Source indicates where the request to reconcile came from, i.e. an
	// interval event or requested via an API interaction.
	Source string
	// RequestedTime is the time that the request for a reconciliaton was
	// created. This is used to track how long it took to fulfil the request.
	RequestedTime time.Time
}

type IPAddressWhitelister struct {
	logger                  log.Logger
	sentryClient            *raven.Client
	whitelistedAddressStore store.WhitelistedAddressStore
	reconcileTrigger        chan (reconcileRequest)
}

func NewIPAddressWhitelister(logger log.Logger, sentryClient *raven.Client, addressStore store.WhitelistedAddressStore) *IPAddressWhitelister {
	return &IPAddressWhitelister{
		logger:                  logger,
		sentryClient:            sentryClient,
		whitelistedAddressStore: addressStore,

		// Use a capacity of 100 requests. If this is ever reached, and the buffer
		// fills up, then we'll block API calls from completing.
		// But at this point something has likely gone very wrong and we should be
		// receiving sentries.
		reconcileTrigger: make(chan reconcileRequest, 100),
	}
}

func (iw *IPAddressWhitelister) Start(ctx context.Context, interval time.Duration) error {
	ipt, err := iptables.New()
	if err != nil {
		return errors.Wrap(err, "unable to setup iptables wrapper")
	}

	err = ensureChainPresent(ipt)
	if err != nil {
		return errors.Wrap(err, "failed to ensure whitelist chain is present")
	}

	// Trigger reconciles every interval, along with a single first-time reconcile
	go func() {
		for {
			iw.TriggerReconcile("timer")

			select {
			case <-ctx.Done():
				return
			case <-time.After(interval):
				// continue
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return nil
		case request := <-iw.reconcileTrigger:
			err = iw.reconcile(ipt, request)
			if err != nil {
				err = errors.Wrap(err, "failed to reconcile whitelist rules")
				// Given that this is an asynchronous process, and the worst case
				// scenario of a failure is that a legitimate user cannot access their
				// instance because it has not been whitelisted, record the error in
				// the form of logs and a sentry only, rather than bubbling up.
				iw.logger.Error(err)
				iw.sentryClient.CaptureError(err, map[string]string{})
			}
		}
	}
}

// TriggerReconcile allows external callers to request that a reconciliation
// occurs.
func (iw *IPAddressWhitelister) TriggerReconcile(source string) {
	iw.reconcileTrigger <- reconcileRequest{source, time.Now()}
}

func (iw *IPAddressWhitelister) reconcile(ipt *iptables.IPTables, request reconcileRequest) error {
	start := time.Now()
	logger := iw.logger.With("trigger_source", request.Source)

	logger.With("latency", time.Since(request.RequestedTime).Seconds()).
		Info("Starting whitelist reconciliation")

	// Build up a list of desired rules, as per the whitelisted_addresses table
	whitelist, err := iw.whitelistedAddressStore.List()
	if err != nil {
		return errors.Wrap(err, "failed to retrieve whitelisted IP addresses")
	}

	var desired []RuleEntry
	for _, a := range whitelist {
		desired = append(desired, RuleEntry{a.IPAddress, a.Instance.Port, a.Instance.UserEmail})
	}

	// Build up a list of existing rules
	existing, err := retrieveExistingRules(ipt)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve existing rules")
	}

	// Take the differences of the two sets, to determine what to add and remove
	// to the iptables chain.
	remove := sliceDifference(existing, desired)
	add := sliceDifference(desired, existing)

	for _, rule := range add {
		err = ipt.AppendUnique("filter", ChainName, buildRuleString(rule)...)
		if err != nil {
			return errors.Wrapf(err, "failed to add rule to chain: %v", rule)
		}
		logger.
			With("user_email", rule.UserEmail).
			With("ip_address", rule.IPNet).
			With("port", rule.Port).
			Info("Added rule to whitelist chain")
	}

	for _, rule := range remove {
		err = ipt.Delete("filter", ChainName, buildRuleString(rule)...)
		if err != nil {
			return errors.Wrapf(err, "failed to remove rule from chain: %v", rule)
		}
		logger.
			With("user_email", rule.UserEmail).
			With("ip_address", rule.IPNet).
			With("port", rule.Port).
			Info("Removed rule from whitelist chain")
	}

	if len(add) == 0 && len(remove) == 0 {
		logger.Info("No changes to whitelist chain required")
	}

	duration := time.Since(start)
	logger.With("duration", duration.Seconds()).Info("Finished whitelist reconciliation")

	return err
}

func ensureChainPresent(ipt *iptables.IPTables) error {
	chains, err := ipt.ListChains("filter")
	if err != nil {
		return errors.Wrap(err, "unable to determine iptables chains")
	}
	exists := false
	for _, c := range chains {
		if c == ChainName {
			exists = true
			break
		}
	}

	if !exists {
		err = ipt.NewChain("filter", ChainName)
	}

	return err
}

func retrieveExistingRules(ipt *iptables.IPTables) ([]RuleEntry, error) {
	var existing []RuleEntry

	// Retrieve all existing rules from iptables
	rules, err := ipt.StructuredStats("filter", ChainName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list rules in chain")
	}

	// Parse the existing rules into the common structure
	for _, s := range rules {
		// example of stat.Options string:
		// 'state NEW tcp dpt:1234 /* user: user@example.com */'
		opts := s.Options

		dstPortStr := ruleDestPortRegexp.FindStringSubmatch(opts)
		if dstPortStr == nil {
			return nil, fmt.Errorf("failed to find destination port in rule options: '%s'", opts)
		}
		dstPort, err := strconv.ParseUint(dstPortStr[1], 10, 16)
		if err != nil {
			return nil, errors.Wrap(err, "failed to find parse destination port")
		}

		userEmail := ruleUserEmailRegexp.FindStringSubmatch(opts)
		if userEmail == nil {
			err := fmt.Errorf("failed to find user email in rule options: '%s'", opts)
			return nil, err
		}

		existing = append(existing, RuleEntry{s.Source.IP.String(), uint16(dstPort), userEmail[1]})
	}

	return existing, nil
}

// Return the iptables rule specification for a given IP address
func buildRuleString(rule RuleEntry) []string {
	port := strconv.FormatUint(uint64(rule.Port), 10)
	comment := fmt.Sprintf("user: %s", rule.UserEmail)
	return []string{
		"-p", "tcp",
		"-m", "state", "--state", "NEW",
		"-s", rule.IPNet,
		"--dport", port,
		"-m", "comment", "--comment", comment,
		"-j", "ACCEPT",
	}
}

// Return the elements which are in slice a, but not slice b
func sliceDifference(a, b []RuleEntry) []RuleEntry {
	elements := []RuleEntry{}

	for _, aa := range a {
		found := false
		for _, bb := range b {
			if aa == bb {
				found = true
				break
			}
		}

		if !found {
			elements = append(elements, aa)
		}
	}

	return elements
}
