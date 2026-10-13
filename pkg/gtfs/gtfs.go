package gtfs

import (
	"database/sql"
	"time"
)

func GetActiveServiceIDs(t *time.Time, db *sql.DB) ([]string, error) {
	weekday := t.Weekday()

	rows, err := db.Query(`select service_id from calendar
	WHERE (
		case $1
			when 0 then sunday
			when 1 then monday
			when 2 then tuesday
			when 3 then wednesday
			when 4 then thursday
			when 5 then friday
			when 6 then saturday
		end
	) AND $2 >= start_date AND $2 <= end_date AND NOT EXISTS (
		SELECT calendar_dates.service_id FROM calendar_dates WHERE calendar_dates.service_id = calendar.service_id AND calendar_dates.date = $2 AND calendar_dates.exception_type = 2
	)
	union
	select service_id from calendar_dates where date = $2 AND exception_type = 1;`, weekday, t.Format("20060102"))

	if err != nil {
		return nil, err
	}

	result := []string{}
	for rows.Next() {
		var serviceId string
		rows.Scan(&serviceId)
		result = append(result, serviceId)
	}

	return result, nil
}
