package services

import (
	"math"
	"time"
)

type AirportSlotEntry struct {
	DepIdent string `json:"dep"`
	ArrIdent string `json:"arr"`
	DepTime  string `json:"depTime"`
	ArrTime  string `json:"arrTime"`
}

type DepartureAirportData struct {
	Name           string
	SlotsAllocated int
	MaxSlots       uint16
	MaxAcPerHour   uint16
	UtilPct        float64
	CardClass      string
	Slots          []AirportSlotEntry
}

type SectorTimingData struct {
	Identifier   string
	MaxAcPerHour uint16
	UtilPct      float64
	CardClass    string
	HasTimings   bool
	Peak         int
	PeakLabel    string
	TimeSeries   []int
	TimeLabels   []string
}

type DepSeriesData struct {
	DepIdent string `json:"dep"`
	Color    string `json:"color"`
	Buckets  []int  `json:"buckets"`
}

type ArrivalAirportData struct {
	Identifier string
	MaxSlots   uint16
	Labels     []string
	Total      []int
	DepSeries  []DepSeriesData
}

func cardClass(utilPct float64) string {
	if utilPct >= 100 {
		return "red"
	}
	if utilPct >= 90 {
		return "orange"
	}
	return "green"
}

func BuildDepartureAirports(resp *DepAirportsResponse) []DepartureAirportData {
	if resp == nil {
		return nil
	}

	var result []DepartureAirportData
	for _, ap := range resp.Airports {
		utilPct := 0.0
		if ap.MaximumSlots > 0 && ap.MaximumSlots < 65535 {
			utilPct = math.Round(float64(ap.SlotsAllocated)/float64(ap.MaximumSlots)*1000) / 10
		}

		slots := make([]AirportSlotEntry, 0, len(ap.Slots))
		for _, s := range ap.Slots {
			slots = append(slots, AirportSlotEntry{
				DepIdent: s.Dep,
				ArrIdent: s.Arr,
				DepTime:  s.DepTime,
				ArrTime:  s.ArrTime,
			})
		}

		result = append(result, DepartureAirportData{
			Name:           ap.Identifier,
			SlotsAllocated: ap.SlotsAllocated,
			MaxSlots:       ap.MaximumSlots,
			UtilPct:        utilPct,
			CardClass:      cardClass(utilPct),
			Slots:          slots,
		})
	}

	return result
}

func BuildSectors(resp *SectorsResponse) []SectorTimingData {
	if resp == nil {
		return nil
	}

	depHours := 3.0
	if d, err := time.ParseDuration(resp.DepartureTimeWindow); err == nil && d > 0 {
		depHours = d.Hours()
	}

	var result []SectorTimingData
	for _, s := range resp.Sectors {
		utilPct := 0.0
		if s.MaxAcPerHour > 0 && s.MaxAcPerHour < 65535 && depHours > 0 {
			utilPct = math.Round(float64(s.TotalSlots)/(float64(s.MaxAcPerHour)*depHours)*1000) / 10
		}

		series := make([]int, 0, len(s.Buckets))
		labels := make([]string, 0, len(s.Buckets))
		peak, peakIdx := 0, 0
		for i, b := range s.Buckets {
			series = append(series, b.Count)
			labels = append(labels, b.Label)
			if b.Count > peak {
				peak = b.Count
				peakIdx = i
			}
		}

		peakLabel := ""
		if len(labels) > 0 && peakIdx < len(labels) {
			peakLabel = labels[peakIdx]
		}

		result = append(result, SectorTimingData{
			Identifier:   s.Identifier,
			MaxAcPerHour: s.MaxAcPerHour,
			UtilPct:      utilPct,
			CardClass:    cardClass(utilPct),
			HasTimings:   s.HasTimings,
			Peak:         peak,
			PeakLabel:    peakLabel,
			TimeSeries:   series,
			TimeLabels:   labels,
		})
	}

	return result
}

func BuildArrivalAirports(resp *ArrAirportsResponse) []ArrivalAirportData {
	if resp == nil {
		return nil
	}

	var result []ArrivalAirportData
	for _, ap := range resp.Airports {
		depSeries := make([]DepSeriesData, 0, len(ap.DepSeries))
		for _, ds := range ap.DepSeries {
			depSeries = append(depSeries, DepSeriesData{
				DepIdent: ds.Dep,
				Color:    ds.Color,
				Buckets:  ds.Buckets,
			})
		}

		result = append(result, ArrivalAirportData{
			Identifier: ap.Identifier,
			MaxSlots:   ap.MaximumSlots,
			Labels:     ap.Labels,
			Total:      ap.Total,
			DepSeries:  depSeries,
		})
	}

	return result
}
