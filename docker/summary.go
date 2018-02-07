package scale

import (
	"fmt"
	"strings"
)

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

func (s Summary) String() string {
	var (
		ingress   float64
		egress    float64
		newEnv    float64
		oldEnv    float64
		oldPerNew = s.MeanOldPerNew()
	)
	for _, r := range s {
		ingress += r.Ingress
		egress += r.Egress
		newEnv += r.NewEnvelopes
		oldEnv += r.OldEnvelopes
	}
	ingress = ingress / 1024 / 1024
	egress = egress / 1024 / 1024
	return strings.Join([]string{
		"=== SUMMARY",
		fmt.Sprintf("ingress      = %f mb", ingress),
		fmt.Sprintf("egress       = %f mb", egress),
		fmt.Sprintf("duplicates   = %f", oldEnv),
		fmt.Sprintf("new          = %f", newEnv),
		fmt.Sprintf("dups per new = %f", oldPerNew),
	}, "\n    ")
}
