package cardigann

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"
)

// Explicit -bench runs only; no timing thresholds in the regular test suite.
// Use a fixed iteration count so every observed operation contributes to p50/p95.
func BenchmarkCardigann(b *testing.B) {
	b.Run("CompilePlan", func(b *testing.B) {
		source := []byte(minimalDefinition)
		samples := make([]time.Duration, b.N)
		b.ReportAllocs()
		b.ResetTimer()
		for i := range b.N {
			start := time.Now()
			if _, err := Load(source); err != nil {
				b.Fatal(err)
			}
			samples[i] = time.Since(start)
		}
		b.StopTimer()
		reportLatency(b, samples)
	})
	for _, format := range []string{"HTML", "JSON", "HTMLFiltered"} {
		b.Run(format+"1000", func(b *testing.B) {
			plan, body := benchmarkSearchFixture(b, format)
			benchmarkSearch(b, plan, body, 1)
		})
	}
	b.Run("Concurrent8JSON1000", func(b *testing.B) {
		plan, body := benchmarkSearchFixture(b, "JSON")
		benchmarkSearch(b, plan, body, 8)
	})
	b.Run("CancelInFlight", func(b *testing.B) {
		plan, _ := benchmarkSearchFixture(b, "JSON")
		samples := make([]time.Duration, b.N)
		b.ReportAllocs()
		b.ResetTimer()
		for i := range b.N {
			ctx, cancel := context.WithCancel(context.Background())
			started := make(chan struct{})
			finished := make(chan error, 1)
			session := &fakeSession{respond: func(Request) (HTTPResponse, error) {
				close(started)
				<-ctx.Done()
				return HTTPResponse{}, ctx.Err()
			}}
			execution, err := plan.NewExecution(session, "", nil, benchmarkNow)
			if err != nil {
				cancel()
				b.Fatal(err)
			}
			go func() {
				_, err := execution.Search(ctx, Query{Keywords: "ABCD-123"})
				finished <- err
			}()
			<-started
			start := time.Now()
			cancel()
			err = <-finished
			samples[i] = time.Since(start)
			if err != context.Canceled || len(session.requests) != 1 {
				b.Fatal("cancellation lost its cause or replayed a request", err)
			}
		}
		b.StopTimer()
		reportLatency(b, samples)
		b.ReportMetric(1, "requests/op")
	})
}

var benchmarkNow = time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC)

func benchmarkSearchFixture(b *testing.B, format string) (*Plan, []byte) {
	b.Helper()
	definition := minimalDefinition
	var body strings.Builder
	if format == "JSON" {
		definition = strings.Replace(definition, "paths: [{path: /}]", "paths: [{path: /api, response: {type: json}}]", 1)
		definition = strings.Replace(definition, "rows: {selector: article}", "rows: {selector: data}", 1)
		definition = strings.Replace(definition, "title: {selector: h2}", "title: {selector: name}", 1)
		definition = strings.Replace(definition, "download: {selector: a, attribute: href}", "download: {selector: url}", 1)
		body.WriteString(`{"data":[`)
		for i := range 1000 {
			if i > 0 {
				body.WriteByte(',')
			}
			fmt.Fprintf(&body, `{"name":"ABCD-123 item %d","url":"/file/%d.torrent"}`, i, i)
		}
		body.WriteString(`]}`)
	} else {
		if format == "HTMLFiltered" {
			definition = strings.Replace(definition, "title: {selector: h2}", "title: {selector: h2, filters: [{name: regexp, args: '(ABCD-[0-9]+)'}]}", 1)
			definition += "    date: {text: '2026-11-03 00:00:00 +00:00', filters: [{name: dateparse, args: 'yyyy-MM-dd HH:mm:ss zzz'}]}\n"
		}
		for i := range 1000 {
			fmt.Fprintf(&body, `<article><h2>ABCD-123 item %d</h2><a href='/file/%d.torrent'>file</a></article>`, i, i)
		}
	}
	plan, err := Load([]byte(definition))
	if err != nil {
		b.Fatal(err)
	}
	return plan, []byte(body.String())
}

func benchmarkSearch(b *testing.B, plan *Plan, body []byte, concurrency int) {
	b.Helper()
	run := func() error {
		session := &fakeSession{respond: func(Request) (HTTPResponse, error) {
			return HTTPResponse{Status: 200, Body: body}, nil
		}}
		execution, err := plan.NewExecution(session, "", nil, benchmarkNow)
		if err != nil {
			return err
		}
		result, err := execution.Search(context.Background(), Query{Keywords: "ABCD-123", Limit: queryInteger(1000)})
		if err != nil {
			return err
		}
		if len(result.Results) != 1000 || len(result.Issues) != 0 || result.Truncated || len(session.requests) != 1 {
			return fmt.Errorf("benchmark lost rows or changed request count: rows=%d issues=%d truncated=%v requests=%d", len(result.Results), len(result.Issues), result.Truncated, len(session.requests))
		}
		return nil
	}
	samples := make([]time.Duration, b.N)
	b.ReportAllocs()
	b.SetBytes(int64(len(body) * concurrency))
	b.ResetTimer()
	for i := range b.N {
		start := time.Now()
		if concurrency == 1 {
			if err := run(); err != nil {
				b.Fatal(err)
			}
		} else {
			var workers sync.WaitGroup
			errors := make(chan error, concurrency)
			for range concurrency {
				workers.Add(1)
				go func() { defer workers.Done(); errors <- run() }()
			}
			workers.Wait()
			close(errors)
			for err := range errors {
				if err != nil {
					b.Fatal(err)
				}
			}
		}
		samples[i] = time.Since(start)
	}
	b.StopTimer()
	reportLatency(b, samples)
	b.ReportMetric(float64(concurrency), "requests/op")
}

func reportLatency(b *testing.B, samples []time.Duration) {
	b.Helper()
	sort.Slice(samples, func(i, j int) bool { return samples[i] < samples[j] })
	for _, percentile := range []int{50, 95} {
		index := (len(samples)*percentile+99)/100 - 1
		b.ReportMetric(float64(samples[index].Nanoseconds()), fmt.Sprintf("p%d-ns/op", percentile))
	}
}
