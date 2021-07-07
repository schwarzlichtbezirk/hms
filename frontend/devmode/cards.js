"use strict";

const sortmodeicon = {
	byalpha: 'sort_by_alpha',
	bysize: 'sort',
	unsorted: 'reorder'
};
const sortmodehint = {
	byalpha: "sort by alpha",
	bysize: "sort by size",
	unsorted: "as is unsorted"
};

const listmodetag = {
	smicon: 'file-icon-tag',
	mdicon: 'file-icon-tag',
	lgicon: 'img-icon-tag'
};
const listmoderow = {
	smicon: 'align-items-start',
	mdicon: 'align-items-start',
	lgicon: 'align-items-center'
};
const listmodeicon = {
	smicon: 'format_align_justify',
	mdicon: 'view_comfy',
	lgicon: 'view_module'
};
const listmodehint = {
	smicon: "small icons",
	mdicon: "middle icons",
	lgicon: "large icons"
};

const noderadius = 15;

const gpxcolors = [
	'#6495ED', // CornflowerBlue
	'#DA70D6', // Orchid
	'#DAA520', // GoldenRod
	'#9370DB', // MediumPurple
	'#DB7093', // PaleVioletRed
	'#8FBC8F', // DarkSeaGreen
	'#FF7F50', // Coral
	'#9ACD32', // YellowGreen
	'#BC8F8F', // RosyBrown
	'#5F9EA0', // CadetBlue
	'#778899', // LightSlateGray
	'#F4A460', // SandyBrown
	'#FA8072', // Salmon
	'#20B2AA', // LightSeaGreen
	'#4169E1'  // RoyalBlue
];

const copyTextToClipboard = text => {
	const elem = document.createElement("textarea");
	elem.value = text;
	elem.style.position = 'fixed'; // prevent to scroll content to the end
	elem.style.left = '0px';
	elem.style.top = '0px';
	document.body.appendChild(elem);
	elem.focus();
	elem.select();

	try {
		return !document.execCommand('copy') && "copying text command was failed";
	} catch (err) {
		return err;
	} finally {
		document.body.removeChild(elem);
	}
};

///////////////////////////
// Full screen functions //
///////////////////////////

const isFullscreen = () => document.fullscreenElement ||
	document.webkitFullscreenElement ||
	document.mozFullScreenElement ||
	document.msFullscreenElement;

const openFullscreen = e => {
	if (e.requestFullscreen) {
		e.requestFullscreen();
	} else if (e.mozRequestFullScreen) {
		e.mozRequestFullScreen();
	} else if (e.webkitRequestFullscreen) {
		e.webkitRequestFullscreen(Element.ALLOW_KEYBOARD_INPUT);
	} else if (e.msRequestFullscreen) {
		e.msRequestFullscreen();
	}
};

const closeFullscreen = () => {
	if (document.exitFullscreen) {
		document.exitFullscreen();
	} else if (document.mozCancelFullScreen) {
		document.mozCancelFullScreen();
	} else if (document.webkitExitFullscreen) {
		document.webkitExitFullscreen();
	} else if (document.msExitFullscreen) {
		document.msExitFullscreen();
	}
};

Vue.component('dir-card-tag', {
	template: '#dir-card-tpl',
	props: ["list"],
	data: function () {
		return {
			isauth: false, // is authorized
			sortorder: 1,
			listmode: "smicon",
			iid: makestrid(10) // instance ID
		};
	},
	computed: {
		// is it authorized or running on localhost
		isadmin() {
			return this.isauth || window.location.hostname === 'localhost' || window.location.hostname === '127.0.0.1';
		},
		isvisible() {
			return this.list.length > 0;
		},
		sortable() {
			for (const fp of this.list) {
				if (fp.type === FT.ctgr) {
					return false;
				}
			}
			return true;
		},
		// sorted subfolders list
		sortedlist() {
			return this.sortable
				? this.list.slice().sort((v1, v2) => {
					return this.sortorder * (v1.name.toLowerCase() > v2.name.toLowerCase() ? 1 : -1);
				})
				: this.list;
		},

		clsfilelist() {
			return listmoderow[this.listmode];
		},

		clsorder() {
			return this.sortorder > 0
				? 'arrow_downward'
				: 'arrow_upward';
		},
		clslistmode() {
			return listmodeicon[this.listmode];
		},

		iconmodetag() {
			return listmodetag[this.listmode];
		},

		hintorder() {
			return this.sortorder > 0
				? "direct order"
				: "reverse order";
		},
		hintlist() {
			return listmodehint[this.listmode];
		}
	},
	methods: {
		authclosure(is) {
			this.isauth = is;
		},
		onorder() {
			this.sortorder = -this.sortorder;
		},
		onlistmodesm() {
			this.listmode = 'smicon';
		},
		onlistmodemd() {
			this.listmode = 'mdicon';
		},
		onlistmodelg() {
			this.listmode = 'lgicon';
		},

		onunselect() {
			eventHub.$emit('select', null);
		}
	},
	created() {
		eventHub.$on('auth', this.authclosure);
	},
	beforeDestroy() {
		eventHub.$off('auth', this.authclosure);
	}
});

Vue.component('file-card-tag', {
	template: '#file-card-tpl',
	props: ["list"],
	data: function () {
		return {
			isauth: false, // is authorized
			sortorder: 1,
			sortmode: 'byalpha',
			listmode: 'smicon',
			thumbmode: true,
			fgshow: [
				false, // other
				true, // video
				true, // audio
				true, // image
				true, // books
				true, // texts
				true, // packs
				true // dir
			],
			audioonly: false,

			iid: makestrid(10) // instance ID
		};
	},
	computed: {
		// is it authorized or running on localhost
		isadmin() {
			return this.isauth || window.location.hostname === 'localhost' || window.location.hostname === '127.0.0.1';
		},
		isvisible() {
			return this.list.length > 0;
		},
		// filtered sorted playlist
		playlist() {
			const res = [];
			for (const file of this.list) {
				if (this.fgshow[getFileGroup(file)]) {
					res.push(file);
				}
			}
			if (this.sortmode === 'byalpha') {
				res.sort((v1, v2) => {
					return this.sortorder * (v1.name.toLowerCase() > v2.name.toLowerCase() ? 1 : -1);
				});
			} else if (this.sortmode === 'bysize') {
				res.sort((v1, v2) => {
					if (v1.size === v2.size) {
						return this.sortorder * (v1.name.toLowerCase() > v2.name.toLowerCase() ? 1 : -1);
					} else {
						return this.sortorder * (v1.size > v2.size ? 1 : -1);
					}
				});
			}
			eventHub.$emit('playlist', res);
			return res;
		},

		clsfilelist() {
			return listmoderow[this.listmode];
		},

		clsorder() {
			return this.sortorder > 0
				? 'arrow_downward'
				: 'arrow_upward';
		},
		clssortmode() {
			return sortmodeicon[this.sortmode];
		},
		clslistmode() {
			return listmodeicon[this.listmode];
		},

		showmusic() {
			return !!this.list.find(file => extfmt.audio[pathext(file.name)]);
		},
		showvideo() {
			return !!this.list.find(file => extfmt.video[pathext(file.name)]);
		},
		showphoto() {
			return !!this.list.find(file => extfmt.image[pathext(file.name)]);
		},
		showbooks() {
			return !!this.list.find(file => extfmt.books[pathext(file.name)]);
		},
		showtexts() {
			return !!this.list.find(file => extfmt.texts[pathext(file.name)]);
		},
		showpacks() {
			return !!this.list.find(file => extfmt.packs[pathext(file.name)]);
		},
		showother() {
			return !!this.list.find(file => getFileGroup(file) === FG.other);
		},

		clsthumbmode() {
			return { active: this.thumbmode };
		},
		clsaudioonly() {
			return { active: this.audioonly };
		},
		clsaudio() {
			return { active: this.fgshow[FG.audio] };
		},
		clsvideo() {
			return { active: this.fgshow[FG.video] };
		},
		clsphoto() {
			return { active: this.fgshow[FG.image] };
		},
		clsbooks() {
			return { active: this.fgshow[FG.books] };
		},
		clstexts() {
			return { active: this.fgshow[FG.texts] };
		},
		clspacks() {
			return { active: this.fgshow[FG.packs] };
		},
		clsother() {
			return { active: this.fgshow[FG.other] };
		},

		iconmodetag() {
			return listmodetag[this.listmode];
		},

		hintorder() {
			return this.sortorder > 0
				? "direct order"
				: "reverse order";
		},
		hintsortmode() {
			return sortmodehint[this.sortmode];
		},
		hintlist() {
			return listmodehint[this.listmode];
		}
	},
	methods: {
		authclosure(is) {
			this.isauth = is;
		},

		onorder() {
			this.sortorder = -this.sortorder;
			const pl = this.playlist; // update playlist now
		},
		onsortalpha() {
			this.sortmode = 'byalpha';
			const pl = this.playlist; // update playlist now
		},
		onsortsize() {
			this.sortmode = 'bysize';
			const pl = this.playlist; // update playlist now
		},
		onsortunsorted() {
			this.sortmode = 'unsorted';
			const pl = this.playlist; // update playlist now
		},
		onlistmodesm() {
			this.listmode = 'smicon';
		},
		onlistmodemd() {
			this.listmode = 'mdicon';
		},
		onlistmodelg() {
			this.listmode = 'lgicon';
		},
		onthumbmode() {
			thumbmode = this.thumbmode = !this.thumbmode;
			iconev.emit('thumb');
		},

		onaudioonly() {
			this.audioonly = !this.audioonly;
		},
		onaudio() {
			Vue.set(this.fgshow, FG.audio, !this.fgshow[FG.audio]);
			const pl = this.playlist; // update playlist now
		},
		onvideo() {
			Vue.set(this.fgshow, FG.video, !this.fgshow[FG.video]);
			const pl = this.playlist; // update playlist now
		},
		onphoto() {
			Vue.set(this.fgshow, FG.image, !this.fgshow[FG.image]);
			const pl = this.playlist; // update playlist now
		},
		onbooks() {
			Vue.set(this.fgshow, FG.books, !this.fgshow[FG.books]);
			const pl = this.playlist; // update playlist now
		},
		ontexts() {
			Vue.set(this.fgshow, FG.texts, !this.fgshow[FG.texts]);
			const pl = this.playlist; // update playlist now
		},
		onpacks() {
			Vue.set(this.fgshow, FG.packs, !this.fgshow[FG.packs]);
			const pl = this.playlist; // update playlist now
		},
		onother() {
			Vue.set(this.fgshow, FG.other, !this.fgshow[FG.other]);
			const pl = this.playlist; // update playlist now
		},

		onunselect() {
			eventHub.$emit('select', null);
		}
	},
	created() {
		eventHub.$on('auth', this.authclosure);
	},
	beforeDestroy() {
		eventHub.$off('auth', this.authclosure);
	}
});

Vue.component('map-card-tag', {
	template: '#map-card-tpl',
	data: function () {
		return {
			styleid: 'mapbox-hybrid',
			map: null, // set it on mounted event
			tiles: null,
			markers: null,
			markermode: "thumb",
			phototrack: null,
			showtrack: false,
			gpslist: [],
			gpxlist: [],

			iid: makestrid(10) // instance ID
		};
	},
	computed: {
		isvisible() {
			return this.gpslist.length > 0 || this.gpxlist.length > 0;
		},
		clsmapboxhybrid() {
			return { active: this.styleid === 'mapbox-hybrid' };
		},
		clsmapboxsatellite() {
			return { active: this.styleid === 'mapbox-satellite' };
		},
		clsmapboxoutdoors() {
			return { active: this.styleid === 'mapbox-outdoors' };
		},
		clsmapboxstreets() {
			return { active: this.styleid === 'mapbox-streets' };
		},
		clsgooglehybrid() {
			return { active: this.styleid === 'google-hybrid' };
		},
		clsgooglesatellite() {
			return { active: this.styleid === 'google-satellite' };
		},
		clsgoogleterrain() {
			return { active: this.styleid === 'google-terrain' };
		},
		clsgooglestreets() {
			return { active: this.styleid === 'google-streets' };
		},
		clsosm() {
			return { active: this.styleid === 'osm' };
		},
		clsopentopomap() {
			return { active: this.styleid === 'opentopo' };
		},
		clshikebike() {
			return { active: this.styleid === 'hikebike' };
		},
		clsphototrack() {
			return { active: this.showtrack };
		},

		iconmarkermode() {
			switch (this.markermode) {
				case 'marker': return 'place';
				case 'thumb': return 'local_see';
		}
		},
		hintlandscape() {
			switch (this.styleid) {
				case 'mapbox-hybrid':
					return "Mapbox satellite & streets map";
				case 'mapbox-satellite':
					return "Mapbox satellite map";
				case 'mapbox-outdoors':
					return "Mapbox outdoors map";
				case 'mapbox-streets':
					return "Mapbox streets map";
				case 'google-hybrid':
					return "Google maps hybrid";
				case 'google-satellite':
					return "Google maps satellite";
				case 'google-terrain':
					return "Google maps terrain";
				case 'google-streets':
					return "Google maps streets";
				case 'osm':
					return "Open Street Map";
				case 'opentopo':
					return "Open topo map";
				case 'hikebike':
					return "HikeBike map";
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
				// remove previous track
				if (this.phototrack) {
					this.map.removeLayer(this.phototrack);
					this.phototrack = null;
				}
				// new empty list
				this.gpslist = [];
			}
			// remove previous gpx-tracks
			for (const gpx of this.gpxlist) {
				if (gpx.layer) {
					this.map.removeLayer(gpx.layer);
				}
			}
			this.gpxlist = [];
		},
		// make tiles layer
		maketiles(id) {
			switch (id) {
				case 'mapbox-hybrid':
					return L.tileLayer('https://api.mapbox.com/styles/v1/{id}/tiles/{z}/{x}/{y}?access_token={accessToken}', {
						attribution: 'Map data: &copy; <a href="https://www.openstreetmap.org/copyright" target="_blank">OpenStreetMap</a> contributors, ' +
							'Imagery &copy <a href="https://www.mapbox.com/" target="_blank">Mapbox</a>',
						tileSize: 512,
						minZoom: 2,
						zoomOffset: -1,
						id: 'mapbox/satellite-streets-v11',
						accessToken: 'pk.eyJ1Ijoic2Nod2FyemxpY2h0YmV6aXJrIiwiYSI6ImNrazdseXRxZjBlemsycG8wZ3BodTR1aWUifQ.Mt99AhglX08nolRj_bsyog'
					});
				case 'mapbox-satellite':
					return L.tileLayer('https://api.mapbox.com/styles/v1/{id}/tiles/{z}/{x}/{y}?access_token={accessToken}', {
						attribution: 'Map data: &copy; <a href="https://www.openstreetmap.org/copyright" target="_blank">OpenStreetMap</a> contributors, ' +
							'Imagery &copy <a href="https://www.mapbox.com/" target="_blank">Mapbox</a>',
						tileSize: 512,
						minZoom: 2,
						zoomOffset: -1,
						id: 'mapbox/satellite-v9',
						accessToken: 'pk.eyJ1Ijoic2Nod2FyemxpY2h0YmV6aXJrIiwiYSI6ImNrazdseXRxZjBlemsycG8wZ3BodTR1aWUifQ.Mt99AhglX08nolRj_bsyog'
					});
				case 'mapbox-outdoors':
					return L.tileLayer('https://api.mapbox.com/styles/v1/{id}/tiles/{z}/{x}/{y}?access_token={accessToken}', {
						attribution: 'Map data: &copy; <a href="https://www.openstreetmap.org/copyright" target="_blank">OpenStreetMap</a> contributors, ' +
							'Imagery &copy <a href="https://www.mapbox.com/" target="_blank">Mapbox</a>',
						tileSize: 512,
						minZoom: 2,
						zoomOffset: -1,
						id: 'mapbox/outdoors-v11',
						accessToken: 'pk.eyJ1Ijoic2Nod2FyemxpY2h0YmV6aXJrIiwiYSI6ImNrazdseXRxZjBlemsycG8wZ3BodTR1aWUifQ.Mt99AhglX08nolRj_bsyog'
					});
				case 'mapbox-streets':
					return L.tileLayer('https://api.mapbox.com/styles/v1/{id}/tiles/{z}/{x}/{y}?access_token={accessToken}', {
						attribution: 'Map data: &copy; <a href="https://www.openstreetmap.org/copyright" target="_blank">OpenStreetMap</a> contributors, ' +
							'Imagery &copy <a href="https://www.mapbox.com/" target="_blank">Mapbox</a>',
						tileSize: 512,
						minZoom: 2,
						zoomOffset: -1,
						id: 'mapbox/streets-v11',
						accessToken: 'pk.eyJ1Ijoic2Nod2FyemxpY2h0YmV6aXJrIiwiYSI6ImNrazdseXRxZjBlemsycG8wZ3BodTR1aWUifQ.Mt99AhglX08nolRj_bsyog'
					});
				case 'google-hybrid':
					return L.tileLayer('http://{s}.google.com/vt/lyrs=s,h&x={x}&y={y}&z={z}', {
						subdomains: ['mt0', 'mt1', 'mt2', 'mt3'],
						attribution: 'Map data: &copy; <a href="https://developers.google.com/maps/documentation/javascript/overview" target="_blank">Google Maps Platform</a> contributors',
						minZoom: 2,
						maxZoom: 20
					});
				case 'google-satellite':
					return L.tileLayer('http://{s}.google.com/vt/lyrs=s&x={x}&y={y}&z={z}', {
						subdomains: ['mt0', 'mt1', 'mt2', 'mt3'],
						attribution: 'Map data: &copy; <a href="https://developers.google.com/maps/documentation/javascript/overview" target="_blank">Google Maps Platform</a> contributors',
						minZoom: 2,
						maxZoom: 20
					});
				case 'google-terrain':
					return L.tileLayer('http://{s}.google.com/vt/lyrs=p&x={x}&y={y}&z={z}', {
						subdomains: ['mt0', 'mt1', 'mt2', 'mt3'],
						attribution: 'Map data: &copy; <a href="https://developers.google.com/maps/documentation/javascript/overview" target="_blank">Google Maps Platform</a> contributors',
						minZoom: 2,
						maxZoom: 20
					});
				case 'google-streets':
					return L.tileLayer('http://{s}.google.com/vt/lyrs=m&x={x}&y={y}&z={z}', {
						subdomains: ['mt0', 'mt1', 'mt2', 'mt3'],
						attribution: 'Map data: &copy; <a href="https://developers.google.com/maps/documentation/javascript/overview" target="_blank">Google Maps Platform</a> contributors',
						minZoom: 2,
						maxZoom: 20
					});
				case 'osm':
					return L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
						maxZoom: 19,
						attribution: 'Map data: &copy; <a href="https://www.openstreetmap.org/copyright" target="_blank">OpenStreetMap</a> contributors'
					});
				case 'opentopo':
					return L.tileLayer('https://{s}.tile.opentopomap.org/{z}/{x}/{y}.png', {
						attribution: 'Map data: &copy; <a href="https://www.openstreetmap.org/copyright" target="_blank">OpenStreetMap</a> contributors, ' +
							'<a href="http://viewfinderpanoramas.org" target="_blank">SRTM</a> | Map style: &copy; <a href="https://opentopomap.org" target="_blank">OpenTopoMap</a>',
						minZoom: 2,
						maxZoom: 17
					});
				case 'hikebike':
					return L.tileLayer('https://tiles.wmflabs.org/hikebike/{z}/{x}/{y}.png', {
						attribution: 'Map data: &copy; <a href="https://www.openstreetmap.org/copyright" target="_blank">OpenStreetMap</a> contributors',
						minZoom: 2,
						maxZoom: 19
					});
			}
		},
		// change tiles layer
		changetiles(id) {
			this.map.removeLayer(this.tiles);
			this.styleid = id;
			this.tiles = this.maketiles(id);
			this.map.addLayer(this.tiles);
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
				if (this.markermode === 'thumb' && file.puid) {
					opt.icon = L.divIcon({
						html: `<img src="${filetmbimg(file)}" class="position-absolute top-50 start-50 translate-middle w-100">`,
						className: "photomarker",
						iconSize: [size, size],
						iconAnchor: [size / 2, size / 2],
						popupAnchor: [0, -size / 4]
					});
				}

				const template = document.createElement('template');
				template.innerHTML = makemarkercontent(file).trim();
				const popup = template.content.firstChild;
				popup.querySelector(".photoinfo picture").addEventListener('click', () => {
					this.$root.$refs.slider.popup(file, this.gpslist);
				});

				L.marker([file.latitude, file.longitude], opt)
					.addTo(this.markers)
					.bindPopup(popup);
			}

			const mustsize = !this.gpslist.length;
			Vue.nextTick(() => {
				if (mustsize) {
					this.map.invalidateSize();
				}
				this.onfitbounds();
			});

			this.gpslist.push(...gpslist);
			if (this.showtrack) {
				this.makephototrack();
			}
		},
		// produces reduced track polyline
		makephototrack() {
			let last = L.latLng(this.gpslist[0].latitude, this.gpslist[0].longitude, this.gpslist[0].altitude);
			let prev = last;
			let route = 0, trk = 0, asc = 0;
			const latlngs = [last];
			for (const file of this.gpslist) {
				const p = L.latLng(file.latitude, file.longitude, file.altitude);
				const d = last.distanceTo(p);
				if (d > noderadius) {
					route += d;
					if (p.alt > last.alt) {
						asc += p.alt - last.alt;
					}
					latlngs.push(p);
					last = p;
				}
				trk += prev.distanceTo(p);
				prev = p;
			}
			if (this.phototrack) {
				this.map.removeLayer(this.phototrack);
			}
			this.phototrack = L.polyline(latlngs, { color: '#3CB371' }) // MediumSeaGreen
				.bindPopup(`total <b>${latlngs.length}</b> waypoints<br>route <b>${route.toFixed()}</b> m<br>track <b>${trk.toFixed()}</b> m<br>ascent <b>${asc.toFixed()}</b> m`)
				.addTo(this.map);
		},
		// add GPX track polyline
		addgpx(fp) {
			const gpx = {};
			gpx.prop = fp;
			gpx.trkpt = [];
			const ci = this.gpxlist.length % gpxcolors.length;
			this.gpxlist.push(gpx);

			(async () => {
				eventHub.$emit('ajax', +1);
				try {
					const response = await fetch(fileurl(fp));
					const body = await response.text();
					const re = /lat="(\d+\.\d+)" lon="(\d+\.\d+)"/g;
					const matches = body.matchAll(re);
					let prev = null;
					let route = 0;
					for (const m of matches) {
						const p = L.latLng(Number(m[1]), Number(m[2]));
						if (prev) {
							route += p.distanceTo(prev);
						}
						prev = p;
						gpx.trkpt.push(p);
					}
					gpx.layer = L.polyline(gpx.trkpt, { color: gpxcolors[ci] })
						.bindPopup(`points <b>${gpx.trkpt.length}</b><br>route <b>${route.toFixed()}</b> m`)
						.addTo(this.map);
				} catch (e) {
					ajaxfail(e);
				} finally {
					eventHub.$emit('ajax', -1);
				}
			})();
		},

		onmapboxhybrid() {
			this.changetiles('mapbox-hybrid');
		},
		onmapboxsatellite() {
			this.changetiles('mapbox-satellite');
		},
		onmapboxoutdoors() {
			this.changetiles('mapbox-outdoors');
		},
		onmapboxstreets() {
			this.changetiles('mapbox-streets');
		},
		ongooglehybrid() {
			this.changetiles('google-hybrid');
		},
		ongooglesatellite() {
			this.changetiles('google-satellite');
		},
		ongoogleterrain() {
			this.changetiles('google-terrain');
		},
		ongooglestreets() {
			this.changetiles('google-streets');
		},
		onosm() {
			this.changetiles('osm');
		},
		onopentopo() {
			this.changetiles('opentopo');
		},
		onhikebike() {
			this.changetiles('hikebike');
		},
		onphototrack() {
			this.showtrack = !this.showtrack;
			if (this.showtrack) {
				this.makephototrack();
			} else {
				this.map.removeLayer(this.phototrack);
				this.phototrack = null;
			}
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
		onfullscreenchange() {
			const e = this.$refs.map.querySelector(".leaflet-control-fullscreen > span");
			e.innerHTML = isFullscreen() ? 'fullscreen_exit' : 'fullscreen';
		},
		onfullscreen() {
			if (isFullscreen()) {
				closeFullscreen();
			} else {
				openFullscreen(this.$refs.map);
			}
		},
		onfitbounds() {
			const bounds = this.markers.getBounds();
			for (const gpx of this.gpxlist) {
				if (gpx.layer) {
					bounds.extend(gpx.layer.getBounds());
				}
			}
			this.map.flyToBounds(bounds, {
				padding: [20, 20]
			});
		}
	},
	mounted() {
		this.tiles = this.maketiles('mapbox-hybrid');
		this.map = L.map(this.$refs.map, {
			attributionControl: true,
			zoomControl: false,
			center: [0, 0],
			zoom: 8,
			layers: [this.tiles]
		});
		L.control.scale().addTo(this.map);
		L.control.zoom({
			zoomInText: '<span class="material-icons">add</span>',
			zoomOutText: '<span class="material-icons">remove</span>'
		}).addTo(this.map);
		// create fullscreen button
		L.Control.Fullscreen = L.Control.extend({
			onAdd: map => {
				const html = document.getElementById("leaflet-toolbar").innerHTML;
				const template = document.createElement('template');
				template.innerHTML = html.trim();
				const tb = template.content.firstChild;

				const butfs = tb.querySelector(".leaflet-control-fullscreen");
				butfs.addEventListener('click', e => {
					this.onfullscreen();
				});

				const butfb = tb.querySelector(".leaflet-control-adjust");
				butfb.addEventListener('click', e => {
					this.onfitbounds();
				});

				return tb;
			},
			onRemove: map => {
			}
		});
		L.control.fullscreen = opts => {
			return new L.Control.Fullscreen(opts);
		};
		L.control.fullscreen({ position: 'topright' }).addTo(this.map);
	}
});

// The End.
