package utils

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	testName  string
	iteration int
	phase     string
	mu        sync.Mutex
)

func RecordMemoryMetricsCSV(testName string, iter int, phase string, heapKB, rssKB uint64) error {
	mu.Lock()
	defer mu.Unlock()

	f, err := os.OpenFile("memory_metrics.csv", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	stat, err := f.Stat()
	if err != nil {
		return err
	}
	if stat.Size() == 0 {
		header := []string{"TestName", "Iteration", "Phase", "HeapAlloc(KB)", "RSS(KB)", "Timestamp"}
		if err := w.Write(header); err != nil {
			return err
		}
	}

	row := []string{
		testName,
		strconv.Itoa(iter),
		phase,
		strconv.FormatUint(heapKB, 10),
		strconv.FormatUint(rssKB, 10),
		time.Now().Format(time.RFC3339),
	}

	return w.Write(row)
}

func GetRSSKB() (uint64, error) {
	f, err := os.Open("/proc/self/statm")
	if err != nil {
		return 0, err
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		return 0, err
	}
	fields := strings.Fields(string(data))
	if len(fields) < 2 {
		return 0, fmt.Errorf("unexpected /proc/self/statm format")
	}
	rssPages, err := strconv.ParseUint(fields[1], 10, 64)
	if err != nil {
		return 0, err
	}
	pageSize := os.Getpagesize()
	return (rssPages * uint64(pageSize)) / 1024, nil
}

