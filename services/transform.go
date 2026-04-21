package services

import (
	"fmt"
	"math"
	"sort"
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
	Identifier     string
	Datasource     string
	MaxAcPerHour   uint16
	HasTimings     bool
	Peak           int
	PeakLabel      string
	PeakRef        int
	PeakUtilPct    float64
	PeakCardClass  string
	TotalPeak      int
	TotalPeakLabel string
	TotalRef       int
	TotalUtilPct   float64
	TotalCardClass string
	AvgDwellMin    float64
}

type DepSeriesData struct {
	DepIdent string `json:"dep"`
	Color    string `json:"color"`
	Buckets  []int  `json:"buckets"`
}

type ArrivalAirportData struct {
	Identifier     string
	MaxSlots       uint16
	UtilPct        float64
	CardClass      string
	ArrWindowHours string
	Labels         []string
	Total          []int
	DepSeries      []DepSeriesData
}

func cardClass(utilPct float64) string {
	if utilPct > 100 {
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

	result := make([]SectorTimingData, 0, len(resp.Sectors))
	for _, s := range resp.Sectors {
		peakUtil := 0.0
		if s.PeakRef > 0 {
			peakUtil = math.Round(float64(s.Peak)/float64(s.PeakRef)*1000) / 10
		}
		totalUtil := 0.0
		if s.TotalRef > 0 {
			totalUtil = math.Round(float64(s.TotalPeak)/float64(s.TotalRef)*1000) / 10
		}
		peakClass, totalClass := cardClass(peakUtil), cardClass(totalUtil)
		if !s.HasTimings {
			peakClass, totalClass = "grey", "grey"
		}

		result = append(result, SectorTimingData{
			Identifier:     s.Identifier,
			Datasource:     s.Datasource,
			MaxAcPerHour:   s.MaxAcPerHour,
			HasTimings:     s.HasTimings,
			Peak:           s.Peak,
			PeakLabel:      s.PeakLabel,
			PeakRef:        s.PeakRef,
			PeakUtilPct:    peakUtil,
			PeakCardClass:  peakClass,
			TotalPeak:      s.TotalPeak,
			TotalPeakLabel: s.TotalPeakLabel,
			TotalRef:       s.TotalRef,
			TotalUtilPct:   totalUtil,
			TotalCardClass: totalClass,
			AvgDwellMin:    s.AvgDwellMin,
		})
	}

	// Sectors with timing data first, no-timing sectors at the bottom.
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].HasTimings != result[j].HasTimings {
			return result[i].HasTimings
		}
		return false
	})

	return result
}

func arrWindowHours(labels []string) string {
	if len(labels) < 2 {
		return ""
	}
	const layout = "15:04Z"
	first, err1 := time.Parse(layout, labels[0])
	last, err2 := time.Parse(layout, labels[len(labels)-1])
	if err1 != nil || err2 != nil {
		return ""
	}
	d := last.Sub(first)
	if d < 0 {
		d += 24 * time.Hour
	}
	if d <= 0 {
		return ""
	}
	hours := d.Hours()
	if hours == float64(int(hours)) {
		return fmt.Sprintf("%.0fh", hours)
	}
	return fmt.Sprintf("%.1fh", hours)
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

		totalArrivals := 0
		for _, v := range ap.Total {
			totalArrivals += v
		}
		utilPct := 0.0
		if ap.MaximumSlots > 0 && ap.MaximumSlots < 65535 {
			utilPct = math.Round(float64(totalArrivals)/float64(ap.MaximumSlots)*1000) / 10
		}

		result = append(result, ArrivalAirportData{
			Identifier:     ap.Identifier,
			MaxSlots:       ap.MaximumSlots,
			UtilPct:        utilPct,
			CardClass:      cardClass(utilPct),
			ArrWindowHours: arrWindowHours(ap.Labels),
			Labels:         ap.Labels,
			Total:          ap.Total,
			DepSeries:      depSeries,
		})
	}

	return result
}
