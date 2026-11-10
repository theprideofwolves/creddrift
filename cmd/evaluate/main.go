// Command evaluate measures a fixed synthetic corpus and scanner-only throughput.
// It never accepts a source path: all scans target temporary generated fixtures.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/theprideofwolves/creddrift/config"
	"github.com/theprideofwolves/creddrift/internal/store"
	"github.com/theprideofwolves/creddrift/scanner"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

type sample struct {
	ID, Text string
	Expected []string
}
type caseResult struct {
	ID                             string
	Expected, Detected, TP, FP, FN int
}
type report struct {
	Dataset, CorpusSHA256, Revision, Go, OS, Arch string
	Cases, TP, FP, FN                             int
	Precision, Recall                             float64
	CaseResults                                   []caseResult
	Performance                                   performance
}
type performance struct {
	Files, Bytes, MatchesPerTrial          int
	Milliseconds                           []float64
	MedianMilliseconds, MedianMiBPerSecond float64
	Method                                 string
}

func corpus() []sample {
	aws := "AKIA" + strings.Repeat("A", 16)
	gh := "ghp_" + strings.Repeat("b", 36)
	return []sample{
		{"aws-bare", aws, []string{aws}}, {"aws-assignment", "access_id=" + aws, []string{aws}},
		{"github-bare", gh, []string{gh}}, {"github-comment", "// token: " + gh, []string{gh}},
		{"generic-unquoted", "password=synthetic-test-only", []string{"synthetic-test-only"}},
		{"generic-single-quoted", "api_key='synthetic-key-only'", []string{"synthetic-key-only"}},
		{"generic-double-quoted", "password=\"synthetic-double-only\"", []string{"synthetic-double-only"}},
		{"generic-uppercase", "API_KEY : synthetic-upper-only", []string{"synthetic-upper-only"}},
		{"json-password", "\"password\": \"synthetic-json-only\"", []string{"synthetic-json-only"}},
		{"quoted-space", "password='synthetic two words'", []string{"synthetic two words"}},
		{"ordinary-text", "ordinary project documentation", nil},
		{"empty-assignment", "password=", nil}, {"empty-quotes", "password=\"\"", nil},
		{"aws-too-long", aws + "A", nil}, {"aws-too-short", aws[:len(aws)-1], nil},
		{"github-too-long", gh + "b", nil}, {"github-too-short", gh[:len(gh)-1], nil},
		{"nonsecret-placeholder", "password=example", nil},
		{"suffix-key", "not_password=synthetic-decoy-only", nil},
		{"unrelated-key", "username=synthetic-user-only", nil},
	}
}
func evaluate(revision string) (report, error) {
	r := report{Dataset: "synthetic-v1", Revision: revision, Go: runtime.Version(), OS: runtime.GOOS, Arch: runtime.GOARCH, CaseResults: []caseResult{}}
	root, err := os.MkdirTemp("", "creddrift-eval-")
	if err != nil {
		return r, err
	}
	defer os.RemoveAll(root)
	hash := sha256.New()
	samples := corpus()
	r.Cases = len(samples)
	// Each file is one labeled case, including deliberate known blind spots and decoys.
	for _, c := range samples {
		encoded, _ := json.Marshal(c)
		hash.Write(encoded)
		if err := os.WriteFile(filepath.Join(root, c.ID+".txt"), []byte(c.Text+"\n"), 0600); err != nil {
			return r, err
		}
	}
	r.CorpusSHA256 = hex.EncodeToString(hash.Sum(nil))
	matches, err := scanner.NewScanner(&config.Config{ScanTargets: []string{root}}).Scan()
	if err != nil {
		return r, err
	}
	found := map[string]map[string]bool{}
	for _, m := range matches {
		id := strings.TrimSuffix(filepath.Base(m.File), ".txt")
		if found[id] == nil {
			found[id] = map[string]bool{}
		}
		found[id][store.HashSecret(m.MatchTxt)] = true
	}
	for _, c := range samples {
		actual := found[c.ID]
		expected := map[string]bool{}
		for _, v := range c.Expected {
			expected[store.HashSecret(v)] = true
		}
		cr := caseResult{ID: c.ID, Expected: len(expected), Detected: len(actual)}
		for v := range actual {
			if expected[v] {
				cr.TP++
			} else {
				cr.FP++
			}
		}
		for v := range expected {
			if !actual[v] {
				cr.FN++
			}
		}
		r.TP += cr.TP
		r.FP += cr.FP
		r.FN += cr.FN
		r.CaseResults = append(r.CaseResults, cr)
	}
	if r.TP+r.FP > 0 {
		r.Precision = float64(r.TP) / float64(r.TP+r.FP)
	}
	if r.TP+r.FN > 0 {
		r.Recall = float64(r.TP) / float64(r.TP+r.FN)
	}
	return r, nil
}
func measure() (performance, error) {
	p := performance{Files: 1000, Bytes: 1000 * 65536, MatchesPerTrial: 100, Method: "scanner only; 1000 generated 64-KiB files; 100 candidate occurrences; one unreported warm-up then five sequential trials; fixture creation, SQLite, UI and process startup excluded; filesystem cache not flushed; not peak-memory measurement"}
	root, err := os.MkdirTemp("", "creddrift-perf-")
	if err != nil {
		return p, err
	}
	defer os.RemoveAll(root)
	for i := 0; i < p.Files; i++ {
		prefix := "ordinary generated source\n"
		if i%10 == 0 {
			prefix = "password=synthetic-perf-only\n"
		}
		data := prefix + strings.Repeat("x", 65536-len(prefix)-1) + "\n"
		if err := os.WriteFile(filepath.Join(root, fmt.Sprintf("%04d.txt", i)), []byte(data), 0600); err != nil {
			return p, err
		}
	}
	s := scanner.NewScanner(&config.Config{ScanTargets: []string{root}})
	for i := 0; i < 6; i++ {
		start := time.Now()
		matches, err := s.Scan()
		elapsed := float64(time.Since(start).Microseconds()) / 1000
		if err != nil {
			return p, err
		}
		if len(matches) != p.MatchesPerTrial || len(s.ScannedFiles) != p.Files {
			return p, fmt.Errorf("performance fixture coverage mismatch")
		}
		if i > 0 {
			p.Milliseconds = append(p.Milliseconds, elapsed)
		}
	}
	sorted := append([]float64{}, p.Milliseconds...)
	sort.Float64s(sorted)
	p.MedianMilliseconds = sorted[2]
	p.MedianMiBPerSecond = float64(p.Bytes) / (1 << 20) / (p.MedianMilliseconds / 1000)
	return p, nil
}
func main() {
	revision := flag.String("revision", "unrecorded", "tested source revision")
	out := flag.String("out", "", "JSON output file")
	flag.Parse()
	r, err := evaluate(*revision)
	if err == nil {
		r.Performance, err = measure()
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "Evaluation failed")
		os.Exit(1)
	}
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		panic(err)
	}
	data = append(data, '\n')
	if *out != "" {
		err = os.WriteFile(*out, data, 0600)
	} else {
		_, err = os.Stdout.Write(data)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "Cannot write evaluation report")
		os.Exit(1)
	}
}
