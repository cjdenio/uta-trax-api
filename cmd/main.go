package main

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"

	pb "github.com/cjdenio/uta-trax-api/proto"
	"google.golang.org/protobuf/proto"

	_ "github.com/mattn/go-sqlite3"
)

var scheduleDb *sql.DB

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

func getVehicles() ([]*pb.VehiclePosition, error) {
	resp, err := http.Get("https://apps.rideuta.com/tms/gtfs/Vehicle")
	if err != nil {
		return nil, err
	}

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	feed := pb.FeedMessage{}
	err = proto.Unmarshal(bytes, &feed)
	if err != nil {
		return nil, err
	}

	log.Printf("Found %d vehicles...", len(feed.Entity))

	vehicles := make([]*pb.VehiclePosition, 0, len(feed.Entity))
	for _, v := range feed.Entity {
		if v.Vehicle != nil {
			vehicles = append(vehicles, v.Vehicle)
		}
	}

	return vehicles, nil
}

type TripInfo struct {
	Line      pb.VehicleFeed_Line
	Direction int32
	Headsign  string
}

var trips = make(map[string]*TripInfo)

func loadTrips() error {
	rows, err := scheduleDb.Query("SELECT route_id, trip_id, trip_headsign, direction_id FROM trips;")
	if err != nil {
		return err
	}

	for {
		if next := rows.Next(); next {
			trip_info := new(TripInfo)

			var route_id string
			var trip_id string
			var trip_headsign string
			var direction_id int32

			err = rows.Scan(&route_id, &trip_id, &trip_headsign, &direction_id)
			if err != nil {
				continue
			}

			trip_info.Headsign = trip_headsign

			trip_info.Direction = direction_id

			switch route_id {
			case "8246":
				trip_info.Line = pb.VehicleFeed_RED
				trips[trip_id] = trip_info
			case "39020":
				trip_info.Line = pb.VehicleFeed_GREEN
				trips[trip_id] = trip_info
			case "5907":
				trip_info.Line = pb.VehicleFeed_BLUE
				trips[trip_id] = trip_info
			case "45389":
				trip_info.Line = pb.VehicleFeed_STREETCAR
				trips[trip_id] = trip_info
				// case "41065":
				// 	trip_info.Line = pb.VehicleFeed_FRONTRUNNER
				// 	trips[record[trip_id]] = trip_info
			}
		} else {
			break
		}
	}

	return nil
}

func feedifyVehicles(vehicles []*pb.VehiclePosition) pb.VehicleFeed {
	vehicle_feed := make([]*pb.VehicleFeed_Vehicle, 0, len(vehicles))

	for _, vehicle := range vehicles {
		trip, ok := trips[*vehicle.Trip.TripId]
		if !ok {
			log.Printf("No matching trip '%s', skipping...", *vehicle.Trip.TripId)
			continue
		}

		vehicle_feed = append(vehicle_feed, &pb.VehicleFeed_Vehicle{
			Lat:            *vehicle.Position.Latitude,
			Lon:            *vehicle.Position.Longitude,
			Line:           trip.Line,
			Id:             *vehicle.Vehicle.Id,
			Direction:      trip.Direction,
			NearestStation: getStationForVehicle(vehicle),
			Headsign:       trip.Headsign,
		})
	}

	return pb.VehicleFeed{
		Vehicles: vehicle_feed,
	}
}

func main() {
	fmt.Println("Opening database...")

	_db, err := sql.Open("sqlite3", "uta-gtfs.db")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Loading trip data...")

	scheduleDb = _db
	if err := loadTrips(); err != nil {
		log.Fatalln(err)
	}

	// vehicles, err := getVehicles()
	// if err != nil {
	// 	log.Fatalln(err)
	// }

	// for _, v := range vehicles {
	// 	station := getStationForVehicle(v)
	// 	fmt.Println(station)
	// }

	http.Handle("/", http.FileServer(http.Dir("./static")))

	http.HandleFunc("/schema.proto", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./proto/schema.proto")
	})

	http.HandleFunc("/api", func(w http.ResponseWriter, r *http.Request) {
		vehicles, err := getVehicles()
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		feed := feedifyVehicles(vehicles)
		b, err := proto.Marshal(&feed)
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Add("Content-Type", "application/protobuf")
		w.Write(b)
	})

	port := "3000"
	if portEnv, ok := os.LookupEnv("PORT"); ok {
		port = portEnv
	}

	fmt.Printf("Started on port %s\n", port)

	http.ListenAndServe(":"+port, nil)
}
