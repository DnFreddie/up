package pkg

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

const tmplRequiredProp = "required property: '%s'"

// Probe is an experiment to measure the connectivity of a network using
// different protocols and public servers.
type Probe struct {
	// Protocols to use.
	Protocols []*Protocol
	// Number of iterations. Zero means infinite.
	Count uint
	// Delay between requests.
	Delay time.Duration
	// Time to wait for a response.
	// TODO: Support specific timeouts per protocol.
	Timeout time.Duration
	// For debugging purposes.
	Logger *slog.Logger
	// Optional channel to send back partial results.
	ReportCh chan *Report
}

// Ensures the probe setup is correct.
func (p Probe) validate() error {
	if p.Protocols == nil || len(p.Protocols) == 0 {
		return fmt.Errorf(tmplRequiredProp, "Protocols")
	}
	if p.Timeout == 0 {
		return fmt.Errorf(tmplRequiredProp, "Timeout")
	}
	if p.Logger == nil {
		return fmt.Errorf(tmplRequiredProp, "Logger")
	}
	if p.ReportCh == nil {
		return fmt.Errorf(tmplRequiredProp, "ReportCh")
	}
	return nil
}

// Run the connection attempts to the public servers.
//
// The context can be cancelled between different protocol attempts or count
// iterations.
// Returns an error if the setup is invalid.
func (p Probe) Run(ctx context.Context) error {
	p.Logger.Debug("Starting", "setup", p)
	err := p.validate()
	if err != nil {
		return fmt.Errorf("invalid setup: %w", err)
	}
	count := uint(0)
	for {
		select {
		case <-ctx.Done():
			p.Logger.Debug(
				"Context cancelled between iterations",
				"count", count,
			)
			return nil
		default:
			p.Logger.Debug("New iteration", "count", count)
			for _, proto := range p.Protocols {
				select {
				case <-ctx.Done():
					p.Logger.Debug(
						"Context cancelled between protocols",
						"count", count, "protocol", proto.ID,
					)
					return nil
				default:
					start := time.Now()
					rhost := proto.RHost()
					p.Logger.Debug(
						"New protocol",
						"count", count, "protocol", proto.ID, "rhost", rhost,
					)
					extra, err := proto.Request(rhost, p.Timeout)
					report := Report{
						ProtocolID: proto.ID,
						RHost:      rhost,
						Time:       time.Since(start),
						Error:      err,
						Extra:      extra,
					}
					p.Logger.Debug(
						"Sending report back",
						"count", count, "report", report,
					)
					p.ReportCh <- &report
					time.Sleep(p.Delay)
				}
			}
			count++
			p.Logger.Debug(
				"End of iteration",
				"count", count, "p.Count", p.Count,
			)
			if count == p.Count {
				p.Logger.Debug("Count limit reached", "count", count)
				return nil
			}
		}
	}
}
