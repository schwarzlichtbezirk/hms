"use strict";

Vue.component('dir-card-tag', {
	template: '#dir-card-tpl',
	props: ["list"],
	data: function () {
		return {
			selfile: null, // current selected item
			order: 1,
			listmode: "mdicon",
			iid: makestrid(10) // instance ID
		};
	},
	computed: {
		isvisible() {
			this.selfile = null;
			if (this.list.length > 0) {
				return true;
			}
			return false;
		},
		// sorted subfolders list
		sortedlist() {
			return this.list.slice().sort((v1, v2) => {
				return this.order*(v1.name.toLowerCase() > v2.name.toLowerCase() ? 1 : -1);
			});
		},

		clsfilelist() {
			switch (this.listmode) {
				case "lgicon":
					return 'align-items-center';
				case "mdicon":
					return 'align-items-start';
			}
		},

		clsorder() {
			return this.order > 0
				? 'arrow_downward'
				: 'arrow_upward';
		},
		clslistmode() {
			switch (this.listmode) {
				case "lgicon":
					return 'view_module';
				case "mdicon":
					return 'subject';
			}
		},

		iconmodetag() {
			switch (this.listmode) {
				case "lgicon":
					return 'img-icon-tag';
				case "mdicon":
					return 'file-icon-tag';
			}
		},

		hintorder() {
			return this.order > 0
				? "direct order"
				: "reverse order";
		},
		hintlist() {
			switch (this.listmode) {
				case "lgicon":
					return "large icons";
				case "mdicon":
					return "middle icons";
			}
		}
	},
	methods: {
		onorder() {
			this.order = -this.order;
		},
		onlistmode() {
			switch (this.listmode) {
				case "lgicon":
					this.listmode = 'mdicon';
					break;
				case "mdicon":
					this.listmode = 'lgicon';
					break;
			}
		},

		onselect(file) {
			this.selfile = file;
			this.$emit('select', file);
		},
		onopen(file) {
			this.$emit('open', file);
		},
		onunselect() {
			this.selfile = null;
		}
	}
});

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
