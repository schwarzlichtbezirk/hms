"use strict";

const isMainImage = ext => ({
	".tga": true, ".bmp": true, ".dib": true, ".rle": true, ".dds": true,
	".tif": true, ".tiff": true, ".dng": true, ".jpg": true, ".jpe": true, ".jpeg": true, ".jfif": true,
	".gif": true, ".png": true, ".avif": true, ".webp": true, ".psd": true, ".psb": true
})[ext];

const isMainAudio = ext => ({
	".wav": true, ".flac": true, ".mp3": true, ".ogg": true, ".opus": true,
	".acc": true, ".m4a": true, ".alac": true
})[ext];

const isMainVideo = ext => ({
	".mp4": true, ".webm": true
})[ext];

const imagefilter = file => file.type === FT.file && file.size && isMainImage(pathext(file.name));
const audiofilter = file => file.type === FT.file && file.size && isMainAudio(pathext(file.name));
const videofilter = file => file.type === FT.file && file.size && isMainVideo(pathext(file.name));

const filehint = file => {
	const lst = [];
	// Std properties
	lst.push(['name', file.name.length > 31 ? file.name.substring(0, 32) + "..." : file.name]);
	if (file.type === FT.file) {
		lst.push(['size', fmtitemsize(file.size ?? 0)]);
	}
	if (file.time > "0001-01-01T00:00:00Z") {
		lst.push(['time', (new Date(file.time)).toLocaleString('en-GB')]);
	}
	// Dir properties
	if (file.fgrp) {
		if (file.fgrp.group) {
			lst.push(['directories', file.fgrp.group]);
		}
		if (file.fgrp.video) {
			lst.push(['video', file.fgrp.video]);
		}
		if (file.fgrp.audio) {
			lst.push(['audio', file.fgrp.audio]);
		}
		if (file.fgrp.image) {
			lst.push(['image', file.fgrp.image]);
		}
		if (file.fgrp.books) {
			lst.push(['books', file.fgrp.video]);
		}
		if (file.fgrp.texts) {
			lst.push(['texts', file.fgrp.texts]);
		}
		if (file.fgrp.packs) {
			lst.push(['packs', file.fgrp.packs]);
		}
		if (file.fgrp.other) {
			lst.push(['other', file.fgrp.other]);
		}
	}
	return lst;
};

const fileinfo = file => {
	const lst = [];
	// Std properties
	if (file.type === FT.file) {
		lst.push(['size', fmtitemsize(file.size ?? 0)]);
	}
	if (file.time) {
		lst.push(['time', (new Date(file.time)).toLocaleString('en-GB')]);
	}
	// MP3 tags properties
	if (file.title) {
		lst.push(['title', file.title]);
	}
	if (file.album) {
		lst.push(['album', file.album]);
	}
	if (file.artist) {
		lst.push(['artist', file.artist]);
	}
	if (file.composer) {
		lst.push(['composer', file.composer]);
	}
	if (file.genre) {
		lst.push(['genre', file.genre]);
	}
	if (file.year) {
		lst.push(['year', file.year]);
	}
	if (file.tracknum ?? file.tracksum) {
		if (file.tracksum) {
			lst.push(['track', `${file.tracknum}/${file.tracksum}`]);
		} else {
			lst.push(['track', file.tracknum ?? file.tracksum]);
		}
	}
	if (file.discnum ?? file.discsum) {
		if (file.discsum) {
			lst.push(['disc', `${file.discnum}/${file.discsum}`]);
		} else {
			lst.push(['disc', file.discnum ?? file.discsum]);
		}
	}
	if (file.comment) {
		lst.push(['comment', file.comment.substring(0, 80)]);
	}
	// EXIF tags properties
	if (file.width && file.height) {
		lst.push(['resolution', `${file.width}x${file.height}`]);
	}
	if (file.model) {
		lst.push(['model', file.model]);
	}
	if (file.make) {
		lst.push(['manufacturer', file.make]);
	}
	/*if (file.software) {
		lst.push(['software', file.software]);
	}*/
	if (file.datetime) {
		lst.push(['photo taken', (new Date(file.datetime)).toLocaleString('en-GB')]);
	}
	switch (file.orientation) {
		case 1:
			lst.push(['orientation', 'normal']);
			break;
		case 2:
			lst.push(['orientation', 'horizontal reversed']);
			break;
		case 3:
			lst.push(['orientation', 'flipped']);
			break;
		case 4:
			lst.push(['orientation', 'flipped & horizontal reversed']);
			break;
		case 5:
			lst.push(['orientation', 'clockwise turned & horizontal reversed']);
			break;
		case 6:
			lst.push(['orientation', 'clockwise turned']);
			break;
		case 7:
			lst.push(['orientation', 'anticlockwise turned & horizontal reversed']);
			break;
		case 8:
			lst.push(['orientation', 'anticlockwise turned']);
			break;
	}
	switch (file.exposureprog) {
		case 1:
			lst.push(['exposure program', 'manual']);
			break;
		case 2:
			lst.push(['exposure program', 'normal program']);
			break;
		case 3:
			lst.push(['exposure program', 'aperture priority']);
			break;
		case 4:
			lst.push(['exposure program', 'shutter priority']);
			break;
		case 5:
			lst.push(['exposure program', 'creative program (depth of field)']);
			break;
		case 6:
			lst.push(['exposure program', 'action program (fast shutter speed)']);
			break;
		case 7:
			lst.push(['exposure program', 'portrait mode (background out of focus)']);
			break;
		case 8:
			lst.push(['exposure program', 'landscape mode (background in focus)']);
			break;
	}
	if (file.exposuretime) {
		lst.push(['exposure time', `${file.exposuretime} sec`]);
	}
	if (file.fnumber) {
		lst.push(['F-number', `f/${file.fnumber}`]);
	}
	if (file.isospeed) {
		lst.push(['ISO speed rating', `ISO-${file.isospeed}`]);
	}
	if (file.shutterspeed) {
		lst.push(['shutter speed', file.shutterspeed]);
	}
	if (file.aperture) {
		lst.push(['lens aperture', file.aperture]);
	}
	if (file.exposurebias) {
		lst.push(['exposure bias', file.exposurebias]);
	}
	if (file.lightsource) {
		switch (file.lightsource) {
			case 1:
				lst.push(['light source', 'daylight']);
				break;
			case 2:
				lst.push(['light source', 'fluorescent']);
				break;
			case 3:
				lst.push(['light source', 'tungsten (incandescent light)']);
				break;
			case 4:
				lst.push(['light source', 'flash']);
				break;
			case 9:
				lst.push(['light source', 'fine weather']);
				break;
			case 10:
				lst.push(['light source', 'cloudy weather']);
				break;
			case 11:
				lst.push(['light source', 'shade']);
				break;
			case 12:
				lst.push(['light source', 'daylight fluorescent (D 5700-7100K)']);
				break;
			case 13:
				lst.push(['light source', 'day white fluorescent (N 4600-5700K)']);
				break;
			case 14:
				lst.push(['light source', 'cool white fluorescent (W 3800-4600K)']);
				break;
			case 15:
				lst.push(['light source', 'white fluorescent (WW 3250-3800K)']);
				break;
			case 16:
				lst.push(['light source', 'warm white fluorescent (L 2600-3250K)']);
				break;
			case 17:
				lst.push(['light source', 'standard light A']);
				break;
			case 18:
				lst.push(['light source', 'standard light B']);
				break;
			case 19:
				lst.push(['light source', 'standard light C']);
				break;
			case 20:
				lst.push(['light source', 'D55']);
				break;
			case 21:
				lst.push(['light source', 'D65']);
				break;
			case 22:
				lst.push(['light source', 'D75']);
				break;
			case 23:
				lst.push(['light source', 'D50']);
				break;
			case 24:
				lst.push(['light source', 'ISO studio tungsten']);
				break;
			case 255:
				lst.push(['light source', 'other light source']);
				break;
			default:
				lst.push(['light source', `light code #${file.lightsource}`]);
				break;
		}
	}
	if (file.focal) {
		lst.push(['focal length', `${file.focal} mm`]);
	}
	if (file.focal35mm) {
		lst.push(['35mm equivalent focal length', `${file.focal35mm} mm`]);
	}
	if (file.digitalzoom) {
		lst.push(['digital zoom ratio', file.digitalzoom]);
	}
	if (file.flash) {
		if (file.flash & 0x01) {
			lst.push(['flash', 'fired']);
		} else {
			lst.push(['flash', 'did not fire']);
		}
		if ((file.flash & 0x06) === 0x04) {
			lst.push(['strobe return light', 'not detected']);
		} else if ((file.flash & 0x06) === 0x06) {
			lst.push(['strobe return light', 'detected']);
		}
		if ((file.flash & 0x18) === 0x08) {
			lst.push(['flash mode', 'compulsory flash firing']);
		} else if ((file.flash & 0x18) === 0x10) {
			lst.push(['flash mode', 'compulsory flash suppression']);
		} else if ((file.flash & 0x18) === 0x18) {
			lst.push(['flash mode', 'auto']);
		}
		if (file.flash & 0x40) {
			lst.push(['flash', 'red-eye reduction supported']);
		}
	}
	if (file.uniqueid) {
		lst.push(['unique ID', file.uniqueid]);
	}
	if (file.thumbjpeglen) {
		lst.push(['thumbnail length', file.thumbjpeglen]);
	}
	return lst;
};

const iconpixsize = {
	xs: 'icon-pix-xs imgscale',
	sm: 'icon-pix-sm imgscale',
	md: 'icon-pix-md imgscale',
	lg: 'icon-pix-lg imgscale',
};

const iconsvgsize = {
	xs: 'icon-svg-xs imgcontain',
	sm: 'icon-svg-sm imgcontain',
	md: 'icon-svg-md imgcontain',
	lg: 'icon-svg-lg imgcontain',
};

const iconwdh = {
	sm: 'icon-wdh-sm',
	md: 'icon-wdh-md',
	lg: 'icon-wdh-lg',
};

const VueIcon = {
	template: '#icon-tpl',
	props: ["file", "size"],
	data() {
		return {
			im: [],
			tm: true
		};
	},
	computed: {
		fmtalt() {
			return pathext(this.file.name);
		},
		clsicon() {
			for (const fmt of this.im.iconfmt) {
				if (fmt.mime === 'image/svg+xml') {
					return iconsvgsize[this.size];
				} else {
					return iconpixsize[this.size];
				}
			}
		},
		ismtmb() {
			return Number(this.file.mtmb) > 0 && this.tm;
		},
		isetmb() {
			return Number(this.file.etmb) > 0 && this.tm;
		},
		iconsrc() {
			const res = geticonpath(this.im, this.file);
			return res.org || res.alt;
		},
		mtmbsrc() {
			return `/id${this.$root.aid}/mtmb/${this.file.puid}`;
		},
		etmbsrc() {
			return `/id${this.$root.aid}/etmb/${this.file.puid}`;
		},
		mtmbmime() {
			return MimeStr[this.file.mtmb];
		},
		etmbmime() {
			return MimeStr[this.file.etmb];
		}
	},
	methods: {
		_iconset(im) {
			this.im = im;
		},
		_thumbmode(tm) {
			this.tm = tm;
		}
	},
	created() {
		this.im = iconmapping;
		this.tm = thumbmode;
	},
	mounted() {
		eventHub.on('iconset', this._iconset);
		eventHub.on('thumbmode', this._thumbmode);
	},
	unmounted() {
		eventHub.off('iconset', this._iconset);
		eventHub.off('thumbmode', this._thumbmode);
	}
};

const VueIconMenu = {
	template: '#iconmenu-tpl',
	props: ["file"],
	data() {
		return {
			popover: null,
			scanned: false,
		};
	},
	computed: {
		clsshared() {
			return {
				'active': this.file.shared,
			};
		},
		showinfo() {
			return this.file.type === FT.file;
		},
		shownewtab() {
			if (this.file.type !== FT.file) {
				return false;
			}
			const ext = pathext(this.file.name);
			return extfmt.image[ext] || extfmt.video[ext]
				|| extfmt.books[ext] || extfmt.texts[ext] || extfmt.playlist[ext]
				|| (extfmt.audio[ext] && this.file.etmb);
		},
		showcopy() {
			return this.file.type === FT.file || this.file.type === FT.dir;
		},
		showcutdel() {
			return !this.file.static;
		}
	},
	methods: {
		scan() {
			(async () => {
				try {
					const response = await fetchjsonauth("POST", `/id${this.$root.aid}/api/res/tags`, {
						puid: this.file.puid
					});
					const data = await response.json();
					traceajax(response, data);
					if (response.ok) {
						extend(this.file, data.prop);
						this.popover.setContent({
							'.popover-header': this.file.name,
							'.popover-body': fileinfo(this.file).map(e => `<b>${e[0]}</b>: ${e[1]}`).join('<br>')
						})
					} else {
						throw new HttpError(response.status, data);
					}
				} catch (e) {
				}
			})();
		},
		oninfo() {
			if (!this.scanned) {
				this.scanned = true;
				this.scan();
			}
		},
		onnewtab() {
			const ext = pathext(this.file.name);
			if (extfmt.image[ext] || extfmt.video[ext] || extfmt.books[ext] || extfmt.texts[ext] || extfmt.playlist[ext]) {
				const url = `/id${this.$root.aid}/file/${this.file.puid}?media=1&hd=0`;
				window.open(url, this.file.name);
			} else if (extfmt.audio[ext] && this.file.etmb) {
				const url = `/id${this.$root.aid}/etmb/${this.file.puid}`;
				window.open(url, this.file.name);
			}
		},
		onlink() {
			copyTextToClipboard(window.location.origin + `/id${this.$root.aid}/file/${this.file.puid}`);
		},
		onshare() {
			(async () => {
				eventHub.emit('ajax', +1);
				try {
					if (this.file.shared) { // should remove share
						await this.$root.fetchsharedel(this.file);
					} else { // should add share
						await this.$root.fetchshareadd(this.file);
					}
				} catch (e) {
					ajaxfail(e);
				} finally {
					eventHub.emit('ajax', -1);
				}
			})();
		},
		oncopy() {
			this.$root.copied = this.file;
			this.$root.cuted = null;
		},
		oncut() {
			this.$root.cuted = this.file;
			this.$root.copied = null;
		},
		ondelask() {
			this.$root.delfile = this.file;
			if (this.$root.delensured) {
				this.$root.ondelete();
			} else {
				const dlg = new bootstrap.Modal('#delask');
				dlg.show();
			}
		}
	},
	mounted() {
		if (this.file.type === FT.file) {
			this.popover = new bootstrap.Popover(this.$refs.info, {
				title: this.file.name,
				content: fileinfo(this.file).map(e => `<b>${e[0]}</b>: ${e[1]}`).join('<br>'),
				html: true
			});
		}
	}
};

const VueListItem = {
	template: '#list-item-tpl',
	props: ["file"],
	data() {
		return {
			im: [],
			tm: true
		};
	},
	computed: {
		fmttitle() {
			return filehint(this.file).map(e => `${e[0]}: ${e[1]}`).join('\n');
		},
		label() {
			if (Number(this.file.mtmb) > 0 && this.tm
				|| !geticonpath(this.im, this.file).org) {
				return this.file.shared
					? this.im.shared.label
					: this.im.private.label;
			}
		},
		clsiconwdh() {
			return iconwdh['xs'];
		},
		clsicon() {
			for (const fmt of this.im.iconfmt) {
				if (fmt.mime === 'image/svg+xml') {
					return iconsvgsize['xs'];
				} else {
					return iconpixsize['xs'];
				}
			}
		},
		clsiconsvg() {
			return iconsvgsize['xs'];
		},
		clsiconpix() {
			return iconpixsize['xs'];
		},

		// manage items classes
		isfile() {
			return this.file.type === FT.file;
		},
		itemview() {
			return { 'selected': this.file.selected };
		},
		filesize() {
			return fmtfilesize(this.file.size ?? 0);
		},
		filedate() {
			return (new Date(this.file.time)).toLocaleDateString();
		},
		filetime() {
			return (new Date(this.file.time)).toLocaleTimeString('en-GB');
		}
	},
	methods: {
		_iconset(im) {
			this.im = im;
		},
		_thumbmode(tm) {
			this.tm = tm;
		}
	},
	created() {
		this.im = iconmapping;
		this.tm = thumbmode;
	},
	mounted() {
		eventHub.on('iconset', this._iconset);
		eventHub.on('thumbmode', this._thumbmode);
	},
	unmounted() {
		eventHub.off('iconset', this._iconset);
		eventHub.off('thumbmode', this._thumbmode);
	}
};

const VueFileItem = {
	template: '#file-item-tpl',
	props: ["file", "size"],
	data() {
		return {
			im: [],
			tm: true
		};
	},
	computed: {
		fmttitle() {
			return filehint(this.file).map(e => `${e[0]}: ${e[1]}`).join('\n');
		},
		label() {
			if (Number(this.file.mtmb) > 0 && this.tm
				|| !geticonpath(this.im, this.file).org) {
				return this.file.shared
					? this.im.shared.label
					: this.im.private.label;
			}
		},
		clsiconwdh() {
			return iconwdh[this.size];
		},
		clsicon() {
			for (const fmt of this.im.iconfmt) {
				if (fmt.mime === 'image/svg+xml') {
					return iconsvgsize[this.size];
				} else {
					return iconpixsize[this.size];
				}
			}
		},
		clsiconsvg() {
			return iconsvgsize[this.size];
		},
		clsiconpix() {
			return iconpixsize[this.size];
		},

		// manage items classes
		itemview() {
			return { 'selected': this.file.selected };
		}
	},
	methods: {
		_iconset(im) {
			this.im = im;
		},
		_thumbmode(tm) {
			this.tm = tm;
		}
	},
	created() {
		this.im = iconmapping;
		this.tm = thumbmode;
	},
	mounted() {
		eventHub.on('iconset', this._iconset);
		eventHub.on('thumbmode', this._thumbmode);
	},
	unmounted() {
		eventHub.off('iconset', this._iconset);
		eventHub.off('thumbmode', this._thumbmode);
	}
};

const VueImgItem = {
	template: '#img-item-tpl',
	props: ["file"],
	data() {
		return {
			im: [],
			tm: true
		};
	},
	computed: {
		fmttitle() {
			return filehint(this.file).map(e => `${e[0]}: ${e[1]}`).join('\n');
		},
		label() {
			if (Number(this.file.mtmb) > 0 && this.tm
				|| !geticonpath(this.im, this.file).org) {
				return this.file.shared
					? this.im.shared.label
					: this.im.private.label;
			}
		},

		// manage items classes
		itemview() {
			return { 'selected': this.file.selected };
		}
	},
	methods: {
		_iconset(im) {
			this.im = im;
		},
		_thumbmode(tm) {
			this.tm = tm;
		}
	},
	created() {
		this.im = iconmapping;
		this.tm = thumbmode;
	},
	mounted() {
		eventHub.on('iconset', this._iconset);
		eventHub.on('thumbmode', this._thumbmode);
	},
	unmounted() {
		eventHub.off('iconset', this._iconset);
		eventHub.off('thumbmode', this._thumbmode);
	}
};

// media max-width multiplier
let wdhmult = 0;
(() => {
	const mqlsm = window.matchMedia('(width <= 576px)');
	const mqlmd = window.matchMedia('(576px < width <= 854px)');
	const mqllg = window.matchMedia('(854px < width <= 1280px)');
	const mqlhd = window.matchMedia('(width > 1280px)');
	const hsm = e => {
		if (e.matches) {
			wdhmult = 2;
			eventHub.emit('wdhmult', wdhmult);
		}
	};
	const hmd = e => {
		if (e.matches) {
			wdhmult = 3;
			eventHub.emit('wdhmult', wdhmult);
		}
	};
	const hlg = e => {
		if (e.matches) {
			wdhmult = 4;
			eventHub.emit('wdhmult', wdhmult);
		}
	};
	const hhd = e => {
		if (e.matches) {
			wdhmult = 6;
			eventHub.emit('wdhmult', wdhmult);
		}
	};
	hsm(mqlsm);
	hmd(mqlmd);
	hlg(mqllg);
	hhd(mqlhd);
	mqlsm.addEventListener('change', hsm);
	mqlmd.addEventListener('change', hmd);
	mqllg.addEventListener('change', hlg);
	mqlhd.addEventListener('change', hhd);
})();

const VueTileItem = {
	template: '#tile-item-tpl',
	props: ["file", "sx", "sy"],
	data() {
		return {
			wdhmult: wdhmult,
			im: [],
			tm: true
		};
	},
	computed: {
		fmtalt() {
			return pathext(this.file.name);
		},
		istile() {
			const fld = 'mt' + (this.wdhmult * this.sx < 10 ? '0' : '') + this.wdhmult * this.sx;
			return Number(this.file[fld]) > 0;
		},
		fmttitle() {
			return filehint(this.file).map(e => `${e[0]}: ${e[1]}`).join('\n');
		},
		iconsrc() {
			return `/id${this.$root.aid}/tile/${this.file.puid}/${24 * this.wdhmult * this.sx}x${18 * this.wdhmult * this.sy}`;
		},
		iconblank() {
			return `/fs/assets/blank-tile/${24 * this.wdhmult * this.sx}x${18 * this.wdhmult * this.sy}.svg`;
		}
	},
	methods: {
		_iconset(im) {
			this.im = im;
		},
		_thumbmode(tm) {
			this.tm = tm;
		},
		_wdhmult(m) {
			this.wdhmult = m;
		}
	},
	created() {
		this.im = iconmapping;
		this.tm = thumbmode;
	},
	mounted() {
		eventHub.on('iconset', this._iconset);
		eventHub.on('thumbmode', this._thumbmode);
		eventHub.on('wdhmult', this._wdhmult);
	},
	unmounted() {
		eventHub.off('iconset', this._iconset);
		eventHub.off('thumbmode', this._thumbmode);
		eventHub.off('wdhmult', this._wdhmult);
	}
};

// The End.
