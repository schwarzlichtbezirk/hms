"use strict";

const sortbyalpha = "name";
const sortbysize = "size";
const unsorted = "";

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
			(() => {
				for (const file of this.list) {
					if (file === this.selfile) return;
				}
				this.selfile = null;
			})();
			return this.list.length > 0;
		},
		// sorted subfolders list
		sortedlist() {
			return this.list.slice().sort((v1, v2) => {
				return this.order * (v1.name.toLowerCase() > v2.name.toLowerCase() ? 1 : -1);
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

		disshared() {
			return !this.selfile;
		},
		clsshared() {
			return { active: this.selfile && this.selfile.pref };
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
		onshare() {
			this.$emit('share', this.selfile);
		},
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
		},
		onopen(file) {
			this.$emit('open', file);
		},
		onunselect() {
			this.onselect(null);
		}
	}
});

Vue.component('file-card-tag', {
	template: '#file-card-tpl',
	props: ["list"],
	data: function () {
		return {
			selfile: null, // current selected item
			playbackfile: null,
			order: 1,
			sortmode: sortbyalpha,
			listmode: "mdicon",
			music: true, video: true, photo: true, pdf: true, books: true, other: false,
			viewer: null, // file viewers

			iid: makestrid(10) // instance ID
		};
	},
	computed: {
		isvisible() {
			(() => {
				for (const file of this.list) {
					if (file === this.selfile) return;
				}
				this.selfile = null;
				this.closeviewer();
			})();
			return this.list.length > 0;
		},
		// filtered sorted playlist
		playlist() {
			const res = [];
			for (const file of this.list) {
				if (this.showitem(file)) {
					res.push(file);
				}
			}
			if (this.sortmode === sortbyalpha) {
				res.sort((v1, v2) => {
					return this.order * (v1.name.toLowerCase() > v2.name.toLowerCase() ? 1 : -1);
				});
			} else if (this.sortmode === sortbysize) {
				res.sort((v1, v2) => {
					if (v1.size === v2.size) {
						return this.order * (v1.name.toLowerCase() > v2.name.toLowerCase() ? 1 : -1);
					} else {
						return this.order * (v1.size > v2.size ? 1 : -1);
					}
				});
			}
			return res;
		},

		clsfilelist() {
			switch (this.listmode) {
				case "lgicon":
					return 'align-items-center';
				case "mdicon":
					return 'align-items-start';
			}
		},

		disshared() {
			return !this.selfile;
		},
		clsshared() {
			return { active: this.selfile && this.selfile.pref };
		},

		clsorder() {
			return this.order > 0
				? 'arrow_downward'
				: 'arrow_upward';
		},
		clssortmode() {
			switch (this.sortmode) {
				case sortbyalpha:
					return "sort_by_alpha";
				case sortbysize:
					return "sort";
				case unsorted:
					return "reorder";
			}
		},
		clslistmode() {
			switch (this.listmode) {
				case "lgicon":
					return 'view_module';
				case "mdicon":
					return 'subject';
			}
		},

		clsmusic() {
			return { active: this.music };
		},
		clsvideo() {
			return { active: this.video };
		},
		clsphoto() {
			return { active: this.photo };
		},
		clspdf() {
			return { active: this.pdf };
		},
		clsbooks() {
			return { active: this.books };
		},
		clsother() {
			return { active: this.other };
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
		hintsortmode() {
			switch (this.sortmode) {
				case sortbyalpha:
					return "sort by alpha";
				case sortbysize:
					return "sort by size";
				case unsorted:
					return "as is unsorted";
			}
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
		// show/hide functions
		showitem(file) {
			switch (file.type) {
				case FT.dir:
					return true;
				case FT.wave:
				case FT.flac:
				case FT.mp3:
					return this.music;
				case FT.ogg:
				case FT.mp4:
				case FT.webm:
					return this.video;
				case FT.photo:
				case FT.tga:
				case FT.bmp:
				case FT.gif:
				case FT.png:
				case FT.jpeg:
				case FT.tiff:
				case FT.webp:
					return this.photo;
				case FT.pdf:
				case FT.html:
					return this.pdf;
				case FT.text:
				case FT.scr:
				case FT.cfg:
				case FT.log:
					return this.books;
				default:
					return this.other;
			}
		},
		// playlist files attributes
		getstate(file) {
			return {
				selected: this.selfile === file,
				playback: this.playbackfile && this.playbackfile.name === file.name
			};
		},
		// close current single file viewer
		closeviewer() {
			if (this.viewer) {
				this.viewer.close();
				this.viewer.visible = false;
				this.viewer = null;
			}
		},

		onshare() {
			this.$emit('share', this.selfile);
		},
		onorder() {
			this.order = -this.order;
		},
		onsortmode() {
			switch (this.sortmode) {
				case sortbyalpha:
					this.sortmode = sortbysize;
					break;
				case sortbysize:
					this.sortmode = unsorted;
					break;
				case unsorted:
					this.sortmode = sortbyalpha;
					break;
			}
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

		onmusic() {
			this.music = !this.music;
		},
		onvideo() {
			this.video = !this.video;
		},
		onphoto() {
			this.photo = !this.photo;
		},
		onpdf() {
			this.pdf = !this.pdf;
		},
		onbooks() {
			this.books = !this.books;
		},
		onother() {
			this.other = !this.other;
		},

		onselect(file) {
			this.selfile = file;

			if (!file) {
				this.closeviewer();
				return;
			}

			// Run viewer/player
			switch (FTtoFV[file.type]) {
				case FV.none:
					this.closeviewer();
					break;
				case FV.music:
					this.viewer = this.$refs.mp3player;
					this.viewer.setup(file);
					this.viewer.visible = true;
					break;
				case FV.video:
					this.closeviewer();
					break;
				case FV.image:
					this.closeviewer();
					break;
				default:
					this.closeviewer();
					break;
			}
		},
		onopen(file) {
			switch (FTtoFV[file.type]) {
				case FV.image:
					this.closeviewer();
					this.$refs.slider.popup(file);
					break;
				default:
					const url = getfileurl(file);
					window.open(url, file.name);
			}
		},
		onunselect() {
			this.onselect(null);
		},

		onplayback(file) {
			this.playbackfile = file;
		}
	}
});

Vue.component('map-card-tag', {
	template: '#map-card-tpl',
	data: function () {
		return {
			map: null, // set it on mounted event
			markers: null,
			markermode: "thumb",
			gpslist: [],

			iid: makestrid(10) // instance ID
		};
	},
	computed: {
		isvisible() {
			return this.gpslist.length > 0;
		},

		iconmarkermode() {
			switch (this.markermode) {
				case 'marker': return 'place';
				case 'thumb': return 'local_see';
			}
		},
		hintmarkermode() {
			switch (this.markermode) {
				case 'marker': return "photo positions by markers";
				case 'thumb': return "photo positions by thumbnails";
			}
		}
	},
	methods: {
		// create new opened folder
		new() {
			if (!this.markers || this.gpslist.length > 0) {
				// remove previous set
				if (this.markers) {
					this.map.removeLayer(this.markers);
				}
				// create new group
				this.markers = L.markerClusterGroup();
				// add new set
				this.map.addLayer(this.markers);
				this.gpslist = [];
			}
		},
		// setup markers on map, remove previous
		addmarkers(gpslist) {
			if (!gpslist.length) {
				return;
			}
			const size = 60;
			for (const file of gpslist) {
				const opt = {
					title: file.name,
					riseOnHover: true
				};
				if (this.markermode === 'thumb' && file.ktmb) {
					opt.icon = L.divIcon({
						html: `<img src="${filetmb(file)}">`,
						className: "photomarker",
						iconSize: [size, size],
						iconAnchor: [size / 2, size / 2],
						popupAnchor: [0, -size / 4]
					});
				}

				const template = document.createElement('template');
				template.innerHTML = makemarkercontent(file).trim();
				const popup = template.content.firstChild;
				popup.querySelector(".photoinfo img.thumb").addEventListener('click', () => {
					this.$refs.slider.popup(file);
				});

				L.marker([file.latitude, file.longitude], opt)
					.addTo(this.markers)
					.bindPopup(popup);
			}

			this.map.flyToBounds(this.markers.getBounds(), {
				padding: [20, 20]
			});

			if (!this.gpslist.length) {
				Vue.nextTick(() => {
					this.map.invalidateSize();
				});
			}
			this.gpslist.push(...gpslist);
		},

		onmarkermode() {
			switch (this.markermode) {
				case 'marker':
					this.markermode = 'thumb';
					break;
				case 'thumb':
					this.markermode = 'marker';
					break;
			}
			const gpslist = this.gpslist;
			this.new();
			this.addmarkers(gpslist);
		},
		onfitbounds() {
			this.map.flyToBounds(this.markers.getBounds(), {
				padding: [20, 20]
			});
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
