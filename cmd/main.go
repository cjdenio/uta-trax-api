package main

import (
	"encoding/csv"
	"io"
	"log"
	"net/http"
	"os"
	"slices"

	pb "github.com/cjdenio/uta-trax-api/proto"
	"google.golang.org/protobuf/proto"
)

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
	Direction int
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
	// direction_id := slices.Index(header, "direction_id")

	for {
		if record, err := r.Read(); err == nil {
			trip_info := new(TripInfo)

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
			Lat:  *vehicle.Position.Latitude,
			Lon:  *vehicle.Position.Longitude,
			Line: trip.Line,
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

	http.DefaultServeMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		vehicles, err := getVehicles()
		if err != nil {
			log.Fatalln(err)
		}

		feed := feedifyVehicles(vehicles)
		b, _ := proto.Marshal(&feed)
		w.Header().Add("Content-Type", "application/protobuf")
		w.Write(b)
	})

	http.ListenAndServe(":"+os.Getenv("PORT"), nil)
}
