package main

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	serverURL          = "http://srv.msk01.gigacorp.local/_stats"
	pollInterval       = 2 * time.Second
	loadAvgThreshold   = 30.0
	memoryThreshold    = 80.0
	diskThreshold      = 90.0
	networkThreshold   = 90.0
	maxConsecutiveErrs = 3
)

type ServerStats struct {
	LoadAverage   float64
	TotalMemory   int64
	UsedMemory    int64
	TotalDisk     int64
	UsedDisk      int64
	NetBandwidth  int64
	NetUsage      int64
}

func fetchStats() (*ServerStats, error) {
	resp, err := http.Get(serverURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return parseStats(string(body))
}

func parseStats(data string) (*ServerStats, error) {
	data = strings.TrimSpace(data)
	parts := strings.Split(data, ",")

	if len(parts) != 7 {
		return nil, fmt.Errorf("invalid data format: expected 7 values, got %d", len(parts))
	}

	stats := &ServerStats{}
	var err error

	stats.LoadAverage, err = strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return nil, fmt.Errorf("invalid load average: %w", err)
	}

	stats.TotalMemory, err = strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid total memory: %w", err)
	}

	stats.UsedMemory, err = strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid used memory: %w", err)
	}

	stats.TotalDisk, err = strconv.ParseInt(parts[3], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid total disk: %w", err)
	}

	stats.UsedDisk, err = strconv.ParseInt(parts[4], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid used disk: %w", err)
	}

	stats.NetBandwidth, err = strconv.ParseInt(parts[5], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid network bandwidth: %w", err)
	}

	stats.NetUsage, err = strconv.ParseInt(parts[6], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid network usage: %w", err)
	}

	return stats, nil
}

func checkThresholds(stats *ServerStats) {
	if stats.LoadAverage > loadAvgThreshold {
		fmt.Printf("Load Average is too high: %.1f\n", stats.LoadAverage)
	}

	if stats.TotalMemory > 0 {
		memoryPercent := float64(stats.UsedMemory) / float64(stats.TotalMemory) * 100
		if memoryPercent > memoryThreshold {
			fmt.Printf("Memory usage too high: %.0f%%\n", memoryPercent)
		}
	}

	if stats.TotalDisk > 0 {
		diskPercent := float64(stats.UsedDisk) / float64(stats.TotalDisk) * 100
		if diskPercent > diskThreshold {
			freeDiskBytes := stats.TotalDisk - stats.UsedDisk
			freeDiskMB := freeDiskBytes / (1024 * 1024)
			fmt.Printf("Free disk space is too low: %d MB left\n", freeDiskMB)
		}
	}

	if stats.NetBandwidth > 0 {
		networkPercent := float64(stats.NetUsage) / float64(stats.NetBandwidth) * 100
		if networkPercent > networkThreshold {
			availableBandwidthBytes := stats.NetBandwidth - stats.NetUsage
			availableBandwidthMbits := float64(availableBandwidthBytes) * 8 / (1024 * 1024)
			fmt.Printf("Network bandwidth usage is too high: %.0f Mbit/s available\n", availableBandwidthMbits)
		}
	}
}

func main() {
	consecutiveErrors := 0

	for {
		stats, err := fetchStats()
		if err != nil {
			consecutiveErrors++
			if consecutiveErrors >= maxConsecutiveErrs {
				fmt.Println("Unable to fetch server statistic")
			}
		} else {
			consecutiveErrors = 0
			checkThresholds(stats)
		}

		time.Sleep(pollInterval)
	}
}
