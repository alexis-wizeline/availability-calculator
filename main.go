package main

import (
	"fmt"
	"github.com/xuri/excelize/v2"
	"strconv"
	"strings"
	"time"
)

type routeStop struct {
	stopIndex                  int32
	VisitArrivalTimeStampSec   int64 `json:"VisitArrivalTimeStampSec"`
	VisitServiceDurationSec    int64 `json:"VisitServiceDurationSec"`
	RestBreakStartTimestampSec int64 `json:"RestBreakStartTimestampSec"`
	RestBreakDurationSec       int64 `json:"RestBreakDurationSec"`
}

func (r routeStop) arrivalTimestampSec() int64 {
	if r.RestBreakStartTimestampSec > 0 {
		return r.RestBreakStartTimestampSec
	}

	return r.VisitArrivalTimeStampSec
}

func (r routeStop) departureTimestampSec() int64 {
	if r.RestBreakStartTimestampSec > 0 {
		return r.RestBreakStartTimestampSec + r.RestBreakDurationSec
	}

	return r.VisitArrivalTimeStampSec + r.VisitServiceDurationSec
}

func (r routeStop) serviceDurationSec() int64 {
	if r.RestBreakDurationSec > 0 {
		return r.RestBreakDurationSec
	}
	return r.VisitServiceDurationSec
}

type attributeNamesMap map[string]bool
type shiftTeam struct {
	ID                 int64  `json:"id"`
	ShiftTeamID        int64  `json:"shiftTeamID"`
	StartTimestampSec  int64  `json:"startTimestampSec"`
	EndTimestampSec    int64  `json:"endtimestampSec"`
	AttributesString   string `json:"attributeNamesMap"`
	attributeNamesMap  attributeNamesMap
	RouteStops         []routeStop `json:"routeStops"`
	AllowedCapacitySec int64       `json:"allowedCapacitySec"`
	UsedCapacitySec    int64       `json:"usedCapacitySec"`
	IsActive           bool        `json:"isActive"`
}

func (s shiftTeam) hasAttributeNames(attributeNames []string) bool {
	for _, attributeName := range attributeNames {
		if !s.attributeNamesMap[attributeName] {
			return false
		}
	}

	return true
}

func (s shiftTeam) hasCapacity() bool {
	if s.AllowedCapacitySec == 0 {
		return true
	}
	return s.UsedCapacitySec < s.AllowedCapacitySec
}

func (s shiftTeam) isAvailable() bool {
	return s.hasCapacity() && s.IsActive
}

type routeStopGap struct {
	gapTimeSec       int64
	departureTimeSec int64
}

func (s shiftTeam) routeStopGapsFilteredByCurrentTime(currentTimestampSec int64) []routeStopGap {
	if len(s.RouteStops) == 0 {
		departureTimestampSec := s.StartTimestampSec
		if departureTimestampSec < currentTimestampSec {
			departureTimestampSec = currentTimestampSec
		}

		return []routeStopGap{
			{
				gapTimeSec:       s.EndTimestampSec - departureTimestampSec,
				departureTimeSec: departureTimestampSec,
			},
		}
	}

	var routeGaps []routeStopGap
	stop := s.RouteStops[0]
	initialGap := routeStopGap{
		gapTimeSec:       stop.arrivalTimestampSec() - s.StartTimestampSec,
		departureTimeSec: s.StartTimestampSec,
	}

	if initialGap.departureTimeSec >= currentTimestampSec {
		routeGaps = append(routeGaps, initialGap)
	}

	for i := 1; i < len(s.RouteStops); i++ {
		stop = s.RouteStops[i]
		prevStop := s.RouteStops[i-1]
		departureTimestampSec := prevStop.departureTimestampSec()
		if departureTimestampSec < currentTimestampSec {
			continue
		}

		routeGaps = append(routeGaps,
			routeStopGap{
				gapTimeSec:       stop.arrivalTimestampSec() - departureTimestampSec,
				departureTimeSec: departureTimestampSec,
			})
	}

	lastStop := s.RouteStops[len(s.RouteStops)-1]
	departureTimestampSec := lastStop.departureTimestampSec()
	if departureTimestampSec < currentTimestampSec && s.EndTimestampSec > currentTimestampSec {
		// if the last departure is in the past, but we still have time to see visits then we fix the departure
		// to be in the current time.
		departureTimestampSec = currentTimestampSec
	}
	routeGaps = append(routeGaps,
		routeStopGap{
			gapTimeSec:       s.EndTimestampSec - departureTimestampSec,
			departureTimeSec: departureTimestampSec,
		})

	return routeGaps
}

func attributesMapFromString(attributes string) attributeNamesMap {
	attributeMap := make(attributeNamesMap)
	for _, v := range strings.Split(attributes, ",") {
		attributeMap[v] = true
	}
	return attributeMap
}

type availabilityCalculatorVisit struct {
	ServiceDurationSec  int64 `json:"serviceDurationSec"`
	arrivalTimestampSec *int64
	shiftTeamID         *int64
	AttributeNames      []string `json:"attributeNames"`
}

type availabilityCalculator struct {
	ShiftTeams              []shiftTeam                    `json:"shiftTeams"`
	Visits                  []*availabilityCalculatorVisit `json:"Visits"`
	drivingTimeToVisitSec   int64
	drivingTimeFromVisitSec int64
	CurrentTimestampSec     int64 `json:"currentTimestampSec"`
	useCurrentTimestampSec  bool
}

func (a availabilityCalculator) shiftTeamsRouteStopGaps() map[int64][]routeStopGap {
	shiftTeamsRouteGapsMap := map[int64][]routeStopGap{}
	for _, s := range a.ShiftTeams {
		shiftTeamsRouteGapsMap[s.ID] = s.routeStopGapsFilteredByCurrentTime(a.timestampSec())
	}

	return shiftTeamsRouteGapsMap
}

func (a availabilityCalculator) timestampSec() int64 {

	return a.CurrentTimestampSec
}

func (a availabilityCalculator) visitWithCalculatedArrivals() []*availabilityCalculatorVisit {
	shiftTeamsAvailabilityGaps := a.shiftTeamsRouteStopGaps()
	for _, visit := range a.Visits {
		visitDuration := visit.ServiceDurationSec + a.drivingTimeToVisitSec + a.drivingTimeFromVisitSec
		for _, shift := range a.ShiftTeams {
			if visit.arrivalTimestampSec != nil && *visit.arrivalTimestampSec > 0 {
				break
			}

			shiftTeamAvailabilitiesGaps, ok := shiftTeamsAvailabilityGaps[shift.ID]
			if !ok || !shift.isAvailable() || !shift.hasAttributeNames(visit.AttributeNames) {
				continue
			}

			for _, gap := range shiftTeamAvailabilitiesGaps {
				arrivalTime := gap.departureTimeSec + a.drivingTimeToVisitSec
				// Check if the potential arrival is not in the past.
				if gap.gapTimeSec >= visitDuration && arrivalTime >= a.timestampSec() {
					visit.arrivalTimestampSec = &arrivalTime
					visit.shiftTeamID = &shift.ShiftTeamID
					break
				}
			}
		}
	}

	return a.Visits
}

func main() {

	f, err := excelize.OpenFile("calculator_34000066.xlsx")
	if err != nil {
		fmt.Println(err)
		return
	}

	rows, err := f.GetRows("Sheet1")
	if err != nil {
		fmt.Println(err)
		return
	}

	current := time.Date(2023, 12, 13, 1, 38, 0, 0, time.UTC)
	shiftAttributes := make(map[int64]map[string]bool)
	shiftStopsMap := make(map[int64][]routeStop)
	var shifts []shiftTeam
	for i, row := range rows {
		if i == 0 {
			continue
		}

		id, _ := strconv.Atoi(strings.ReplaceAll(row[0], " ", ""))
		shiftID, _ := strconv.Atoi(strings.ReplaceAll(row[1], " ", ""))
		start, _ := strconv.Atoi(strings.ReplaceAll(row[4], " ", ""))
		end, _ := strconv.Atoi(strings.ReplaceAll(row[5], " ", ""))

		shifts = append(shifts, shiftTeam{
			ID:                int64(id),
			ShiftTeamID:       int64(shiftID),
			StartTimestampSec: int64(start),
			EndTimestampSec:   int64(end),
			IsActive:          int64(end) >= current.Unix(),
		})
		shiftAttributes[int64(id)] = make(map[string]bool)
		shiftStopsMap[int64(id)] = make([]routeStop, 0)
	}

	rows, err = f.GetRows("Sheet4")
	if err != nil {
		fmt.Println(err)
		return
	}

	for i, row := range rows {
		if i == 0 {
			continue
		}

		id, _ := strconv.Atoi(strings.ReplaceAll(row[1], " ", ""))
		name := strings.ReplaceAll(row[0], " ", "")
		shiftID := int64(id)
		shiftAttributes[shiftID][name] = true
	}

	rows, err = f.GetRows("Sheet3")
	if err != nil {
		fmt.Println(err)
		return
	}

	shiftTeamRouteMap := make(map[int64]int64)
	for i, row := range rows {
		if i == 0 {
			continue
		}

		routeID, _ := strconv.Atoi(strings.ReplaceAll(row[0], " ", ""))
		ID, _ := strconv.Atoi(strings.ReplaceAll(row[2], " ", ""))

		shiftTeamRouteMap[int64(routeID)] = int64(ID)
	}

	rows, err = f.GetRows("Sheet2")
	if err != nil {
		fmt.Println(err)
		return
	}

	for i, row := range rows {
		if i == 0 {
			continue
		}

		routeID, _ := strconv.Atoi(strings.ReplaceAll(row[2], " ", ""))
		index, _ := strconv.Atoi(strings.ReplaceAll(row[3], " ", ""))

		restBreakStart, _ := strconv.Atoi(strings.ReplaceAll(row[7], " ", ""))
		restBreakDuration, _ := strconv.Atoi(strings.ReplaceAll(row[8], " ", ""))

		var visitArrival, visitDuration int
		if restBreakStart == 0 {
			visitArrival, _ = strconv.Atoi(strings.ReplaceAll(row[9], " ", ""))
			visitDuration, _ = strconv.Atoi(strings.ReplaceAll(row[12], " ", ""))
		}

		shiftID := shiftTeamRouteMap[int64(routeID)]

		shiftStopsMap[shiftID] = append(shiftStopsMap[shiftID], routeStop{
			stopIndex:                  int32(index),
			RestBreakStartTimestampSec: int64(restBreakStart),
			RestBreakDurationSec:       int64(restBreakDuration),
			VisitArrivalTimeStampSec:   int64(visitArrival),
			VisitServiceDurationSec:    int64(visitDuration),
		})
	}

	for i := range shifts {
		shifts[i].attributeNamesMap = shiftAttributes[shifts[i].ID]
		shifts[i].RouteStops = shiftStopsMap[shifts[i].ID]
	}

	c := availabilityCalculator{
		drivingTimeFromVisitSec: 1319,
		drivingTimeToVisitSec:   1319,
		Visits: []*availabilityCalculatorVisit{
			{
				ServiceDurationSec: 1446,
				AttributeNames: []string{
					"service_name:Acute",
					"presentation_modality:in_person",
				},
			},
		},
		ShiftTeams:             shifts,
		CurrentTimestampSec:    current.Unix(),
		useCurrentTimestampSec: true,
	}

	for _, v := range c.visitWithCalculatedArrivals() {
		fmt.Printf("%v \n", *v)
	}

	if err := f.Close(); err != nil {
		fmt.Println(err)
	}

}
