package handlers

import (
	"encoding/json"
	"html/template"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog/log"

	"ctpcharts/config"
	"ctpcharts/services"
)

func UtilClass(pct float64) string {
	if pct >= 100 {
		return "util-red"
	}
	if pct >= 80 {
		return "util-orange"
	}
	return "util-green"
}

func ToJSON(v interface{}) template.JS {
	b, _ := json.Marshal(v)
	return template.JS(b)
}

func AirportUtilClass(pct float64) string {
	if pct >= 100 {
		return "util-red"
	}
	if pct >= 90 {
		return "util-orange"
	}
	return "util-green"
}

var TemplateFuncs = template.FuncMap{
	"utilClass":        UtilClass,
	"airportUtilClass": AirportUtilClass,
	"toJSON":           ToJSON,
}

type baseData struct {
	PageTitle        string
	ActivePage       string
	CurrentPath      string
	BasePath         string
	EventQuery       string
	Events           []services.EventSummary
	Revisions        []uint
	SelectedEventID  uint
	SelectedRevision uint
	CurrentRevision  uint
	HasRevision      bool
	EventTitle       string
	CacheBust        string
}

func newBaseData(c fiber.Ctx, activePage string) baseData {
	events, _ := services.FetchEvents()

	eventIDStr := c.Query("event")
	var eventID uint
	if v, err := strconv.ParseUint(eventIDStr, 10, 64); err == nil {
		eventID = uint(v)
	}

	// Auto-select latest event when none specified
	if eventID == 0 && len(events) > 0 {
		latest := events[len(events)-1]
		eventID = latest.ID
		eventIDStr = strconv.FormatUint(uint64(eventID), 10)
	}

	eventQuery := ""
	if eventID > 0 {
		eventQuery = "?event=" + eventIDStr
	}

	revisionStr := c.Query("revision")
	var revision uint
	if v, err := strconv.ParseUint(revisionStr, 10, 64); err == nil {
		revision = uint(v)
		if eventID > 0 {
			eventQuery += "&revision=" + revisionStr
		}
	}

	title := ""
	for _, e := range events {
		if e.ID == eventID {
			title = e.Title
			break
		}
	}

	return baseData{
		PageTitle:        activePage,
		ActivePage:       activePage,
		CurrentPath:      c.Path(),
		BasePath:         config.C.BasePath,
		EventQuery:       eventQuery,
		Events:           events,
		SelectedEventID:  eventID,
		SelectedRevision: revision,
		EventTitle:       title,
		CacheBust:        strconv.FormatInt(time.Now().Unix(), 10),
	}
}

func fetchData(c fiber.Ctx, eventID uint) (*services.SimulatorData, error) {
	if eventID == 0 {
		return nil, nil
	}

	if revStr := c.Query("revision"); revStr != "" {
		if v, err := strconv.ParseUint(revStr, 10, 64); err == nil {
			rev := uint(v)
			data, err := services.FetchSimulatorData(eventID, &rev)
			if err != nil {
				log.Error().Err(err).Uint("eventId", eventID).Msg("failed to fetch simulator data")
				return nil, err
			}
			return data, nil
		}
	}

	// No explicit revision — use latest revision that has slots
	data, err := services.FetchSimulatorDataWithSlots(eventID)
	if err != nil {
		log.Error().Err(err).Uint("eventId", eventID).Msg("failed to fetch simulator data")
		return nil, err
	}
	return data, nil
}

// ── Departure airports page ───────────────────────────────────────────────────

type departureAirportView struct {
	Name           string
	SlotsAllocated int
	MaxSlots       uint16
	MaxAcPerHour   uint16
	UtilPct        float64
	CardClass      string
	SlotsJSON      template.JS
}

func DepartureAirportsPage(c fiber.Ctx) error {
	bd := newBaseData(c, "airports-departure")

	resp, err := services.FetchDepAirports(bd.SelectedEventID)
	if err != nil {
		log.Error().Err(err).Uint("eventId", bd.SelectedEventID).Msg("failed to fetch departure airports")
	}

	hasRevision := resp != nil && resp.RevisionNumber > 0
	var airports []departureAirportView
	if hasRevision {
		raw := services.BuildDepartureAirports(resp)
		for _, ap := range raw {
			slotsJSON, _ := json.Marshal(ap.Slots)
			airports = append(airports, departureAirportView{
				Name:           ap.Name,
				SlotsAllocated: ap.SlotsAllocated,
				MaxSlots:       ap.MaxSlots,
				MaxAcPerHour:   ap.MaxAcPerHour,
				UtilPct:        ap.UtilPct,
				CardClass:      ap.CardClass,
				SlotsJSON:      template.JS(slotsJSON),
			})
		}
	}

	return c.Render("airports-departure", fiber.Map{
		"PageTitle":        "Departure Airports",
		"ActivePage":       bd.ActivePage,
		"CurrentPath":      bd.CurrentPath,
		"BasePath":         bd.BasePath,
		"EventQuery":       bd.EventQuery,
		"Events":           bd.Events,
		"SelectedEventID":  bd.SelectedEventID,
		"SelectedRevision": bd.SelectedRevision,
		"CurrentRevision":  resp.RevisionNumber,
		"HasRevision":      hasRevision,
		"EventTitle":       bd.EventTitle,
		"CacheBust":        bd.CacheBust,
		"Airports":         airports,
	}, "layout")
}

// ── Sectors page ─────────────────────────────────────────────────────────────

type sectorView struct {
	Identifier     string
	MaxAcPerHour   uint16
	RefLine        int // reference line value for the chart (0 = no line)
	UtilPct        float64
	CardClass      string
	HasTimings     bool
	Peak           int
	PeakLabel      string
	TimeSeriesJSON template.JS
	TimeLabelsJSON template.JS
}

func buildSectorViews(raw []services.SectorTimingData, peak bool) []sectorView {
	views := make([]sectorView, 0, len(raw))
	for _, s := range raw {
		var series []int
		var refLine int
		var utilPct float64
		var cardClass string
		var pk int
		var pkLabel string
		if peak {
			series = s.PeakTimeSeries
			refLine = s.EstimatedMaxOccupancy
			utilPct = s.PeakUtilPct
			cardClass = s.PeakCardClass
			pk = s.Peak
			pkLabel = s.PeakLabel
		} else {
			series = s.TotalTimeSeries
			refLine = s.EstimatedTotalOccupancy
			utilPct = s.TotalUtilPct
			cardClass = s.TotalCardClass
			pk = s.TotalPeak
			pkLabel = s.TotalPeakLabel
		}
		tsJSON, _ := json.Marshal(series)
		tlJSON, _ := json.Marshal(s.TimeLabels)
		views = append(views, sectorView{
			Identifier:     s.Identifier,
			MaxAcPerHour:   s.MaxAcPerHour,
			RefLine:        refLine,
			UtilPct:        utilPct,
			CardClass:      cardClass,
			HasTimings:     s.HasTimings,
			Peak:           pk,
			PeakLabel:      pkLabel,
			TimeSeriesJSON: template.JS(tsJSON),
			TimeLabelsJSON: template.JS(tlJSON),
		})
	}
	return views
}

func renderSectorsPage(c fiber.Ctx, peakMode bool) error {
	activePage := "sectors-peak"
	pageTitle := "Sectors – Max Occupancy"
	if !peakMode {
		activePage = "sectors-total"
		pageTitle = "Sectors – Total Occupancy"
	}
	bd := newBaseData(c, activePage)

	resp, err := services.FetchSectors(bd.SelectedEventID)
	if err != nil {
		log.Error().Err(err).Uint("eventId", bd.SelectedEventID).Msg("failed to fetch sectors")
	}

	hasRevision := resp != nil && resp.RevisionNumber > 0
	var sectors []sectorView
	if hasRevision {
		sectors = buildSectorViews(services.BuildSectors(resp), peakMode)
	}

	return c.Render("sectors", fiber.Map{
		"PageTitle":        pageTitle,
		"ActivePage":       bd.ActivePage,
		"CurrentPath":      bd.CurrentPath,
		"BasePath":         bd.BasePath,
		"EventQuery":       bd.EventQuery,
		"Events":           bd.Events,
		"SelectedEventID":  bd.SelectedEventID,
		"SelectedRevision": bd.SelectedRevision,
		"CurrentRevision":  resp.RevisionNumber,
		"HasRevision":      hasRevision,
		"EventTitle":       bd.EventTitle,
		"CacheBust":        bd.CacheBust,
		"Sectors":          sectors,
		"BodyClass":        "sectors-page",
	}, "layout")
}

func SectorsMaxOccPage(c fiber.Ctx) error   { return renderSectorsPage(c, true) }
func SectorsTotalOccPage(c fiber.Ctx) error { return renderSectorsPage(c, false) }

// ── Arrival airports page ─────────────────────────────────────────────────────

type arrivalAirportView struct {
	Identifier     string
	MaxSlots       uint16
	UtilPct        float64
	CardClass      string
	ArrWindowHours string
	LabelsJSON     template.JS
	TotalJSON      template.JS
	DepSeriesJSON  template.JS
}

func ArrivalAirportsPage(c fiber.Ctx) error {
	bd := newBaseData(c, "airports-arrival")

	resp, err := services.FetchArrAirports(bd.SelectedEventID)
	if err != nil {
		log.Error().Err(err).Uint("eventId", bd.SelectedEventID).Msg("failed to fetch arrival airports")
	}

	hasRevision := resp != nil && resp.RevisionNumber > 0
	var airports []arrivalAirportView
	if hasRevision {
		raw := services.BuildArrivalAirports(resp)
		for _, ap := range raw {
			labelsJSON, _ := json.Marshal(ap.Labels)
			totalJSON, _ := json.Marshal(ap.Total)
			depJSON, _ := json.Marshal(ap.DepSeries)
			airports = append(airports, arrivalAirportView{
				Identifier:     ap.Identifier,
				MaxSlots:       ap.MaxSlots,
				UtilPct:        ap.UtilPct,
				CardClass:      ap.CardClass,
				ArrWindowHours: ap.ArrWindowHours,
				LabelsJSON:     template.JS(labelsJSON),
				TotalJSON:      template.JS(totalJSON),
				DepSeriesJSON:  template.JS(depJSON),
			})
		}
	}

	return c.Render("airports-arrival", fiber.Map{
		"PageTitle":        "Arrival Airports",
		"ActivePage":       bd.ActivePage,
		"CurrentPath":      bd.CurrentPath,
		"BasePath":         bd.BasePath,
		"EventQuery":       bd.EventQuery,
		"Events":           bd.Events,
		"SelectedEventID":  bd.SelectedEventID,
		"SelectedRevision": bd.SelectedRevision,
		"CurrentRevision":  resp.RevisionNumber,
		"HasRevision":      hasRevision,
		"EventTitle":       bd.EventTitle,
		"CacheBust":        bd.CacheBust,
		"Airports":         airports,
	}, "layout")
}

func ProxySectorFine(c fiber.Ctx) error {
	eventID := c.Query("event")
	identifier := c.Params("identifier")
	if eventID == "" || identifier == "" {
		return fiber.NewError(fiber.StatusBadRequest, "event and identifier are required")
	}

	id, err := strconv.ParseUint(eventID, 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid event id")
	}

	data, err := services.FetchSectorFine(uint(id), identifier)
	if err != nil {
		log.Error().Err(err).Msg("failed to fetch sector fine data")
		return fiber.NewError(fiber.StatusInternalServerError, "failed to fetch data")
	}

	return c.JSON(data)
}

func ProxyArrivalFine(c fiber.Ctx) error {
	eventID := c.Query("event")
	identifier := c.Params("identifier")
	if eventID == "" || identifier == "" {
		return fiber.NewError(fiber.StatusBadRequest, "event and identifier are required")
	}

	id, err := strconv.ParseUint(eventID, 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid event id")
	}

	data, err := services.FetchArrivalFine(uint(id), identifier)
	if err != nil {
		log.Error().Err(err).Msg("failed to fetch arrival fine data")
		return fiber.NewError(fiber.StatusInternalServerError, "failed to fetch data")
	}

	return c.JSON(data)
}
