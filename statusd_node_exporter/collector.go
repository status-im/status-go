package main

import (
	"fmt"
	"regexp"
)

type collector struct {
	c       *client
	filters []*regexp.Regexp
}

func compileFilters(rawFilters []string) []*regexp.Regexp {
	var filters []*regexp.Regexp
	for _, raw := range rawFilters {
		s := fmt.Sprintf("^%s", raw)
		filters = append(filters, regexp.MustCompile(s))
	}

	return filters
}

func newCollector(ipcPath string, rawFilters []string) (*collector, error) {
	c, err := newClient(ipcPath)
	if err != nil {
		return nil, err
	}

	filters := compileFilters(rawFilters)
	collector := &collector{
		c:       c,
		filters: filters,
	}

	return collector, nil
}

func (c *collector) collect() (string, error) {
	m, err := c.c.metrics()
	if err != nil {
		return "", err
	}

	all := transformMetrics(m)

	for k, _ := range all {
		if !c.match(k) {
			delete(all, k)
		}
	}

	return all.String(), nil
}

func (c *collector) match(key string) bool {
	if len(c.filters) == 0 {
		return true
	}

	for _, filter := range c.filters {
		if filter.MatchString(key) {
			return true
		}
	}

	return false
}
