package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestEvaluationReportIsRedacted(t *testing.T) {
	r, err := evaluate("test")
	if err != nil {
		t.Fatal(err)
	}
	if r.Cases != 20 || r.TP+r.FN != 10 {
		t.Fatal("invalid ground-truth accounting")
	}
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range corpus() {
		for _, raw := range c.Expected {
			if strings.Contains(string(data), raw) {
				t.Fatal("raw fixture escaped into report")
			}
		}
	}
	if r.Precision < 0 || r.Precision > 1 || r.Recall < 0 || r.Recall > 1 {
		t.Fatal("invalid metric")
	}
}
