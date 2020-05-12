"use strict";

const filetmb = file => file.ntmb === 1
	? '/thumb/' + file.ktmb
	: '/asst/file-png/' + geticonname(file) + '.png';

const makemarkercontent = file => `
<div class="photoinfo">
	<ul class="nav nav-tabs" role="tablist">
		<li class="nav-item"><a class="nav-link active" data-toggle="tab" href="#pict">Thumbnail</a></li>
		<li class="nav-item"><a class="nav-link" data-toggle="tab" href="#prop">Properties</a></li>
	</ul>
	<div class="tab-content">
		<div class="tab-pane active" id="pict">
			<img class="rounded thumb" src="${'/thumb/' + file.ktmb}" alt="${file.name} ">
			<div class="d-flex flex-wrap latlng">
				<div><div class="name">lat:</div> <div class="value">${file.latitude.toFixed(6)}</div></div>
				<div><div class="name">lng:</div> <div class="value">${file.longitude.toFixed(6)}</div></div>
				<div><div class="name">alt:</div> <div class="value">${file.altitude || "N/A"}</div></div>
			</div>
		</div>
		<div class="tab-pane fade" id="prop">
			<ul class="prop p-0 m-0"><li>${filehint(file).join("</li><li>")}</li></ul>
		</div>
	</div>
</div>
`;

const filehint = file => {
	const lst = [];
	lst.push(file.name);
	// Std properties
	if (file.pref) {
		lst.push('share: ' + shareprefix + file.pref);
	}
	if (FTtoFG[file.type] !== FG.dir) {
		lst.push('size: ' + fmtitemsize(file.size || 0));
	}
	if (file.time) {
		lst.push('time: ' + (new Date(file.time)).toLocaleString());
	}
	// MP3 tags properties
	if (file.title) {
		lst.push('title: ' + file.title);
	}
	if (file.album) {
		lst.push('album: ' + file.album);
	}
	if (file.artist) {
		lst.push('artist: ' + file.artist);
	}
	if (file.composer) {
		lst.push('composer: ' + file.composer);
	}
	if (file.genre) {
		lst.push('genre: ' + file.genre);
	}
	if (file.year) {
		lst.push('year: ' + file.year);
	}
	if (file.track && (file.track.number || file.track.total)) {
		lst.push(`track: ${file.track.number || ''}/${file.track.total || ''}`);
	}
	if (file.disc && (file.disc.number || file.disc.total)) {
		lst.push(`disc: ${file.disc.number || ''}/${file.disc.total || ''}`);
	}
	if (file.comment) {
		lst.push('comment: ' + file.comment.substring(0, 80));
	}
	// EXIF tags properties
	if (file.width && file.height) {
		lst.push(`dimension: ${file.width}x${file.height}`);
	}
	if (file.model) {
		lst.push(`model: ${file.model}`);
	}
	if (file.make) {
		lst.push(`manufacturer: ${file.make}`);
	}
	/*if (file.software) {
		lst.push(`software: ${file.software}`);
	}*/
	if (file.datetime) {
		lst.push(`photo taken: ${(new Date(file.datetime)).toLocaleString()}`);
	}
	if (file.orientation) {
		switch (file.orientation) {
			case 1:
				lst.push(`orientation: normal`);
				break;
			case 2:
				lst.push(`orientation: horizontal reversed`);
				break;
			case 3:
				lst.push(`orientation: flipped`);
				break;
			case 4:
				lst.push(`orientation: flipped & horizontal reversed`);
				break;
			case 5:
				lst.push(`orientation: clockwise turned & horizontal reversed`);
				break;
			case 6:
				lst.push(`orientation: clockwise turned`);
				break;
			case 7:
				lst.push(`orientation: anticlockwise turned & horizontal reversed`);
				break;
			case 8:
				lst.push(`orientation: anticlockwise turned`);
				break;
		}
	}
	if (file.exposureprog) {
		switch (file.exposureprog) {
			case 1:
				lst.push(`exposure program: manual`);
				break;
			case 2:
				lst.push(`exposure program: normal program`);
				break;
			case 3:
				lst.push(`exposure program: aperture priority`);
				break;
			case 4:
				lst.push(`exposure program: shutter priority`);
				break;
			case 5:
				lst.push(`exposure program: creative program (depth of field)`);
				break;
			case 6:
				lst.push(`exposure program: action program (fast shutter speed)`);
				break;
			case 7:
				lst.push(`exposure program: portrait mode (background out of focus)`);
				break;
			case 8:
				lst.push(`exposure program: landscape mode (background in focus)`);
				break;
		}
	}
	if (file.exposuretime) {
		lst.push(`exposure time: ${file.exposuretime} sec`);
	}
	if (file.fnumber) {
		lst.push(`F-number: f/${file.fnumber}`);
	}
	if (file.isospeed) {
		lst.push(`ISO speed rating: ISO-${file.isospeed}`);
	}
	if (file.shutterspeed) {
		lst.push(`shutter speed: ${file.shutterspeed}`);
	}
	if (file.aperture) {
		lst.push(`lens aperture: ${file.aperture}`);
	}
	if (file.exposurebias) {
		lst.push(`exposure bias: ${file.exposurebias}`);
	}
	if (file.lightsource) {
		switch (file.lightsource) {
			case 1:
				lst.push(`light source: daylight`);
				break;
			case 2:
				lst.push(`light source: fluorescent`);
				break;
			case 3:
				lst.push(`light source: tungsten (incandescent light)`);
				break;
			case 4:
				lst.push(`light source: flash`);
				break;
			case 9:
				lst.push(`light source: fine weather`);
				break;
			case 10:
				lst.push(`light source: cloudy weather`);
				break;
			case 11:
				lst.push(`light source: shade`);
				break;
			case 12:
				lst.push(`light source: daylight fluorescent (D 5700-7100K)`);
				break;
			case 13:
				lst.push(`light source: day white fluorescent (N 4600-5700K)`);
				break;
			case 14:
				lst.push(`light source: cool white fluorescent (W 3800-4600K)`);
				break;
			case 15:
				lst.push(`light source: white fluorescent (WW 3250-3800K)`);
				break;
			case 16:
				lst.push(`light source: warm white fluorescent (L 2600-3250K)`);
				break;
			case 17:
				lst.push(`light source: standard light A`);
				break;
			case 18:
				lst.push(`light source: standard light B`);
				break;
			case 19:
				lst.push(`light source: standard light C`);
				break;
			case 20:
				lst.push(`light source: D55`);
				break;
			case 21:
				lst.push(`light source: D65`);
				break;
			case 22:
				lst.push(`light source: D75`);
				break;
			case 23:
				lst.push(`light source: D50`);
				break;
			case 24:
				lst.push(`light source: ISO studio tungsten`);
				break;
			case 255:
				lst.push(`light source: other light source`);
				break;
			default:
				lst.push(`light source: light code #${file.lightsource}`);
				break;
		}
	}
	if (file.focal) {
		lst.push(`focal length: ${file.focal} mm`);
	}
	if (file.focal35mm) {
		lst.push(`35mm equivalent focal length: ${file.focal35mm} mm`);
	}
	if (file.digitalzoom) {
		lst.push(`digital zoom ratio: ${file.digitalzoom}`);
	}
	if (file.flash) {
		if (file.flash & 0x01) {
			lst.push(`flash fired`);
		} else {
			lst.push(`flash did not fire`);
		}
		if ((file.flash & 0x06) === 0x04) {
			lst.push("strobe return light: not detected");
		} else if ((file.flash & 0x06) === 0x06) {
			lst.push("strobe return light: detected");
		}
		if ((file.flash & 0x18) === 0x08) {
			lst.push("flash mode: compulsory flash firing");
		} else if ((file.flash & 0x18) === 0x10) {
			lst.push("flash mode: compulsory flash suppression");
		} else if ((file.flash & 0x18) === 0x18) {
			lst.push("flash mode: auto");
		}
		if (file.flash & 0x40) {
			lst.push("red-eye reduction supported");
		}
	}
	if (file.uniqueid) {
		lst.push(`unique ID: ${file.uniqueid}`);
	}
	return lst;
};

Vue.component('file-icon-tag', {
	template: '#file-icon-tpl',
	props: ["file", "state"],
	computed: {
		fmttitle() {
			return filehint(this.file).join('\n');
		},

		webpicon() {
			return this.file.ntmb === 1
				? ''
				: '/asst/file-webp/' + geticonname(this.file) + '.webp';
		},

		pngicon() {
			return this.file.ntmb === 1
				? '/thumb/' + this.file.ktmb
				: '/asst/file-png/' + geticonname(this.file) + '.png';
		},

		// manage items classes
		itemview() {
			return { selected: this.state.selected };
		}
	},
	methods: {
		onselect() {
			this.$emit('select', this.file);
		},

		onopen() {
			this.$emit('open', this.file);
		}
	}
});

Vue.component('img-icon-tag', {
	template: '#img-icon-tpl',
	props: ["file", "state"],
	computed: {
		fmttitle() {
			return filehint(this.file).join('\n');
		},

		webpicon() {
			return this.file.ntmb === 1
				? ''
				: '/asst/file-webp/' + geticonname(this.file) + '.webp';
		},

		pngicon() {
			return this.file.ntmb === 1
				? '/thumb/' + this.file.ktmb
				: '/asst/file-png/' + geticonname(this.file) + '.png';
		},

		// manage items classes
		itemview() {
			return { selected: this.state.selected };
		}
	},
	methods: {
		onselect() {
			this.$emit('select', this.file);
		},

		onopen() {
			this.$emit('open', this.file);
		}
	}
});

// The End.
