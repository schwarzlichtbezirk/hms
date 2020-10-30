"use strict";

const sortbyalpha = "name";
const sortbysize = "size";
const unsorted = "";

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
const listmodenext = {
	smicon: 'mdicon',
	mdicon: 'lgicon',
	lgicon: 'smicon'
};

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

Vue.component('dir-card-tag', {
	template: '#dir-card-tpl',
	props: ["list", "shared"],
	data: function () {
		return {
			isauth: false, // is authorized
			selfile: null, // current selected item
			order: 1,
			listmode: "smicon",
			diskpath: "", // path to disk to add
			diskpathstate: 0,
			iid: makestrid(10) // instance ID
		};
	},
	computed: {
		// is it authorized or running on localhost
		isadmin() {
			return this.isauth || window.location.hostname === 'localhost' || window.location.hostname === '127.0.0.1';
		},
		isvisible() {
			(() => {
				if (this.list.find(file => file === this.selfile)) {
					return;
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
			return listmoderow[this.listmode];
		},

		dislink() {
			return !this.selfile;
		},
		disshared() {
			return !this.selfile;
		},
		clsshared() {
			return { active: this.selfile && this.isshared(this.selfile) };
		},
		disdiskadd() {
			return !this.diskpath.length;
		},
		disdiskremove() {
			return !this.selfile || this.selfile.type !== FT.drive;
		},

		clsorder() {
			return this.order > 0
				? 'arrow_downward'
				: 'arrow_upward';
		},
		clslistmode() {
			return listmodeicon[this.listmode];
		},
		clsdiskpathedt() {
			return !this.diskpathstate ? ''
				: this.passstate === -1 ? 'is-invalid' : 'is-valid';
		},

		iconmodetag() {
			return listmodetag[this.listmode];
		},

		hintorder() {
			return this.order > 0
				? "direct order"
				: "reverse order";
		},
		hintlist() {
			return listmodehint[this.listmode];
		}
	},
	methods: {
		isshared(file) {
			for (const shr of this.shared) {
				if (shr.puid === file.puid) {
					return true;
				}
			}
			return false;
		},
		// playlist files attributes
		getstate(file) {
			return {
				selected: this.selfile === file,
				playback: false,
				shared: this.isshared(file)
			};
		},
		onlink() {
			copyTextToClipboard(window.location.origin + pathurl(this.selfile));
		},
		onshare() {
			this.$emit('share', this.selfile);
		},
		onorder() {
			this.order = -this.order;
		},
		onlistmode() {
			this.listmode = listmodenext[this.listmode];
		},

		ondiskadd() {
			ajaxcc.emit('ajax', +1);
			fetchjsonauth("POST", "/api/drive/add", {
				aid: app.aid,
				path: this.diskpath
			}).then(response => {
				traceajax(response);
				if (response.ok) {
					const file = response.data;
					if (file) {
						this.list.push(file);
					}
					$("#diskadd" + this.iid).modal('hide');
				} else {
					this.diskpathstate = -1;
				}
			}).catch(ajaxfail).finally(() => ajaxcc.emit('ajax', -1));
		},
		ondiskremove() {
			ajaxcc.emit('ajax', +1);
			fetchjsonauth("POST", "/api/drive/del", {
				aid: app.aid,
				puid: this.selfile.puid
			}).then(response => {
				traceajax(response);
				if (response.ok) {
					if (response.data) {
						this.list.splice(this.list.findIndex(elem => elem === this.selfile), 1);
						if (this.isshared(this.selfile)) {
							this.fetchsharedel(this.selfile);
						}
					}
				}
			}).catch(ajaxfail).finally(() => ajaxcc.emit('ajax', -1));
		},
		ondiskpathchange(e) {
			fetchjsonauth("POST", "/api/path/is", {
				aid: app.aid,
				path: this.diskpath
			}).then(response => {
				if (response.ok) {
					this.diskpathstate = response.data ? 1 : 0;
				} else {
					this.diskpathstate = -1;
				}
			}).catch(ajaxfail);
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
	},
	mounted() {
		this._authclosure = is => this.isauth = is;
		auth.on('auth', this._authclosure);
		$('#diskadd' + this.iid).on('shown.bs.modal', function () {
			$(this).find('input').trigger('focus');
		});
	},
	beforeDestroy() {
		auth.off('auth', this._authclosure);
		$('#diskadd' + this.iid).off('shown.bs.modal');
	}
});

Vue.component('file-card-tag', {
	template: '#file-card-tpl',
	props: ["list", "shared"],
	data: function () {
		return {
			isauth: false, // is authorized
			selfile: null, // current selected item
			playbackfile: null,
			order: 1,
			sortmode: sortbyalpha,
			listmode: "smicon",
			music: true, video: true, image: true, books: true, texts: true, other: false,
			audioonly: false,
			viewer: null, // file viewers

			iid: makestrid(10) // instance ID
		};
	},
	computed: {
		// is it authorized or running on localhost
		isadmin() {
			return this.isauth || window.location.hostname === 'localhost' || window.location.hostname === '127.0.0.1';
		},
		isvisible() {
			(() => {
				if (this.list.find(file => file === this.selfile)) {
					return;
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
			return listmoderow[this.listmode];
		},

		dislink() {
			return !this.selfile;
		},
		disshared() {
			return !this.selfile;
		},
		clsshared() {
			return { active: this.selfile && this.isshared(this.selfile) };
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
			return listmodeicon[this.listmode];
		},

		showmusic() {
			return !!this.list.find(file => FTtoFG[file.type] === FG.music);
		},
		showvideo() {
			return !!this.list.find(file => FTtoFG[file.type] === FG.video);
		},
		showphoto() {
			return !!this.list.find(file => FTtoFG[file.type] === FG.image);
		},
		showbooks() {
			return !!this.list.find(file => FTtoFG[file.type] === FG.books);
		},
		showtexts() {
			return !!this.list.find(file => FTtoFG[file.type] === FG.texts);
		},
		showother() {
			return !!this.list.find(file => !file.type
				|| FTtoFG[file.type] === FG.store
				|| FTtoFG[file.type] === FG.other);
		},

		clsaudio() {
			return { active: this.audioonly };
		},
		clsmusic() {
			return { active: this.music };
		},
		clsvideo() {
			return { active: this.video };
		},
		clsphoto() {
			return { active: this.image };
		},
		clsbooks() {
			return { active: this.books };
		},
		clstexts() {
			return { active: this.texts };
		},
		clsother() {
			return { active: this.other };
		},

		iconmodetag() {
			return listmodetag[this.listmode];
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
			return listmodehint[this.listmode];
		}
	},
	methods: {
		// show/hide functions
		showitem(file) {
			switch (FTtoFG[file.type]) {
				case FG.dir:
					return true;
				case FG.music:
					return this.music;
				case FG.video:
					return this.video;
				case FG.image:
					return this.image;
				case FG.books:
					return this.books;
				case FG.texts:
					return this.texts;
				default:
					return this.other;
			}
		},
		isshared(file) {
			for (const shr of this.shared) {
				if (shr.puid === file.puid) {
					return true;
				}
			}
			return false;
		},
		// playlist files attributes
		getstate(file) {
			return {
				selected: this.selfile === file,
				playback: this.playbackfile && this.playbackfile.name === file.name,
				shared: this.isshared(file)
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

		onlink() {
			copyTextToClipboard(window.location.origin + fileurl(this.selfile));
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
			this.listmode = listmodenext[this.listmode];
		},

		onaudio() {
			this.audioonly = !this.audioonly;
		},
		onmusic() {
			this.music = !this.music;
		},
		onvideo() {
			this.video = !this.video;
		},
		onphoto() {
			this.image = !this.image;
		},
		onbooks() {
			this.books = !this.books;
		},
		ontexts() {
			this.texts = !this.texts;
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
					if (this.audioonly) {
						this.closeviewer();
					} else {
						this.viewer = this.$refs.mp3player;
						this.viewer.setup(file);
						this.viewer.visible = true;
					}
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
					const url = mediaurl(file);
					window.open(url, file.name);
			}
		},
		onunselect() {
			this.onselect(null);
		},

		onplayback(file) {
			this.playbackfile = file;
		}
	},
	mounted() {
		this._authclosure = is => this.isauth = is;
		auth.on('auth', this._authclosure);
	},
	beforeDestroy() {
		auth.off('auth', this._authclosure);
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
				if (this.markermode === 'thumb' && file.puid) {
					opt.icon = L.divIcon({
						html: `<img src="${filetmbpng(file)}">`,
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
					this.$refs.slider.popup(file);
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
				this.map.flyToBounds(this.markers.getBounds(), {
					padding: [20, 20]
				});
			});
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
