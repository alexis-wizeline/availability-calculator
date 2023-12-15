use this to debug the availability calculator code, due to the recent move to atlas I choose to get the data from the queries I use and add the result into an Excel file.
The data is being separated by sheets so each of the following queries represents a sheet on the Excel file

```
Sheet 1 - shift team snapshots for reference schedule
Sheet 2 - route stops for the reference schedule
Sheet 3 - reference schedule routes
Sheet 3 - shift team attributes
```

the queries (check slack market place channel for these queries) are ordered as to how the code will map to the calculator stuff.

I added the headers of the query as the first row on the code to know which one is when checking the indexes

the sheet for attributes will look like this
```
name  |       shift_team_snapshot_id
radom string | radom id
radom string | radom id
radom string | radom id
```

as you are not going to evaluate all the visits you may need to get the service duration from our db and the drive times for the service region for the calculator settings on statsig
