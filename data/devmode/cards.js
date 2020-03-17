"use strict";

Vue.component('map-card-tag', {
	template: '#map-card-tpl',
	props: ["list"],
	data: function () {
		return {
			map: null, // set it on mounted event
			markers: null,

			iid: makestrid(10) // instance ID
		};
	},
	computed: {
		isvisible() {
			if (this.list.length > 0) {
				Vue.nextTick(() => {
					this.updatemarkers();
					this.map.invalidateSize();
				});
				return true;
			}
			return false;
		}
	},
	methods: {
		// setup markers on map, remove previous
		updatemarkers() {
			const markers = L.markerClusterGroup();
			for (const file of this.list) {
				L.marker([file.latitude, file.longitude], {
					title: file.name
				})
					.addTo(markers)
					.bindPopup(makemarkercontent(file));
			}

			this.map.flyToBounds(markers.getBounds(), {
				padding: [20, 20]
			});

			// remove previous set
			if (this.markers) {
				this.map.removeLayer(this.markers);
			}
			// add new set
			this.map.addLayer(markers);
			this.markers = markers;
		}
	},
	mounted() {
		const tiles = L.tileLayer('https://api.tiles.mapbox.com/v4/{id}/{z}/{x}/{y}.jpg?access_token=pk.eyJ1IjoibWFwYm94IiwiYSI6ImNpejY4NXVycTA2emYycXBndHRqcmZ3N3gifQ.rJcFIG214AriISLbB6B5aw', {
			attribution: 'Map data &copy; <a href="https://www.openstreetmap.org/" target="_blank">OpenStreetMap</a> contributors, ' +
				'Imagery &copy <a href="https://www.mapbox.com/" target="_blank">Mapbox</a>',
			minZoom: 2,
			id: 'mapbox.streets-satellite'
		});

		this.map = L.map(this.$refs.map, {
			center: [0, 0],
			zoom: 8,
			layers: [tiles]
		});
	}
});

// The End.
