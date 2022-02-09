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

// leaflet html addons

const makemarkericon = file => {
	const res = geticonpath(file);
	const icp = res.org || res.alt;
	let src = "";
	if (file.ntmb === 1 && thumbmode) {
		src = `<source srcset="/id${appvm.aid}/thumb/${file.puid}" type="${file.mtmb}">`;
	} else {
		for (fmt of iconmapping.iconfmt) {
			src += `<source srcset="${icp + fmt.ext}" type="${fmt.mime}">`;
		}
	}
	return `
<div class="position-relative">
	<picture>
		${src}
		<img class="position-absolute top-50 start-50 translate-middle w-100">
	</picture>
</div>
`;
};

const makemarkerpopup = file => {
	const res = geticonpath(file);
	const icp = res.org || res.alt;
	let src = "";
	if (file.ntmb === 1 && thumbmode) {
		src = `<source srcset="/id${appvm.aid}/thumb/${file.puid}" type="${file.mtmb}">`;
	} else {
		for (fmt of iconmapping.iconfmt) {
			src += `<source srcset="${icp + fmt.ext}" type="${fmt.mime}">`;
		}
	}
	return `
<div class="photoinfo">
	<ul class="nav nav-tabs" role="tablist">
		<li class="nav-item"><a class="nav-link active" data-bs-toggle="tab" href="#pict">Thumbnail</a></li>
		<li class="nav-item"><a class="nav-link" data-bs-toggle="tab" href="#prop">Properties</a></li>
	</ul>
	<div class="tab-content">
		<div class="tab-pane active" id="pict">
			<picture>
				${src}
				<img class="rounded thumb" alt="${file.name}">
			</picture>
			<div class="d-flex flex-wrap latlng">
				<div><div class="name">lat:</div> <div class="value">${file.latitude.toFixed(6)}</div></div>
				<div><div class="name">lng:</div> <div class="value">${file.longitude.toFixed(6)}</div></div>
				<div><div class="name">alt:</div> <div class="value">${file.altitude ?? "N/A"}</div></div>
			</div>
		</div>
		<div class="tab-pane fade" id="prop">
			<ul class="prop p-0 m-0"><li>${filehint(file).join("</li><li>")}</li></ul>
		</div>
	</div>
</div>
`;
};

const VueDirCard = {
	template: '#dir-card-tpl',
	props: ["list"],
	data() {
		return {
			sortorder: 1,
			listmode: "smicon",
			iid: makestrid(10) // instance ID
		};
	},
	computed: {
		pathlist() {
			const l = [];
			for (const file of this.list) {
				if (file.type) {
					l.push(file);
				}
			}
			return l;
		},
		isvisible() {
			return this.pathlist.length > 0;
		},
		sortable() {
			for (const file of this.pathlist) {
				if (file.type === FT.ctgr) {
					return false;
				}
			}
			return true;
		},
		// sorted subfolders list
		sortedlist() {
			return this.sortable
				? this.pathlist.slice().sort((v1, v2) => {
					return this.sortorder * (v1.name.toLowerCase() > v2.name.toLowerCase() ? 1 : -1);
				})
				: this.pathlist;
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
			eventHub.emit('select', null);
		}
	}
};

const VueFileCard = {
	template: '#file-card-tpl',
	props: ["list"],
	data() {
		return {
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
		filelist() {
			const l = [];
			for (const file of this.list) {
				if (!file.type) {
					l.push(file);
				}
			}
			return l;
		},
		isvisible() {
			return this.filelist.length > 0;
		},
		// filtered sorted list of files
		filtlist() {
			const res = [];
			for (const file of this.filelist) {
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
			eventHub.emit('playlist', res);
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
			return !!this.filelist.find(file => extfmt.audio[pathext(file.name)]);
		},
		showvideo() {
			return !!this.filelist.find(file => extfmt.video[pathext(file.name)]);
		},
		showphoto() {
			return !!this.filelist.find(file => extfmt.image[pathext(file.name)]);
		},
		showbooks() {
			return !!this.filelist.find(file => extfmt.books[pathext(file.name)]);
		},
		showtexts() {
			return !!this.filelist.find(file => extfmt.texts[pathext(file.name)]);
		},
		showpacks() {
			return !!this.filelist.find(file => extfmt.packs[pathext(file.name)]);
		},
		showother() {
			return !!this.filelist.find(file => getFileGroup(file) === FG.other);
		},

		clsthumbmode() {
			return { active: this.thumbmode };
		},
		clsheadset() {
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
		onorder() {
			this.sortorder = -this.sortorder;
			const fl = this.filtlist; // update filtlist now
		},
		onsortalpha() {
			this.sortmode = 'byalpha';
			const fl = this.filtlist; // update filtlist now
		},
		onsortsize() {
			this.sortmode = 'bysize';
			const fl = this.filtlist; // update filtlist now
		},
		onsortunsorted() {
			this.sortmode = 'unsorted';
			const fl = this.filtlist; // update filtlist now
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
			this.thumbmode = thumbmode = !this.thumbmode;
			eventHub.emit('thumbmode', thumbmode);
		},

		onheadset() {
			eventHub.emit('audioonly', !this.audioonly);
		},
		onaudio() {
			this.fgshow[FG.audio] = !this.fgshow[FG.audio];
			const fl = this.filtlist; // update filtlist now
		},
		onvideo() {
			this.fgshow[FG.video] = !this.fgshow[FG.video];
			const fl = this.filtlist; // update filtlist now
		},
		onphoto() {
			this.fgshow[FG.image] = !this.fgshow[FG.image];
			const fl = this.filtlist; // update filtlist now
		},
		onbooks() {
			this.fgshow[FG.books] = !this.fgshow[FG.books];
			const fl = this.filtlist; // update filtlist now
		},
		ontexts() {
			this.fgshow[FG.texts] = !this.fgshow[FG.texts];
			const fl = this.filtlist; // update filtlist now
		},
		onpacks() {
			this.fgshow[FG.packs] = !this.fgshow[FG.packs];
			const fl = this.filtlist; // update filtlist now
		},
		onother() {
			this.fgshow[FG.other] = !this.fgshow[FG.other];
			const fl = this.filtlist; // update filtlist now
		},

		onunselect() {
			eventHub.emit('select', null);
		},
		onaudioonly(val) {
			this.audioonly = val;
		}
	},
	created() {
		eventHub.on('audioonly', this.onaudioonly);
	},
	unmounted() {
		eventHub.off('audioonly', this.onaudioonly);
	}
};

// set of possible sizes for horizontal images
const htiles = [
	[2, 2], // 0
	[3, 3], // 1
	[4, 4], // 2
	[5, 5], // 3
	[6, 6], // 4
];
// set of possible sizes for vertical images
const vtiles = [
	[2, 4],
	[3, 6],
];
// indexes in htiles depended on free space size
const tilemode246 = [
	[0, 2, 4], // for all
	[], // 1
	[0], // 2
	[], // 3
	[0, 2], // 4
	[], // 5
	[0, 2, 4], // 6
];
const tilemode234 = [
	[0, 1, 2], // for all
	[], // 1
	[0], // 2
	[1], // 3
	[0, 2], // 4
	[0, 1], // 5
];
const tilemode26 = [
	[0, 4], // for all
	[], // 1
	[0], // 2
	[], // 3
	[0], // 4
	[], // 5
	[0, 4], // 6
];
const tilemode2346 = [
	[0, 1, 2, 4], // for all
	[], // 1
	[0], // 2
	[1], // 3
	[0, 2], // 4
	[0, 1], // 5
	[0, 1, 2, 4], // 6
	[0, 1, 2], // 7
];
const tilemodetype = {
	'mode-246': tilemode246,
	'mode-234': tilemode234,
	'mode-26': tilemode26,
	'mode-2346': tilemode2346,
};
const rollbacktiles = 11;

const maketileslide = (list, tilemode) => {
	const tiles = [];
	const sheet = [];
	let fill = 0; // filled line
	const randomint = max => {
		return Math.floor(Math.random() * max);
	};
	const addline = () => {
		sheet.push([0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0]);
	};
	const zeropos = y => {
		for (let x = 0; x < 12; x++) {
			if (!sheet[y][x]) {
				return x;
			}
		}
		return -1;
	};
	const posfill = () => {
		while (fill < sheet.length) {
			const pos = zeropos(fill);
			if (pos >= 0) {
				return pos;
			}
			fill++;
		}
		addline();
		return 0;
	};
	const sizefill = pf => {
		for (let i = pf + 1; i < 12; i++) {
			if (sheet[fill][i]) {
				return i - pf;
			}
		}
		return 12 - pf;
	};

	let rollbackcount = 0;
	addline();
	let ti = 0;
	for (; ;) {
		for (; ti < list.length; ti++) {
			const pf = posfill();
			const sf = sizefill(pf);
			const im = tilemode[sf < tilemode.length ? sf : 0];
			let ts = htiles[im[randomint(im.length)]];
			if (!ts) {
				console.log("null:", sf, pf, sheet);
			}
			const nl = ts[1] - sheet.length + fill;
			for (let i = 0; i < nl; i++) {
				addline();
			}
			for (let x = 0; x < ts[0]; x++) {
				for (let y = 0; y < ts[1]; y++) {
					sheet[fill + y][x + pf] = Number(ti) + 1;
				}
			}

			tiles.push({
				file: list[ti],
				px: pf,
				py: fill,
				sx: ts[0],
				sy: ts[1],
			});
		}

		if (tiles.length < 6 || zeropos(sheet.length - 1) < 0) {
			break;
		}

		// rollback
		if (rollbackcount > 120) {
			console.error("rollback overflows, sheet remains with spaces");
			break;
		}
		if (tiles.length > rollbacktiles) {
			ti = tiles.length - rollbacktiles;
		} else {
			ti = 0;
		}
		// remove from sheet
		fill = tiles[ti].py;
		for (let y = fill; y < sheet.length; y++) {
			for (let x = 0; x < 12; x++) {
				if (sheet[y][x] > ti) {
					sheet[y][x] = 0;
				}
			}
		}
		// remove zero lines
		delzero: for (let y = sheet.length - 1; y >= tiles[ti].py; y--) {
			for (let x = 0; x < 12; x++) {
				if (sheet[y][x] !== 0) {
					break delzero;
				}
			}
			sheet.splice(y, 1);
		}
		// remove tiles
		tiles.splice(ti, rollbacktiles);
		rollbackcount++;
	}
	console.info("rollback count:", rollbackcount)

	return { tiles, sheet };
}

const VueTileCard = {
	template: '#tile-card-tpl',
	props: ["list"],
	data() {
		return {
			tiles: [],
			sheet: [],
			tilemode: "mode-246",
			iid: makestrid(10) // instance ID
		};
	},
	watch: {
		photolist: {
			handler(val, oldval) {
				if (val.length === oldval.length) {
					return; // list size not changed
				}
				const ret = maketileslide(val, tilemodetype[this.tilemode]);
				this.tiles = ret.tiles;
				this.sheet = ret.sheet;
			},
			deep: true
		},
		tilemode: {
			handler(val, oldval) {
				const ret = maketileslide(this.photolist, tilemodetype[val]);
				this.tiles = ret.tiles;
				this.sheet = ret.sheet;
			}
		}
	},
	computed: {
		photolist() {
			const l = [];
			for (const file of this.list) {
				if (!file.type && (file.model || file.height || file.orientation)) {
					l.push(file);
				}
			}
			return l;
		},
		isvisible() {
			return this.tiles.length > 0;
		},

		sheetbl() {
			let tbl = [];
			let row0 = [];
			for (let line = 0; line < this.sheet.length; line++) {
				const tr = [];
				const row1 = this.sheet[line];
				for (let x = 0; x < 12; x++) {
					if (row1[x] && row1[x] !== row1[x - 1] && row0[x] !== row1[x]) {
						const tile = this.tiles[row1[x] - 1];
						tr.push(tile);
					}
				}
				row0 = row1;
				tbl.push(tr);
			}
			return tbl;
		},

		hinttilemode() {
			return this.tilemode;
		},
		clsmode246() {
			return { active: this.tilemode === 'mode-246' };
		},
		clsmode234() {
			return { active: this.tilemode === 'mode-234' };
		},
		clsmode26() {
			return { active: this.tilemode === 'mode-26' };
		},
		clsmode2346() {
			return { active: this.tilemode === 'mode-2346' };
		}
	},
	methods: {
		onrebuild() {
			const ret = maketileslide(this.photolist, tilemodetype[this.tilemode]);
			this.tiles = ret.tiles;
			this.sheet = ret.sheet;
		},
		onmode246() {
			this.tilemode = 'mode-246';
		},
		onmode234() {
			this.tilemode = 'mode-234';
		},
		onmode26() {
			this.tilemode = 'mode-26';
		},
		onmode2346() {
			this.tilemode = 'mode-2346';
		},

		onunselect() {
			eventHub.emit('select', null);
		}
	}
};

const VueMapCard = {
	template: '#map-card-tpl',
	data() {
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
						html: makemarkericon(file),
						className: "photomarker",
						iconSize: [size, size],
						iconAnchor: [size / 2, size / 2],
						popupAnchor: [0, -size / 4]
					});
				}

				const template = document.createElement('template');
				template.innerHTML = makemarkerpopup(file).trim();
				const popup = template.content.firstChild;
				popup.querySelector(".photoinfo picture")?.addEventListener('click', () => {
					eventHub.emit('open', file, this.gpslist);
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
		addgpx(file) {
			const gpx = {};
			gpx.prop = file;
			gpx.trkpt = [];
			const ci = this.gpxlist.length % gpxcolors.length;
			this.gpxlist.push(gpx);

			(async () => {
				eventHub.emit('ajax', +1);
				try {
					const response = await fetch(fileurl(file));
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
					eventHub.emit('ajax', -1);
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
			if (e) {
				e.innerHTML = isFullscreen() ? 'fullscreen_exit' : 'fullscreen';
			}
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
				template.innerHTML = html?.trim();
				const tb = template.content.firstChild;

				tb?.querySelector(".leaflet-control-fullscreen")?.addEventListener('click', e => {
					this.onfullscreen();
				});
				tb?.querySelector(".leaflet-control-adjust")?.addEventListener('click', e => {
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
};

// The End.
