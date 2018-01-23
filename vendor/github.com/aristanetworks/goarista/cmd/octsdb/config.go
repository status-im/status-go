// Copyright (C) 2016  Arista Networks, Inc.
// Use of this source code is governed by the Apache License 2.0
// that can be found in the COPYING file.

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"regexp"
	"strings"
)

// Config is the representation of octsdb's JSON config file.
type Config struct {
	// Prefixes to subscribe to.
	Subscriptions []string

	// MetricPrefix, if set, is used to prefix all the metric names.
	MetricPrefix string

	// Metrics to collect and how to munge them.
	Metrics map[string]*Metric
}

// A Metric to collect and how to massage it into an OpenTSDB put.
type Metric struct {
	// Path is a regexp to match on the Update's full path.
	// The regexp must be a prefix match.
	// The regexp can define named capture groups to use as tags.
	Path string

	// Path compiled as a regexp.
	re *regexp.Regexp

	// Additional tags to add to this metric.
	Tags map[string]string
}

func loadConfig(path string) (*Config, error) {
	cfg, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("Failed to load config: %v", err)
	}
	config := new(Config)
	err = json.Unmarshal(cfg, config)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse config: %v", err)
	}
	for _, metric := range config.Metrics {
		metric.re = regexp.MustCompile(metric.Path)
	}
	return config, nil
}

// Match applies this config to the given OpenConfig path.
// If the path doesn't match anything in the config, an empty string
// is returned as the metric name.
func (c *Config) Match(path string) (metricName string, tags map[string]string) {
	tags = make(map[string]string)

	for _, metric := range c.Metrics {
		found := metric.re.FindStringSubmatch(path)
		if found == nil {
			continue
		}
		for i, name := range metric.re.SubexpNames() {
			if i == 0 {
				continue
			} else if name == "" {
				if metricName != "" {
					metricName += "/"
				}
				metricName += found[i]
			} else {
				tags[name] = found[i]
			}
		}
		for tag, value := range metric.Tags {
			tags[tag] = value
		}
		break
	}
	if metricName != "" {
		metricName = strings.ToLower(strings.Replace(metricName, "/", ".", -1))
		if c.MetricPrefix != "" {
			metricName = c.MetricPrefix + "." + metricName
		}
	}
	return
}
