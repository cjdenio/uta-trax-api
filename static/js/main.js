import * as L from "https://cdn.jsdelivr.net/npm/leaflet@1.9.4/dist/leaflet-src.esm.js";
import Alpine from "https://cdn.jsdelivr.net/npm/alpinejs@3.x.x/dist/module.esm.min.js";

import * as shapes from "./shapes.js";
import stations from "./stations.js"

/** @type {number} */
let lastUpdated;

function displayLastUpdated() {
  document.getElementById("last-updated").innerText = `${Math.floor((Date.now() - (lastUpdated * 1000)) / 1000)}s ago`
}

setInterval(displayLastUpdated, 1000)

const RouteType = {
  TRAM: 1,
  RAIL: 3,
  BUS: 4,
}

const ICONS = {
  bus: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 640 640"><!--!Font Awesome Free v7.1.0 by @fontawesome - https://fontawesome.com License - https://fontawesome.com/license/free Copyright 2025 Fonticons, Inc.--><path d="M192 64C139 64 96 107 96 160L96 448C96 477.8 116.4 502.9 144 510L144 544C144 561.7 158.3 576 176 576L192 576C209.7 576 224 561.7 224 544L224 512L416 512L416 544C416 561.7 430.3 576 448 576L464 576C481.7 576 496 561.7 496 544L496 510C523.6 502.9 544 477.8 544 448L544 160C544 107 501 64 448 64L192 64zM160 240C160 222.3 174.3 208 192 208L296 208L296 320L192 320C174.3 320 160 305.7 160 288L160 240zM344 320L344 208L448 208C465.7 208 480 222.3 480 240L480 288C480 305.7 465.7 320 448 320L344 320zM192 384C209.7 384 224 398.3 224 416C224 433.7 209.7 448 192 448C174.3 448 160 433.7 160 416C160 398.3 174.3 384 192 384zM448 384C465.7 384 480 398.3 480 416C480 433.7 465.7 448 448 448C430.3 448 416 433.7 416 416C416 398.3 430.3 384 448 384zM248 136C248 122.7 258.7 112 272 112L368 112C381.3 112 392 122.7 392 136C392 149.3 381.3 160 368 160L272 160C258.7 160 248 149.3 248 136z"/></svg>`,
  tram: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 640 640"><!--!Font Awesome Free v7.1.0 by @fontawesome - https://fontawesome.com License - https://fontawesome.com/license/free Copyright 2025 Fonticons, Inc.--><path d="M128 72C128 58.7 138.7 48 152 48L488 48C501.3 48 512 58.7 512 72L512 104C512 117.3 501.3 128 488 128C474.7 128 464 117.3 464 104L464 96L344 96L344 160L384 160C437 160 480 203 480 256L480 416C480 447.2 465.1 475 442 492.5L506.3 568.5C514.9 578.6 513.6 593.8 503.5 602.3C493.4 610.8 478.2 609.6 469.7 599.5L395.1 511.4C391.5 511.8 387.8 512 384 512L256 512C252.2 512 248.5 511.8 244.9 511.4L170.3 599.5C161.7 609.6 146.6 610.9 136.5 602.3C126.4 593.7 125.1 578.6 133.7 568.5L198 492.5C174.9 475 160 447.2 160 416L160 256C160 203 203 160 256 160L296 160L296 96L176 96L176 104C176 117.3 165.3 128 152 128C138.7 128 128 117.3 128 104L128 72zM256 224C238.3 224 224 238.3 224 256L224 288C224 305.7 238.3 320 256 320L384 320C401.7 320 416 305.7 416 288L416 256C416 238.3 401.7 224 384 224L256 224zM288 416C288 398.3 273.7 384 256 384C238.3 384 224 398.3 224 416C224 433.7 238.3 448 256 448C273.7 448 288 433.7 288 416zM384 448C401.7 448 416 433.7 416 416C416 398.3 401.7 384 384 384C366.3 384 352 398.3 352 416C352 433.7 366.3 448 384 448z"/></svg>`,
  s70: `
  <svg width="100%" height="100%" viewBox="0 0 243 373" version="1.1" xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" xml:space="preserve" xmlns:serif="http://www.serif.com/" style="fill-rule:evenodd;clip-rule:evenodd;stroke-linecap:round;stroke-linejoin:round;stroke-miterlimit:1.5;">
      <g transform="matrix(1,0,0,1,-692.76,-629.684)">
          <g transform="matrix(1,0,0,1,558.086,539.672)">
              <g transform="matrix(0.791673,0,0,0.791673,67.6053,62.4782)">
                  <path d="M391.223,62.262L391.223,426.631C391.223,441.8 378.908,454.115 363.739,454.115L112.201,454.115C97.033,454.115 84.718,441.8 84.718,426.631L84.718,62.262C84.718,47.093 97.033,34.778 112.201,34.778L363.739,34.778C378.908,34.778 391.223,47.093 391.223,62.262ZM349.718,308.746L293.694,308.888L291.024,339.381C291.024,339.381 333.384,338.553 343.594,333.746C353.44,329.11 349.718,308.746 349.718,308.746ZM126.222,308.746C126.222,308.746 122.5,329.11 132.346,333.746C142.556,338.553 184.916,339.381 184.916,339.381L182.246,308.888L126.222,308.746ZM237.97,45.317C231.106,45.317 225.533,50.889 225.533,57.754C225.533,64.618 231.106,70.19 237.97,70.19C244.834,70.19 250.407,64.618 250.407,57.754C250.407,50.889 244.834,45.317 237.97,45.317ZM141.445,71.353C141.445,68.703 139.294,66.551 136.644,66.551L122.626,66.551C119.976,66.551 117.825,68.703 117.825,71.353C117.825,74.002 119.976,76.154 122.626,76.154L136.644,76.154C139.294,76.154 141.445,74.002 141.445,71.353ZM113.026,93.102L129.099,274.048L343.416,274.048L362.915,93.102L113.026,93.102ZM358.116,71.353C358.116,68.703 355.964,66.551 353.314,66.551L339.297,66.551C336.647,66.551 334.495,68.703 334.495,71.353C334.495,74.002 336.647,76.154 339.297,76.154L353.314,76.154C355.964,76.154 358.116,74.002 358.116,71.353Z" style="fill:white;"/>
              </g>
          </g>
          <g transform="matrix(1,0,0,1,558.086,539.672)">
              <g transform="matrix(1,0,0,1,9.52413,0)">
                  <g>
                      <g transform="matrix(1,0,0,1,25.0181,-20.008)">
                          <path d="M205.143,384.444L115.553,474.034" style="fill:none;stroke:white;stroke-width:16.67px;"/>
                      </g>
                      <g transform="matrix(-1,0,0,1,467.934,-20.008)">
                          <path d="M205.143,384.444L115.553,474.034" style="fill:none;stroke:white;stroke-width:16.67px;"/>
                      </g>
                  </g>
              </g>
          </g>
      </g>
  </svg>
  `,
  sd100: `
  <svg width="100%" height="100%" viewBox="0 0 312 375" version="1.1" xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" xml:space="preserve" xmlns:serif="http://www.serif.com/" style="fill-rule:evenodd;clip-rule:evenodd;stroke-linecap:round;stroke-linejoin:round;stroke-miterlimit:1.5;">
      <g transform="matrix(1,0,0,1,-100.641,-630.634)">
          <g transform="matrix(1,0,0,1,0,543.101)">
              <g transform="matrix(1.01452,0,0,0.955288,17.5049,12.4596)">
                  <path d="M361.398,79.497L389.212,85.321L389.212,428.171L367.378,428.171L359.138,376.654L111.963,375.79L103.724,425.55L81.947,425.55L84.182,88.988L111.963,78.588L361.398,79.497ZM311.478,97.02C311.478,91.619 307.349,87.234 302.264,87.234L168.896,87.234C163.81,87.234 159.681,91.619 159.681,97.02L159.681,108.709C159.681,114.11 163.81,118.494 168.896,118.494L302.264,118.494C307.349,118.494 311.478,114.11 311.478,108.709L311.478,97.02ZM364.31,256.302L368.881,254.952C373.145,253.692 376.096,249.567 376.096,244.867L376.096,145.773C376.096,141.072 373.145,136.948 368.881,135.688L364.31,134.337C361.342,133.46 358.158,134.108 355.705,136.087C353.252,138.066 351.811,141.15 351.811,144.422L351.811,246.218C351.811,249.49 353.252,252.573 355.705,254.553C358.158,256.532 361.342,257.18 364.31,256.302ZM159.787,336.745C159.787,333.476 157.288,330.821 154.209,330.821L135.081,330.821C132.003,330.821 129.504,333.476 129.504,336.745L129.504,348.592C129.504,351.861 132.003,354.515 135.081,354.515L154.209,354.515C157.288,354.515 159.787,351.861 159.787,348.592L159.787,336.745ZM341.656,336.745C341.656,333.476 339.156,330.821 336.078,330.821L316.95,330.821C313.872,330.821 311.373,333.476 311.373,336.745L311.373,348.592C311.373,351.861 313.872,354.515 316.95,354.515L336.078,354.515C339.156,354.515 341.656,351.861 341.656,348.592L341.656,336.745ZM106.695,256.302C109.663,257.18 112.847,256.532 115.3,254.553C117.754,252.573 119.194,249.49 119.194,246.218L119.194,144.422C119.194,141.15 117.754,138.066 115.3,136.087C112.847,134.108 109.663,133.46 106.695,134.337L102.124,135.688C97.86,136.948 94.909,141.072 94.909,145.773L94.909,244.867C94.909,249.567 97.86,253.692 102.124,254.952L106.695,256.302ZM338.471,247.747L332.559,143.523C332.246,137.996 327.934,133.684 322.72,133.684L148.439,133.684C143.226,133.684 138.914,137.996 138.6,143.523L132.688,247.747C132.525,250.627 133.489,253.452 135.351,255.552C137.214,257.653 139.81,258.845 142.528,258.845L328.632,258.845C331.349,258.845 333.945,257.653 335.808,255.552C337.671,253.452 338.634,250.627 338.471,247.747Z" style="fill:white;"/>
              </g>
          </g>
          <g transform="matrix(1,0,0,1,0,543.101)">
              <g transform="matrix(1,0,0,1,6.63059,0)">
                  <g transform="matrix(1,0,0,1,0.525499,-20.008)">
                      <path d="M205.143,384.444L115.553,474.034" style="fill:none;stroke:white;stroke-width:16.67px;"/>
                  </g>
                  <g transform="matrix(-1,0,0,1,499.222,-20.008)">
                      <path d="M205.143,384.444L115.553,474.034" style="fill:none;stroke:white;stroke-width:16.67px;"/>
                  </g>
              </g>
          </g>
      </g>
  </svg>
  `,
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

map.on("zoomstart", () => {
  document.body.classList.add("skip-vehicle-animation")
})
map.on("zoomend", () => {
  document.body.classList.remove("skip-vehicle-animation")
})

const busLayer = L.layerGroup().addTo(map);
const brtLayer = L.layerGroup().addTo(map);
const traxLayer = L.layerGroup().addTo(map);
const frontRunnerLayer = L.layerGroup().addTo(map);

L.polyline(shapes.blueLine, { color: "#004a97", zIndexOffset: 10 }).addTo(map);
L.polyline(shapes.redLine, { color: "#be2036", zIndexOffset: 10 }).addTo(map);
L.polyline(shapes.greenLine, { color: "#2eb566", zIndexOffset: 10 }).addTo(map);
L.polyline(shapes.sLine, { color: "#77777a", zIndexOffset: 10 }).addTo(map);
L.polyline(shapes.frontRunner, { color: "#c227b9", zIndexOffset: 10 }).addTo(map);

// render stations
for (const station of stations) {
    let color = "";
    switch (station.lines[0]) {
        case "red":
            color = "#be2036"
            break
        case "blue":
            color = "#004a97"
            break
        case "green":
            color = "#2eb566"
            break
        case "s-line":
            color = "#77777a"
            break
        case "frontrunner":
            color = "#c227b9"
            break
    }

    L.marker([station.lat, station.lon], {
        zIndexOffset: 2000,
        icon: L.divIcon({
            className: "",
            html: `<div class="station" style="--color: ${color};"></div>`,
            iconSize: [5, 5],
        })
    }).addTo(map).bindPopup(station.name.replace(" Station", ""))
}

function clearMap() {
  busLayer.clearLayers();
  brtLayer.clearLayers();
  traxLayer.clearLayers();
  frontRunnerLayer.clearLayers();
}

function routeDesignator(route) {
  if (route.id == "92235") {
    return "OGX";
  } else if (route.id == "3686") {
    return "UVX";
  } else if (route.id == "87711") {
    return "MVX";
  } else if (route.type == RouteType.BUS) {
    return "#" + route.short_name;
  } else {
    return route.long_name;
  }
}

function populateTotals(vehicles) {
  const totals = {
    bus: 0,
    trax: 0,
    frontrunner: 0,
  }

  for (const vehicle of vehicles) {
    switch(vehicle.route.type) {
      case RouteType.BUS:
        totals.bus++;
        break;
      case RouteType.TRAM:
        totals.trax++;
        break;
      case RouteType.RAIL:
        totals.frontrunner++;
        break;
    }
  }

  document.getElementById("count-bus").innerText = totals.bus.toString()
  document.getElementById("count-trax").innerText = totals.trax.toString()
  document.getElementById("count-frontrunner").innerText = totals.frontrunner.toString()
}

let bannerVisible = false

let currentVehicles = new Map()

function vehicleSvgIcon(vehicle) {
  if (vehicle.route.type == RouteType.BUS) {
    return ICONS.bus
  } else if (vehicle.route.type == RouteType.TRAM) {
    if (localStorage.getItem("new_icons") == "true") {
      if (vehicle.id.startsWith("10")) {
        return ICONS.sd100
      } else {
        return ICONS.s70
      }
    } else {
      return ICONS.tram
    }
  } else if (vehicle.route.type == RouteType.RAIL) {
    return ICONS.train
  }
}

function renderVehicleIcon(vehicle) {
  return L.divIcon({
    className: "",
    html: `<div class="vehicle ${
      vehicle.route.type == RouteType.BUS &&
      !["92235", "3686", "87711"].includes(vehicle.route.id)
        ? "vehicle-plain"
        : ""
    } ${
      vehicle.route.type == RouteType.BUS ? "vehicle-small" : ""
    }" style="--color: #${vehicle.route.color};">
              <div style="position: absolute; inset: 0; display: flex; align-items: center; justify-content: center;">
                <svg style="transform: rotate(${
                  vehicle.bearing ?? 0
                }deg) translateY(-12px); width: 18px;" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 640 640"><!--!Font Awesome Free v7.1.0 by @fontawesome - https://fontawesome.com License - https://fontawesome.com/license/free Copyright 2025 Fonticons, Inc.--><path d="M300.3 199.2C312.9 188.9 331.4 189.7 343.1 201.4L471.1 329.4C480.3 338.6 483 352.3 478 364.3C473 376.3 461.4 384 448.5 384L192.5 384C179.6 384 167.9 376.2 162.9 364.2C157.9 352.2 160.7 338.5 169.9 329.4L297.9 201.4L300.3 199.2z"/></svg>
              </div>
              ${vehicleSvgIcon(vehicle)}
            </div>`,
  })
}

function vehiclePopupContent(vehicle) {
  let content = `<b>${routeDesignator(vehicle.route)}</b>`
  if (vehicle.headsign) {
    content += ` to <b>${vehicle.headsign.replace(
              /^to /i,
              ""
            )}</b>`
  }

  if (vehicle.nearest_station) {
    let station = vehicle.nearest_station.name
    if (station.toLowerCase().endsWith("station")) {
      station = station.replace(/ station$/i, "")
    }
    content += `<br />@ ${station}`
  }

  content += `<br /><br /><small>vehicle #: ${vehicle.id} &middot; trip ID: ${vehicle.trip_id}</small>`

  return content
}

function renderVehicle(vehicle) {
  return L.marker([vehicle.lat, vehicle.lon], {
          zIndexOffset:
            vehicle.route.type == RouteType.TRAM ||
            vehicle.route.type == RouteType.RAIL
              ? 4000
              : 3000,
          icon: renderVehicleIcon(vehicle),
        }).bindPopup(vehiclePopupContent(vehicle));
}

async function reload() {
  const { vehicles, info } = await fetch("/api", { headers: { Accept: "application/json" } }).then((r) => r.json());

  lastUpdated = info.last_update
  displayLastUpdated()

  populateTotals(vehicles)
  if (vehicles.length == 0) {
    if (!bannerVisible) {
      document.getElementById("banner").classList.remove("hidden")
      bannerVisible = true
      map.invalidateSize()
    }
  } else {
    if (bannerVisible) {
      document.getElementById("banner").classList.add("hidden")
      bannerVisible = false
      map.invalidateSize()
    }
  }

  const seenVehicleIds = new Set()

  vehicles.forEach((vehicle) => {
    seenVehicleIds.add(vehicle.id)

    if (currentVehicles.has(vehicle.id)) { // if this vehicle is already on the map, update its location
      currentVehicles.get(vehicle.id).marker.setLatLng([vehicle.lat, vehicle.lon])
      currentVehicles.get(vehicle.id).marker.setIcon(renderVehicleIcon(vehicle))
      currentVehicles.get(vehicle.id).marker.setPopupContent(vehiclePopupContent(vehicle))
    } else {
      const marker = renderVehicle(vehicle)
      marker.addTo(
        ["92235", "3686", "87711"].includes(vehicle.route.id)
          ? brtLayer
          : vehicle.route.type == RouteType.TRAM
            ? traxLayer
            : vehicle.route.type == RouteType.RAIL
              ? frontRunnerLayer
              : busLayer
      );
      currentVehicles.set(vehicle.id, {
        vehicle,
        marker,
      })
    }
  });

  // remove any vehicles that should be removed
  for (const key of currentVehicles.keys()) {
    if (!seenVehicleIds.has(key)) {
      currentVehicles.get(key).marker.remove()
      currentVehicles.delete(key)
    }
  }
}

reload();

setInterval(reload, 5000);

Alpine.data('app', () => ({
  busLayer: true,
  brtLayer: true,
  traxLayer: true,
  frontRunnerLayer: true,

  toggleLayer(name) {
    let layer;

    switch (name) {
      case "bus":
        layer = busLayer;
        break;
      case "brt":
        layer = brtLayer;
        break;
      case "trax":
        layer = traxLayer;
        break;
      case "frontRunner":
        layer = frontRunnerLayer;
        break;
    }

    let oldState = this[`${name}Layer`]

    if (!oldState) {
      layer?.addTo(map);
    } else {
      layer?.remove();
    }

    this[`${name}Layer`] = !oldState
  },
}))

Alpine.start()
