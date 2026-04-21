package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"ctpcharts/config"
)

type Waypoint struct {
	ID                     int64   `json:"id"`
	Identifier             string  `json:"identifier"`
	Latitude               float64 `json:"latitude"`
	Longitude              float64 `json:"longitude"`
	MaximumAircraftPerHour uint16  `json:"maximumAircraftPerHour"`
	MaximumSlots           uint16  `json:"maximumSlots"`
}

type Airport struct {
	ID                     uint     `json:"id"`
	WaypointID             int64    `json:"waypointId"`
	Waypoint               Waypoint `json:"waypoint"`
	EventID                uint     `json:"eventId"`
	MaximumAircraftPerHour uint16   `json:"maximumAircraftPerHour"`
	MaximumSlots           uint16   `json:"maximumSlots"`
	SlotsAllocated         uint16   `json:"slotsAllocated"`
	NumberOfVotes          uint16   `json:"numberOfVotes"`
}

func (a Airport) Identifier() string {
	return a.Waypoint.Identifier
}

type RouteSegmentTag struct {
	ID             uint   `json:"id"`
	RouteSegmentID uint   `json:"routeSegmentId"`
	Tag            string `json:"tag"`
}

type Sector struct {
	ID                     uint   `json:"id"`
	Identifier             string `json:"identifier"`
	MaximumAircraftPerHour uint16 `json:"maximumAircraftPerHour"`
	EventID                *uint  `json:"eventId,omitempty"`
}

type Location struct {
	ID             uint     `json:"id"`
	RouteSegmentID uint     `json:"routeSegmentId"`
	WaypointID     int64    `json:"waypointId"`
	Waypoint       Waypoint `json:"waypoint"`
	SortOrder      uint     `json:"sortOrder"`
}

type RouteSegment struct {
	ID                          uint              `json:"id"`
	Identifier                  string            `json:"identifier"`
	MaximumAircraftPerHour      uint16            `json:"maximumAircraftPerHour"`
	RouteString                 string            `json:"routeString"`
	RouteSegmentGroup           string            `json:"routeSegmentGroup"`
	Color                       string            `json:"color"`
	Enabled                     bool              `json:"enabled"`
	Facilities                  string            `json:"facilities"`
	Tags                        []RouteSegmentTag `json:"tags"`
	ProvidedFacilityProgression []Sector          `json:"providedFacilityProgression"`
	Locations                   []Location        `json:"locations"`
	RouteRevision               uint              `json:"routeRevision"`
	EventID                     *uint             `json:"eventId,omitempty"`
}

type Slot struct {
	ID                   uint           `json:"id"`
	SlotRevisionID       uint           `json:"slotRevisionId"`
	DepartureTime        time.Time      `json:"departureTime"`
	ProjectedArrivalTime time.Time      `json:"projectedArrivalTime"`
	DepartureAirportID   uint           `json:"departureAirportId"`
	DepartureAirport     Airport        `json:"departureAirport"`
	ArrivalAirportID     uint           `json:"arrivalAirportId"`
	ArrivalAirport       Airport        `json:"arrivalAirport"`
	RouteSegments        []RouteSegment `json:"routeSegments"`
}

type ThroughputState struct {
	ID                       uint       `json:"id"`
	SlotRevisionID           uint       `json:"slotRevisionId"`
	ThroughputPointType      string     `json:"throughputPointType"`
	ThroughputPointID        int64      `json:"throughputPointId"`
	MaximumSlots             uint16     `json:"maximumSlots"`
	SlotsAllocated           uint16     `json:"slotsAllocated"`
	DepartureTimeWindowStart *time.Time `json:"departureTimeWindowStart,omitempty"`
}

type ThroughputSnapshot struct {
	ID                  uint   `json:"id"`
	SlotRevisionID      uint   `json:"slotRevisionId"`
	ThroughputPointType string `json:"throughputPointType"`
	ThroughputPointID   int64  `json:"throughputPointId"`
	MinuteOffset        int    `json:"minuteOffset"`
	SlotID              uint   `json:"slotId"`
}

type SlotRevision struct {
	ID                             uint                 `json:"id"`
	EventID                        uint                 `json:"eventId"`
	Number                         uint                 `json:"number"`
	SlotGenerationOutputCommentary string               `json:"slotGenerationOutputCommentary"`
	SimulationOutputCommentary     string               `json:"simulationOutputCommentary"`
	CreatedAt                      time.Time            `json:"createdAt"`
	Slots                          []Slot               `json:"slots"`
	ThroughputStates               []ThroughputState    `json:"throughputStates"`
	ThroughputSnapshots            []ThroughputSnapshot `json:"throughputSnapshots"`
}

type Event struct {
	ID                  uint           `json:"id"`
	Title               string         `json:"title"`
	RouteRevision       uint           `json:"routeRevision"`
	Date                time.Time      `json:"date"`
	DepartureTimeWindow string         `json:"departureTimeWindow"`
	Airports            []Airport      `json:"airports"`
	RouteSegments       []RouteSegment `json:"routeSegments"`
	Sectors             []Sector       `json:"sectors"`
}

type SimulatorData struct {
	Event    Event         `json:"event"`
	Revision *SlotRevision `json:"revision"`
}

type EventSummary struct {
	ID    uint      `json:"id"`
	Title string    `json:"title"`
	Date  time.Time `json:"date"`
}

type cachedData struct {
	data      *SimulatorData
	fetchedAt time.Time
}

var (
	cache    sync.Map
	cacheTTL = 30 * time.Second
)

func cacheKey(eventID uint, revision *uint) string {
	if revision != nil {
		return fmt.Sprintf("%d:%d", eventID, *revision)
	}
	return fmt.Sprintf("%d:latest", eventID)
}

func FetchEvents() ([]EventSummary, error) {
	req, err := http.NewRequest(http.MethodGet, config.C.CTPAPIURL+"/api/events", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-API-Key", config.C.CTPAPIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("events API returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var events []EventSummary
	if err := json.Unmarshal(body, &events); err != nil {
		return nil, err
	}

	return events, nil
}

func FetchSimulatorDataWithSlots(eventID uint) (*SimulatorData, error) {
	key := fmt.Sprintf("%d:latest-with-slots", eventID)
	if cached, ok := cache.Load(key); ok {
		entry := cached.(*cachedData)
		if time.Since(entry.fetchedAt) < cacheTTL {
			return entry.data, nil
		}
	}

	url := fmt.Sprintf("%s/api/events/%d/simulator-data/latest-with-slots", config.C.CTPAPIURL, eventID)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-API-Key", config.C.CTPAPIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("simulator-data/latest-with-slots API returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data SimulatorData
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	cache.Store(key, &cachedData{data: &data, fetchedAt: time.Now()})

	return &data, nil
}

// --- Charts API response types ---

type DepAirportSlot struct {
	Dep     string `json:"dep"`
	Arr     string `json:"arr"`
	DepTime string `json:"depTime"`
	ArrTime string `json:"arrTime"`
}

type DepAirportEntry struct {
	Identifier     string           `json:"identifier"`
	MaximumSlots   uint16           `json:"maximumSlots"`
	SlotsAllocated int              `json:"slotsAllocated"`
	Slots          []DepAirportSlot `json:"slots"`
}

type DepAirportsResponse struct {
	RevisionNumber uint              `json:"revisionNumber"`
	Airports       []DepAirportEntry `json:"airports"`
}

type SectorBucket struct {
	Label       string `json:"label"`
	PeakCount   int    `json:"peakCount"`
	UniqueCount int    `json:"uniqueCount"`
}

type SectorEntry struct {
	Identifier     string  `json:"identifier"`
	Datasource     string  `json:"datasource"`
	MaxAcPerHour   uint16  `json:"maxAcPerHour"`
	HasTimings     bool    `json:"hasTimings"`
	Peak           int     `json:"peak"`
	PeakLabel      string  `json:"peakLabel"`
	PeakRef        int     `json:"peakRef"`
	TotalPeak      int     `json:"totalPeak"`
	TotalPeakLabel string  `json:"totalPeakLabel"`
	TotalRef       int     `json:"totalRef"`
	AvgDwellMin    float64 `json:"avgDwellMinutes"`
}

type SectorsResponse struct {
	RevisionNumber      uint          `json:"revisionNumber"`
	EventDate           time.Time     `json:"eventDate"`
	DepartureTimeWindow string        `json:"departureTimeWindow"`
	Sectors             []SectorEntry `json:"sectors"`
}

type SectorBucketedResponse struct {
	Identifier              string         `json:"identifier"`
	Datasource              string         `json:"datasource"`
	MaxAcPerHour            uint16         `json:"maxAcPerHour"`
	HasTimings              bool           `json:"hasTimings"`
	TotalSlots              int            `json:"totalSlots"`
	AvgDwellMinutes         float64        `json:"avgDwellMinutes"`
	Peak                    int            `json:"peak"`
	PeakLabel               string         `json:"peakLabel"`
	PeakRef                 int            `json:"peakRef"`
	TotalPeak               int            `json:"totalPeak"`
	TotalPeakLabel          string         `json:"totalPeakLabel"`
	TotalRef                int            `json:"totalRef"`
	EstimatedMaxOccupancy    int            `json:"estimatedMaxOccupancy"`
	EstimatedTotalOccupancy int            `json:"estimatedTotalOccupancy"`
	Buckets                 []SectorBucket `json:"buckets"`
}

type ArrDepSeries struct {
	Dep     string `json:"dep"`
	Color   string `json:"color"`
	Buckets []int  `json:"buckets"`
}

type ArrAirportEntry struct {
	Identifier   string         `json:"identifier"`
	MaximumSlots uint16         `json:"maximumSlots"`
	Labels       []string       `json:"labels"`
	Total        []int          `json:"total"`
	DepSeries    []ArrDepSeries `json:"depSeries"`
}

type ArrAirportsResponse struct {
	RevisionNumber uint              `json:"revisionNumber"`
	EventDate      time.Time         `json:"eventDate"`
	Airports       []ArrAirportEntry `json:"airports"`
}

// --- Charts cache entries ---

type cachedDepAirports struct {
	data      *DepAirportsResponse
	fetchedAt time.Time
}

type cachedSectors struct {
	data      *SectorsResponse
	fetchedAt time.Time
}

type cachedArrAirports struct {
	data      *ArrAirportsResponse
	fetchedAt time.Time
}

func FetchDepAirports(eventID uint) (*DepAirportsResponse, error) {
	key := fmt.Sprintf("%d:dep-airports", eventID)
	if cached, ok := cache.Load(key); ok {
		entry := cached.(*cachedDepAirports)
		if time.Since(entry.fetchedAt) < cacheTTL {
			return entry.data, nil
		}
	}

	url := fmt.Sprintf("%s/api/events/%d/charts/departure-airports", config.C.CTPAPIURL, eventID)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-API-Key", config.C.CTPAPIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("charts/departure-airports API returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data DepAirportsResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	cache.Store(key, &cachedDepAirports{data: &data, fetchedAt: time.Now()})

	return &data, nil
}

func FetchSectors(eventID uint) (*SectorsResponse, error) {
	key := fmt.Sprintf("%d:sectors", eventID)
	if cached, ok := cache.Load(key); ok {
		entry := cached.(*cachedSectors)
		if time.Since(entry.fetchedAt) < cacheTTL {
			return entry.data, nil
		}
	}

	url := fmt.Sprintf("%s/api/events/%d/charts/sectors", config.C.CTPAPIURL, eventID)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-API-Key", config.C.CTPAPIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("charts/sectors API returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data SectorsResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	cache.Store(key, &cachedSectors{data: &data, fetchedAt: time.Now()})

	return &data, nil
}

func FetchArrAirports(eventID uint) (*ArrAirportsResponse, error) {
	key := fmt.Sprintf("%d:arr-airports", eventID)
	if cached, ok := cache.Load(key); ok {
		entry := cached.(*cachedArrAirports)
		if time.Since(entry.fetchedAt) < cacheTTL {
			return entry.data, nil
		}
	}

	url := fmt.Sprintf("%s/api/events/%d/charts/arrival-airports", config.C.CTPAPIURL, eventID)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-API-Key", config.C.CTPAPIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("charts/arrival-airports API returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data ArrAirportsResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	cache.Store(key, &cachedArrAirports{data: &data, fetchedAt: time.Now()})

	return &data, nil
}

type SectorFineResponse struct {
	Labels []string `json:"labels"`
	Data   []int    `json:"data"`
}

type ArrivalFineResponse struct {
	Labels    []string       `json:"labels"`
	Total     []int          `json:"total"`
	DepSeries []ArrDepSeries `json:"depSeries"`
}

func FetchSectorBucketed(eventID uint, identifier string) (*SectorBucketedResponse, error) {
	url := fmt.Sprintf("%s/api/events/%d/charts/sector/%s/bucketed", config.C.CTPAPIURL, eventID, identifier)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-API-Key", config.C.CTPAPIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("charts/sector/bucketed API returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data SectorBucketedResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	return &data, nil
}

func FetchSectorFine(eventID uint, identifier string) (*SectorFineResponse, error) {
	url := fmt.Sprintf("%s/api/events/%d/charts/sector/%s/fine", config.C.CTPAPIURL, eventID, identifier)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-API-Key", config.C.CTPAPIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("charts/sector/fine API returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data SectorFineResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	return &data, nil
}

func FetchArrivalFine(eventID uint, identifier string) (*ArrivalFineResponse, error) {
	url := fmt.Sprintf("%s/api/events/%d/charts/arrival/%s/fine", config.C.CTPAPIURL, eventID, identifier)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-API-Key", config.C.CTPAPIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("charts/arrival/fine API returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data ArrivalFineResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	return &data, nil
}

func FetchSimulatorData(eventID uint, revision *uint) (*SimulatorData, error) {
	key := cacheKey(eventID, revision)
	if cached, ok := cache.Load(key); ok {
		entry := cached.(*cachedData)
		if time.Since(entry.fetchedAt) < cacheTTL {
			return entry.data, nil
		}
	}

	url := fmt.Sprintf("%s/api/events/%d/simulator-data", config.C.CTPAPIURL, eventID)
	if revision != nil {
		url += fmt.Sprintf("?revision=%d", *revision)
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-API-Key", config.C.CTPAPIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("simulator-data API returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data SimulatorData
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	cache.Store(key, &cachedData{data: &data, fetchedAt: time.Now()})

	return &data, nil
}
