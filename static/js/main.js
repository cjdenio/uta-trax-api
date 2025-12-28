import * as shapes from "./shapes.js";
import protobuf from "https://cdn.jsdelivr.net/npm/protobufjs@8.0.0/dist/protobuf.js/+esm";
import * as L from "https://unpkg.com/leaflet@1.9.4/dist/leaflet-src.esm.js";

let lastUpdated;

function displayLastUpdated() {
  document.getElementById("last-updated").innerText = `${Math.floor((Date.now() - (lastUpdated * 1000)) / 1000)}s ago`
  console.log("yeah")
}

setInterval(displayLastUpdated, 1000)

const ICONS = {
  bus: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 640 640"><!--!Font Awesome Free v7.1.0 by @fontawesome - https://fontawesome.com License - https://fontawesome.com/license/free Copyright 2025 Fonticons, Inc.--><path d="M192 64C139 64 96 107 96 160L96 448C96 477.8 116.4 502.9 144 510L144 544C144 561.7 158.3 576 176 576L192 576C209.7 576 224 561.7 224 544L224 512L416 512L416 544C416 561.7 430.3 576 448 576L464 576C481.7 576 496 561.7 496 544L496 510C523.6 502.9 544 477.8 544 448L544 160C544 107 501 64 448 64L192 64zM160 240C160 222.3 174.3 208 192 208L296 208L296 320L192 320C174.3 320 160 305.7 160 288L160 240zM344 320L344 208L448 208C465.7 208 480 222.3 480 240L480 288C480 305.7 465.7 320 448 320L344 320zM192 384C209.7 384 224 398.3 224 416C224 433.7 209.7 448 192 448C174.3 448 160 433.7 160 416C160 398.3 174.3 384 192 384zM448 384C465.7 384 480 398.3 480 416C480 433.7 465.7 448 448 448C430.3 448 416 433.7 416 416C416 398.3 430.3 384 448 384zM248 136C248 122.7 258.7 112 272 112L368 112C381.3 112 392 122.7 392 136C392 149.3 381.3 160 368 160L272 160C258.7 160 248 149.3 248 136z"/></svg>`,
  tram: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 640 640"><!--!Font Awesome Free v7.1.0 by @fontawesome - https://fontawesome.com License - https://fontawesome.com/license/free Copyright 2025 Fonticons, Inc.--><path d="M128 72C128 58.7 138.7 48 152 48L488 48C501.3 48 512 58.7 512 72L512 104C512 117.3 501.3 128 488 128C474.7 128 464 117.3 464 104L464 96L344 96L344 160L384 160C437 160 480 203 480 256L480 416C480 447.2 465.1 475 442 492.5L506.3 568.5C514.9 578.6 513.6 593.8 503.5 602.3C493.4 610.8 478.2 609.6 469.7 599.5L395.1 511.4C391.5 511.8 387.8 512 384 512L256 512C252.2 512 248.5 511.8 244.9 511.4L170.3 599.5C161.7 609.6 146.6 610.9 136.5 602.3C126.4 593.7 125.1 578.6 133.7 568.5L198 492.5C174.9 475 160 447.2 160 416L160 256C160 203 203 160 256 160L296 160L296 96L176 96L176 104C176 117.3 165.3 128 152 128C138.7 128 128 117.3 128 104L128 72zM256 224C238.3 224 224 238.3 224 256L224 288C224 305.7 238.3 320 256 320L384 320C401.7 320 416 305.7 416 288L416 256C416 238.3 401.7 224 384 224L256 224zM288 416C288 398.3 273.7 384 256 384C238.3 384 224 398.3 224 416C224 433.7 238.3 448 256 448C273.7 448 288 433.7 288 416zM384 448C401.7 448 416 433.7 416 416C416 398.3 401.7 384 384 384C366.3 384 352 398.3 352 416C352 433.7 366.3 448 384 448z"/></svg>`,
  train: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 640 640"><!--!Font Awesome Free v7.1.0 by @fontawesome - https://fontawesome.com License - https://fontawesome.com/license/free Copyright 2025 Fonticons, Inc.--><path d="M128 160C128 107 171 64 224 64L416 64C469 64 512 107 512 160L512 416C512 456.1 487.4 490.5 452.5 504.8L506.4 568.5C515 578.6 513.7 593.8 503.6 602.3C493.5 610.8 478.3 609.6 469.8 599.5L395.8 512L244.5 512L170.5 599.5C161.9 609.6 146.8 610.9 136.7 602.3C126.6 593.7 125.3 578.6 133.9 568.5L187.8 504.8C152.6 490.5 128 456.1 128 416L128 160zM192 192L192 288C192 305.7 206.3 320 224 320L416 320C433.7 320 448 305.7 448 288L448 192C448 174.3 433.7 160 416 160L224 160C206.3 160 192 174.3 192 192zM320 448C337.7 448 352 433.7 352 416C352 398.3 337.7 384 320 384C302.3 384 288 398.3 288 416C288 433.7 302.3 448 320 448z"/></svg>`,
};

var map = L.map("map").setView([40.656734, -111.890818], 12);
L.tileLayer(
  "https://api.mapbox.com/styles/v1/mapbox/light-v11/tiles/256/{z}/{x}/{y}@2x?access_token=pk.eyJ1IjoiY2pkZW5pbyIsImEiOiJjbHdiMG52amcwaGd4MmttbWtlOWt5Mm1iIn0.GhHTt4W_mZpQcLYNkhsG_w",
  {
    maxZoom: 19,
    attribution:
      '&copy; <a href="http://www.openstreetmap.org/copyright">OpenStreetMap</a>',
  }
).addTo(map);

const busLayer = L.layerGroup().addTo(map);
const brtLayer = L.layerGroup().addTo(map);
const traxLayer = L.layerGroup().addTo(map);
const frontRunnerLayer = L.layerGroup().addTo(map);

L.polyline(shapes.blueLine, { color: "#004a97" }).addTo(map);
L.polyline(shapes.redLine, { color: "#be2036" }).addTo(map);
L.polyline(shapes.greenLine, { color: "#2eb566" }).addTo(map);
L.polyline(shapes.sLine, { color: "#77777a" }).addTo(map);
L.polyline(shapes.frontRunner, { color: "#c227b9" }).addTo(map);

function clearMap() {
  busLayer.clearLayers();
  brtLayer.clearLayers();
  traxLayer.clearLayers();
  frontRunnerLayer.clearLayers();
}

protobuf.load("/schema.proto").then((root) => {
  const RouteType = root.VehicleFeed.Route.RouteType;

  function routeDesignator(route) {
    if (route.id == "92235") {
      return "OGX";
    } else if (route.id == "3686") {
      return "UVX";
    } else if (route.type == RouteType.BUS) {
      return "#" + route.shortName;
    } else {
      return route.longName;
    }
  }

  async function reload() {
    const bin = await fetch("/api").then((r) => r.arrayBuffer());
    const { vehicles, info } = root
      .lookupType("VehicleFeed")
      .decode(new Uint8Array(bin));

    clearMap();

    lastUpdated = info.lastUpdate
    displayLastUpdated()

    vehicles.forEach((vehicle) => {
      L.marker([vehicle.lat, vehicle.lon], {
        zIndexOffset:
          vehicle.route.type == RouteType.TRAM ||
          vehicle.route.type == RouteType.RAIL
            ? 1000
            : 1,
        icon: L.divIcon({
          className: "",
          html: `<div class="vehicle ${
            vehicle.route.type == RouteType.BUS &&
            !["92235", "3686"].includes(vehicle.route.id)
              ? "vehicle-plain"
              : ""
          } ${
            vehicle.route.type == RouteType.BUS ? "vehicle-small" : ""
          }" style="--color: #${vehicle.route.color};">
                    <div style="position: absolute; inset: 0; display: flex; align-items: center; justify-content: center;">
                      <svg style="transform: rotate(${
                        vehicle.bearing
                      }deg) translateY(-12px); width: 18px;" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 640 640"><!--!Font Awesome Free v7.1.0 by @fontawesome - https://fontawesome.com License - https://fontawesome.com/license/free Copyright 2025 Fonticons, Inc.--><path d="M300.3 199.2C312.9 188.9 331.4 189.7 343.1 201.4L471.1 329.4C480.3 338.6 483 352.3 478 364.3C473 376.3 461.4 384 448.5 384L192.5 384C179.6 384 167.9 376.2 162.9 364.2C157.9 352.2 160.7 338.5 169.9 329.4L297.9 201.4L300.3 199.2z"/></svg>
                    </div>
                    ${
                      vehicle.route.type == RouteType.BUS
                        ? ICONS.bus
                        : vehicle.route.type == RouteType.TRAM
                        ? ICONS.tram
                        : vehicle.route.type == RouteType.RAIL
                        ? ICONS.train
                        : ""
                    }
                  </div>`,
        }),
      })
        .addTo(
          ["92235", "3686"].includes(vehicle.route.id)
            ? brtLayer
            : vehicle.route.type == RouteType.TRAM
            ? traxLayer
            : vehicle.route.type == RouteType.RAIL
            ? frontRunnerLayer
            : busLayer
        )
        .bindPopup(
          `${routeDesignator(vehicle.route)} to ${vehicle.headsign.replace(
            /^to /i,
            ""
          )}<br />@ ${vehicle.nearestStation?.name}`
        );
    });
  }

  reload();

  setInterval(reload, 5000);

  for (const checkbox of document.querySelectorAll(".layer-toggle")) {
    checkbox.addEventListener("input", (e) => {
      let layer;

      switch (e.target.name) {
        case "bus":
          layer = busLayer;
          break;
        case "brt":
          layer = brtLayer;
          break;
        case "trax":
          layer = traxLayer;
          break;
        case "frontrunner":
          layer = frontRunnerLayer;
          break;
      }

      if (e.target.checked) {
        layer?.addTo(map);
      } else {
        layer?.remove();
      }
    });
  }
});
