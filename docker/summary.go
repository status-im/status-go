package scale

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
)

const separator = "|"

type WhispReport struct {
	NewEnvelopes float64
	OldEnvelopes float64
	Ingress      float64
	Egress       float64
}

type Summary []WhispReport

func (s Summary) MeanOldPerNew() float64 {
	var sum float64
	for _, r := range s {
		sum += r.OldEnvelopes / r.NewEnvelopes
	}
	return sum / float64(len(s))
}

func (s Summary) Print(w io.Writer) {
	var (
		ingress   float64
		egress    float64
		newEnv    float64
		oldEnv    float64
		oldPerNew = s.MeanOldPerNew()
	)
	tab := tabwriter.NewWriter(w, 0, 8, 1, '\t', 0)
	fmt.Fprintln(w, "=== SUMMARY")
	fmt.Fprintln(tab, strings.Join([]string{"HEADERS", "ingress", "egress", "dups", "new", "dups/new"}, "\t|"))
	for i, r := range s {
		ingress += r.Ingress
		egress += r.Egress
		newEnv += r.NewEnvelopes
		oldEnv += r.OldEnvelopes
		fmt.Fprintln(tab, strings.Join([]string{
			fmt.Sprintf("%d", i),
			fmt.Sprintf("%f mb", r.Ingress/1024/1024),
			fmt.Sprintf("%f mb", r.Egress/1024/1024),
			fmt.Sprintf("%d", int64(r.OldEnvelopes)),
			fmt.Sprintf("%d", int64(r.NewEnvelopes)),
			fmt.Sprintf("%f", r.OldEnvelopes/r.NewEnvelopes),
		}, "\t|"))
	}
	ingress = ingress / 1024 / 1024
	egress = egress / 1024 / 1024
	fmt.Fprintln(tab, strings.Join([]string{
		"TOTAL",
		fmt.Sprintf("%f mb", ingress),
		fmt.Sprintf("%f mb", egress),
		fmt.Sprintf("%d", int64(oldEnv)),
		fmt.Sprintf("%d", int64(newEnv)),
		fmt.Sprintf("%f", oldPerNew),
	}, "\t|"))
	tab.Flush()
}
