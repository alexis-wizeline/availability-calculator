use this to debug the availability calculator code, due to the recent move to atlas I choose to get the data from the queries I use and add the result into an excel file.
The data is being separated by sheets so each of the following queries represents a sheet on the Excel file

SQL queries
```sql
shift team snapshots
SELECT * FROM shift_team_snapshots where id IN (SELECT shift_team_snapshot_id FROM schedule_routes where schedule_id = schedule_id);

route stops
SELECT
    schedule_stops.*,
    shift_team_rest_break_requests.start_timestamp_sec AS break_request_start_timestamp_sec,
    shift_team_rest_break_requests.duration_sec AS break_request_duration_sec,
    schedule_visits.arrival_timestamp_sec,
    visit_phase_types.short_name AS visit_phase_short_name,
    visit_snapshots.care_request_id,
    visit_snapshots.service_duration_sec
FROM
    schedule_stops
    JOIN schedule_routes ON schedule_stops.schedule_route_id = schedule_routes.id
    LEFT JOIN schedule_rest_breaks ON schedule_stops.schedule_rest_break_id = schedule_rest_breaks.id
    LEFT JOIN shift_team_rest_break_requests ON schedule_rest_breaks.shift_team_break_request_id = shift_team_rest_break_requests.id
    LEFT JOIN schedule_visits ON schedule_stops.schedule_visit_id = schedule_visits.id
    LEFT JOIN visit_snapshots ON schedule_visits.visit_snapshot_id = visit_snapshots.id
    LEFT JOIN visit_phase_snapshots ON visit_snapshots.id = visit_phase_snapshots.visit_snapshot_id
    LEFT JOIN visit_phase_types ON visit_phase_snapshots.visit_phase_type_id = visit_phase_types.id
WHERE
    schedule_stops.schedule_id = schedule_id
ORDER BY
    schedule_stops.schedule_route_id,
    schedule_stops.route_index;

shift team routes 
SELECT * FROM schedule_routes where schedule_id = shift_team_id;


shift_team attributes
SELECT a.name, s.shift_team_snapshot_id FROM attributes a JOIN shift_team_attributes s ON a.id = s.attribute_id where s.shift_team_snapshot_id IN (SELECT shift_team_snapshot_id FROM schedule_routes where schedule_id = 33962304);
```

the queries are ordered as to how the code will map to the calculator stuff.

I added the headers of the query as the first row on the code to know which one is when checking the indexes

the sheet for attributes will look like this
```
name  |       shift_team_snapshot_id
radom string | radom id
radom string | radom id
radom string | radom id
```

as you are not going to evaluate all the visits you may need to get the service duration from our db and the drive times for the service region for the calculator settings on statsig
