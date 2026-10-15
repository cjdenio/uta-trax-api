package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/cjdenio/uta-trax-api/pkg/gtfs"
	pb "github.com/cjdenio/uta-trax-api/proto"
	"github.com/jmoiron/sqlx"
	"google.golang.org/protobuf/proto"

	_ "github.com/mattn/go-sqlite3"
)

var scheduleDb *sqlx.DB

var headsignMappings map[string]string = map[string]string{
	"to medical":     "To University Medical",
	"to airport":     "To Airport",
	"to draper":      "To Draper",
	"to salt lake":   "To Salt Lake Central",
	"to meadowbrook": "To Meadowbrook",
	"to ballpark":    "To Ballpark",
}

func distance(lat1 float64, lng1 float64, lat2 float64, lng2 float64) float64 {
	radlat1 := float64(math.Pi * lat1 / 180)
	radlat2 := float64(math.Pi * lat2 / 180)

	theta := float64(lng1 - lng2)
	radtheta := float64(math.Pi * theta / 180)

	dist := math.Sin(radlat1)*math.Sin(radlat2) + math.Cos(radlat1)*math.Cos(radlat2)*math.Cos(radtheta)
	if dist > 1 {
		dist = 1
	}

	dist = math.Acos(dist)
	dist = dist * 180 / math.Pi
	dist = dist * 60 * 1.1515

	return dist
}

func getStationForVehicle(vehicle *pb.VehiclePosition) *pb.VehicleFeed_Station {
	rows, err := scheduleDb.Query(`SELECT stop_lat, stop_lon, stops.stop_id, stop_name FROM stop_times INNER JOIN stops ON stops.stop_id = stop_times.stop_id WHERE stop_times.trip_id = ? AND stop_times.pickup_type != 1 AND stop_times.drop_off_type != 1;`, vehicle.Trip.TripId)
	if err != nil {
		fmt.Println(err)
		return nil
	}

	nearestStop := new(pb.VehicleFeed_Station)
	nearestStopDistance := math.Inf(1)

	for rows.Next() {
		var stop_lat float32
		var stop_lon float32
		var stop_id string
		var stop_name string

		rows.Scan(&stop_lat, &stop_lon, &stop_id, &stop_name)

		distance := distance(float64(stop_lat), float64(stop_lon), float64(*vehicle.Position.Latitude), float64(*vehicle.Position.Longitude))
		if distance < nearestStopDistance {
			nearestStop.Id = stop_id
			nearestStop.Name = stop_name
			nearestStop.Lat = stop_lat
			nearestStop.Lon = stop_lon

			nearestStopDistance = distance
		}
	}

	// sometimes trains aren't where they should be, ignore nearest stop if more than 2 miles away
	if nearestStopDistance > 2 {
		return nil
	}

	return nearestStop
}

func fetchVehicleFeed() ([]byte, error) {
	if os.Getenv("PRETEND_FEED") == "true" {
		return os.ReadFile("pretend-feed.bin")
	} else {
		resp, err := http.Get("https://apps.rideuta.com/tms/gtfs/Vehicle")
		if err != nil {
			return nil, err
		}

		bytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		return bytes, nil
	}
}

func getVehicles() ([]*pb.VehiclePosition, *pb.FeedHeader, error) {
	bytes, err := fetchVehicleFeed()
	if err != nil {
		return nil, nil, err
	}

	feed := pb.FeedMessage{}
	err = proto.Unmarshal(bytes, &feed)
	if err != nil {
		return nil, nil, err
	}

	log.Printf("Found %d vehicles...", len(feed.Entity))

	vehicles := make([]*pb.VehiclePosition, 0, len(feed.Entity))
	for _, v := range feed.Entity {
		if v.Vehicle != nil {
			vehicles = append(vehicles, v.Vehicle)
		}
	}

	return vehicles, feed.Header, nil
}

func feedifyVehicles(vehicles []*pb.VehiclePosition, header *pb.FeedHeader, routeTypes map[int]bool) pb.VehicleFeed {
	vehicle_feed := make([]*pb.VehicleFeed_Vehicle, 0, len(vehicles))

	for _, vehicle := range vehicles {
		rows, err := scheduleDb.Query(`
			SELECT
				trips.route_id,
				trips.shape_id,
				trips.trip_headsign,
				routes.route_type,
				routes.route_color,
				routes.route_short_name,
				routes.route_long_name
			FROM trips
			INNER JOIN routes ON routes.route_id = trips.route_id
			WHERE trips.trip_id = ?
			LIMIT 1;
		`, *vehicle.Trip.TripId)
		if err != nil {
			fmt.Println(err)
			continue
		}

		var route_id string
		var shape_id string
		var trip_headsign sql.NullString
		var route_type int32
		var route_color sql.NullString
		var route_short_name sql.NullString
		var route_long_name sql.NullString

		if !rows.Next() {
			continue
		}

		err = rows.Scan(&route_id, &shape_id, &trip_headsign, &route_type, &route_color, &route_short_name, &route_long_name)
		if err != nil {
			fmt.Println(err)
			continue
		}

		// sometimes headsigns are weird. this lets me manually change them
		if trip_headsign.Valid {
			if mapping, ok := headsignMappings[strings.ToLower(trip_headsign.String)]; ok {
				trip_headsign.String = mapping
			}
		}

		rows.Close()

		if routeTypes != nil && !routeTypes[int(route_type)] {
			continue
		}

		vehicle_feed = append(vehicle_feed, &pb.VehicleFeed_Vehicle{
			Lat:     *vehicle.Position.Latitude,
			Lon:     *vehicle.Position.Longitude,
			Bearing: *vehicle.Position.Bearing,
			// Line:           trip.Line,
			Id: *vehicle.Vehicle.Id,
			// Direction:      trip.Direction,
			NearestStation: getStationForVehicle(vehicle),
			Headsign:       trip_headsign.String,
			Route: &pb.VehicleFeed_Route{
				Id:        route_id,
				Type:      pb.VehicleFeed_Route_RouteType(route_type + 1), // lol
				Color:     route_color.String,
				ShortName: route_short_name.String,
				LongName:  route_long_name.String,
			},
			TripId:  *vehicle.Trip.TripId,
			ShapeId: shape_id,
		})
	}

	return pb.VehicleFeed{
		Vehicles: vehicle_feed,
		Info: &pb.VehicleFeed_FeedInfo{
			LastUpdate: *header.Timestamp,
		},
	}
}

func vehicleHandler(mode string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vehicles, header, err := getVehicles()
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		routeTypes := r.URL.Query().Get("route_types")
		var routeTypeFilter map[int]bool

		if routeTypes != "" {
			routeTypeFilter = make(map[int]bool)
			for t := range strings.SplitSeq(routeTypes, ",") {
				routeTypeNumber, err := strconv.Atoi(t)
				if err == nil {
					routeTypeFilter[routeTypeNumber] = true
				}
			}
		}

		if mode == "json" || r.Header.Get("Accept") == "application/json" {
			feed := feedifyVehicles(vehicles, header, routeTypeFilter)
			response, err := json.Marshal(&feed)
			if err != nil {
				log.Println(err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.Header().Add("Content-Type", "application/json")
			w.Header().Add("Content-Length", strconv.Itoa(len(response)))
			w.Write(response)
		} else {
			// send a protobuf

			feed := feedifyVehicles(vehicles, header, routeTypeFilter)
			b, err := proto.Marshal(&feed)
			if err != nil {
				log.Println(err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.Header().Add("Content-Type", "application/protobuf")
			w.Header().Add("Content-Length", strconv.Itoa(len(b)))
			w.Write(b)
		}
	}
}

func tripHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := scheduleDb.Query(`SELECT shapes.shape_pt_lat, shapes.shape_pt_lon FROM shapes INNER JOIN trips ON trips.shape_id = shapes.shape_id WHERE trips.trip_id = ? ORDER BY shapes.shape_pt_sequence ASC`, r.PathValue("trip"))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var points [][]float64

	for rows.Next() {
		var lat float64
		var lon float64
		err = rows.Scan(&lat, &lon)
		if err != nil {
			break
		}

		points = append(points, []float64{lat, lon})
	}

	if len(points) == 0 {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.Header().Add("Content-Type", "application/json")

	response, _ := json.Marshal(map[string]any{"shape": points})
	w.Write(response)
}

func shapeHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := scheduleDb.Query(`SELECT shapes.shape_pt_lat, shapes.shape_pt_lon FROM shapes WHERE shapes.shape_id = ? ORDER BY shapes.shape_pt_sequence ASC`, r.PathValue("shape"))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var points [][]float64

	for rows.Next() {
		var lat float64
		var lon float64
		err = rows.Scan(&lat, &lon)
		if err != nil {
			break
		}

		points = append(points, []float64{lat, lon})
	}

	if len(points) == 0 {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.Header().Add("Content-Type", "application/json")

	response, _ := json.Marshal(map[string]any{"shape": points})
	w.Write(response)
}

type Route struct {
	ID        string `json:"routeId"`
	ShortName string `json:"shortName"`
	LongName  string `json:"longName"`
}

type Departure struct {
	Route             Route     `json:"route"`
	DepartureTime     time.Time `json:"departureTime"`
	Headsign          string    `json:"headsign"`
	PredictedPlatform int       `json:"predictedPlatform,omitempty"`
	TripId            string    `json:"tripId"`
}

type ApiDepartures struct {
	Departures []Departure `json:"departures"`
	To         time.Time   `json:"to"`
}

type DbDeparture struct {
	RouteId        string `db:"route_id"`
	RouteShortName string `db:"route_short_name"`
	RouteLongName  string `db:"route_long_name"`
	DepartureTime  string `db:"departure_time"`
	TripHeadsign   string `db:"trip_headsign"`
	TripId         string `db:"trip_id"`
	DirectionId    int    `db:"direction_id"`
	StopId         string `db:"stop_id"`
	PassingTrain   bool   `db:"passing_train"`
}

var predictionData map[string][4]int = map[string][4]int{
	"21760": {1, 2, 0, 0}, // Roy Station
	"21757": {0, 0, 1, 1}, // Clearfield Station
	"21772": {1, 2, 0, 0}, // Layton Station
	"21758": {0, 0, 1, 1}, // Farmington Station
	"21759": {0, 0, 1, 1}, // Woods Cross Station
	"23113": {1, 2, 1, 2}, // North Temple Station
	"21771": {1, 2, 1, 2}, // Salt Lake Central Station
	"23080": {2, 1, 1, 1}, // Murray Central Station
	"23082": {0, 0, 2, 2}, // South Jordan Station
	"23081": {1, 2, 0, 0}, // Draper Station
	"23074": {1, 2, 1, 1}, // Lehi Station
	"23071": {0, 0, 2, 2}, // American Fork Station
	"25330": {2, 1, 2, 2}, // Vineyard Station
	"23076": {0, 0, 2, 2}, // Orem Central Station
}

func predictPlatform(d DbDeparture) int {
	if d.RouteShortName != "750" {
		return 0
	}

	data, ok := predictionData[d.StopId]
	if !ok {
		return 0
	}

	idx := 0
	if !d.PassingTrain {
		idx += 2
	}

	if d.DirectionId == 1 {
		idx += 1
	}
	return data[idx]
}

func addToTime(h, m, s, add int) (int, int, int) {
	s += add
	if s >= 60 {
		m += s / 60
		s = s % 60

		if m >= 60 {
			h += m / 60
			m = m % 60
		}
	}

	return h, m, s
}

func upcomingDeparturesHandler(w http.ResponseWriter, r *http.Request) {
	stationIds := strings.Split(r.URL.Query().Get("stationIds"), ",")
	if r.URL.Query().Get("stationIds") == "" || len(stationIds) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("missing stationIds param"))
		return
	}

	loc, _ := time.LoadLocation("America/Denver") // ew this shouldn't be hardcoded

	fromStr := r.URL.Query().Get("from")
	from := time.Now().In(loc)
	if fromStr != "" {
		fromInt, err := strconv.ParseInt(fromStr, 10, 64)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("invalid `from` param, expected UNIX timestamp"))
			return
		}

		from = time.Unix(fromInt, 0).In(loc)
	}

	// get current service IDs
	serviceIds, err := gtfs.GetActiveServiceIDs(&from, scheduleDb.DB)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal server error"))
		return
	}

	toH, toM, toS := addToTime(from.Hour(), from.Minute(), from.Second(), 60*60)
	to := from.Add(time.Hour)

	// get upcoming departures
	query, args, err := sqlx.In(`select
		routes.route_id,
		routes.route_short_name,
		routes.route_long_name,
		stop_times.departure_time,
		trips.trip_headsign,
		trips.trip_id,
		trips.direction_id,
		stop_times.stop_id,
		EXISTS (
			SELECT * from stop_times st2 inner join trips t2 on t2.trip_id = st2.trip_id where st2.stop_id = stop_times.stop_id and t2.service_id IN (?) and (st2.departure_timestamp <= stop_times.departure_timestamp + 300) and (st2.departure_timestamp >= stop_times.departure_timestamp - 300 and stop_times.trip_id != st2.trip_id AND trips.direction_id != t2.direction_id)
		) AS passing_train
	from stop_times
	inner join trips on trips.trip_id = stop_times.trip_id
	inner join routes on routes.route_id = trips.route_id
	where trips.service_id IN (?) and stop_times.stop_id IN (?) and stop_times.departure_time >= ? and stop_times.departure_time < ?
	order by stop_times.departure_time asc`, serviceIds, serviceIds, stationIds, from.Format("15:04:05"), fmt.Sprintf("%02d:%02d:%02d", toH, toM, toS))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal server error"))
		return
	}

	dbDepartures := []DbDeparture{}
	err = scheduleDb.Select(&dbDepartures, query, args...)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal server error"))
		return
	}

	departures := make([]Departure, 0, len(dbDepartures))

	for _, departure := range dbDepartures {
		t, err := time.Parse("15:04:05", departure.DepartureTime)
		if err != nil {
			continue
		}

		departures = append(departures, Departure{
			Route: Route{
				ID:        departure.RouteId,
				ShortName: departure.RouteShortName,
				LongName:  departure.RouteLongName,
			},
			DepartureTime:     time.Date(from.Year(), from.Month(), from.Day(), t.Hour(), t.Minute(), t.Second(), 0, loc),
			Headsign:          departure.TripHeadsign,
			PredictedPlatform: predictPlatform(departure),
			TripId:            departure.TripId,
		})
	}

	response, _ := json.Marshal(ApiDepartures{
		Departures: departures,
		To:         to,
	})

	w.Header().Add("Content-Type", "application/json")
	w.Write(response)
}

func main() {
	fmt.Println("Opening database...")

	_db, err := sqlx.Open("sqlite3", "uta-gtfs.db")
	if err != nil {
		log.Fatal(err)
	}

	scheduleDb = _db

	http.Handle("/", http.FileServer(http.Dir("./static")))

	http.HandleFunc("/schema.proto", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./proto/schema.proto")
	})

	http.HandleFunc("/api", vehicleHandler(""))
	http.HandleFunc("/api.json", vehicleHandler("json"))
	http.HandleFunc("/api/trips/{trip}", tripHandler)
	http.HandleFunc("/api/shapes/{shape}", shapeHandler)
	http.HandleFunc("/api/upcoming-departures", upcomingDeparturesHandler)

	port := "3000"
	if portEnv, ok := os.LookupEnv("PORT"); ok {
		port = portEnv
	}

	fmt.Printf("Started on port %s\n", port)

	err = http.ListenAndServe(":"+port, nil)
	if err != nil {
		log.Fatal(err)
	}
}
