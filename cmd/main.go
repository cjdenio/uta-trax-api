package main

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"slices"
	"strconv"

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
	f, err := os.Open("gtfs/trips.txt")
	if err != nil {
		return err
	}

	r := csv.NewReader(f)

	header, err := r.Read()
	if err != nil {
		return err
	}

	route_id := slices.Index(header, "route_id")
	trip_id := slices.Index(header, "trip_id")
	direction_id := slices.Index(header, "direction_id")
	trip_headsign := slices.Index(header, "trip_headsign")

	for {
		if record, err := r.Read(); err == nil {
			trip_info := new(TripInfo)
			trip_info.Headsign = record[trip_headsign]

			direction, err := strconv.ParseInt(record[direction_id], 10, 32)
			if err == nil {
				trip_info.Direction = int32(direction)
			}

			switch record[route_id] {
			case "8246":
				trip_info.Line = pb.VehicleFeed_RED
				trips[record[trip_id]] = trip_info
			case "39020":
				trip_info.Line = pb.VehicleFeed_GREEN
				trips[record[trip_id]] = trip_info
			case "5907":
				trip_info.Line = pb.VehicleFeed_BLUE
				trips[record[trip_id]] = trip_info
			case "45389":
				trip_info.Line = pb.VehicleFeed_STREETCAR
				trips[record[trip_id]] = trip_info
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
	if err := loadTrips(); err != nil {
		log.Fatalln(err)
	}

	_db, err := sql.Open("sqlite3", "uta-gtfs.db")
	if err != nil {
		log.Fatal(err)
	}
	scheduleDb = _db

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
			log.Fatalln(err)
		}

		feed := feedifyVehicles(vehicles)
		b, _ := proto.Marshal(&feed)
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
