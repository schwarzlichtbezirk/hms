"use strict";

let resmodel = { skinlist: [], iconlist: [] };
let iconmapping = {
	private: {
		blank: "",
		label: "",
		cid: { cid: "" },
		drive: {},
		folder: {},
		grp: {},
		ext: {}
	},
	shared: {
		blank: "",
		label: "",
		cid: { cid: "" },
		drive: {},
		folder: {},
		grp: {},
		ext: {}
	},
	iconfmt: []
};
let thumbmode = true;

// Predefined PUIDs.
const PUID = {
	home: "04",
	drives: "08",
	shares: "0C",
	media: "0G",
	video: "0K",
	audio: "0O",
	image: "0S",
	books: "10",
	texts: "14",
	map: "18",

	reserved: 32
};

// Category identifiers.
const CID = {
	"04": "home",
	"08": "drives",
	"0C": "shares",
	"0G": "media",
	"0K": "video",
	"0O": "audio",
	"0S": "image",
	"10": "books",
	"14": "texts",
	"18": "map",
};

// Category properties.
const CP = {
	"04": "Home",
	"08": "Drives list",
	"0C": "Shared resources",
	"0G": "Multimedia files",
	"0K": "Movie and video files",
	"0O": "Music and audio files",
	"0S": "Photos and images",
	"10": "Books",
	"14": "Text files",
	"18": "Map",
};

// MIME enum values.
const Mime = {
	dis: -1,
	nil: 0,
	unk: 1,
	gif: 2,
	png: 3,
	jpeg: 4,
	webp: 5,
};

// MIME type string by value.
const MimeStr = {
	[Mime.nil]: "",
	[Mime.unk]: "image/*",
	[Mime.gif]: "image/gif",
	[Mime.png]: "image/png",
	[Mime.jpeg]: "image/jpeg",
	[Mime.webp]: "image/webp",
};

// MIME type value by string.
const MimeVal = {
	"image/*": Mime.unk,
	"image/gif": Mime.gif,
	"image/png": Mime.png,
	"image/jpeg": Mime.jpeg,
	"image/webp": Mime.webp,
};

// File types
const FT = {
	file: 0,
	dir: 1,
	drv: 2,
	ctgr: 3
};

// File groups
const FG = {
	other: 0,
	video: 1,
	audio: 2,
	image: 3,
	books: 4,
	texts: 5,
	packs: 6,
	dir: 7
};

// Drive state
const DS = {
	yellow: 3000,
	red: 10000
};

const extfmt = {
	"bitmap": {
		".tga": 1, ".bmp": 1, ".dib": 1, ".rle": 1, ".dds": 1
	},
	"tiff": {
		".tiff": 1, ".tif": 1
	},
	"jpeg": {
		".jpg": 1, ".jpe": 1, ".jpeg": 1, ".jfif": 1
	},
	"jpeg2000": {
		".jp2": 1, ".jpg2": 1, ".jpx": 1, ".jpm": 1
	},
	"psd": {
		".psd": 1, ".psb": 1
	},

	"component": {
		".dll": 1, ".ocx": 1
	},

	"exec": {
		".exe": 1, ".dll": 1, ".ocx": 1, ".bat": 1, ".cmd": 1, ".sh": 1
	},

	"text": {
		".txt": 1, ".md": 1
	},
	"html": {
		".html": 1, ".htm": 1, ".shtml": 1, ".shtm": 1,
		".xhtml": 1, ".phtml": 1, ".hta": 1, ".mht": 1
	},
	"config": {
		".cfg": 1, ".ini": 1, ".inf": 1, ".reg": 1
	},
	"datafmt": {
		".xml": 1, ".xsml": 1, ".xsl": 1, ".xsd": 1,
		".kml": 1, ".gpx": 1,
		".wsdl": 1, ".xlf": 1, ".xliff": 1,
		".yml": 1, ".yaml": 1, ".json": 1
	},
	"script": {
		".css": 1,
		".js": 1, ".jsm": 1, ".vb": 1, ".vbs": 1, ".bat": 1, ".cmd": 1, ".sh": 1,
		".mak": 1, ".iss": 1, ".nsi": 1, ".nsh": 1, ".bsh": 1, ".sql": 1,
		".as": 1, ".mx": 1, ".ps": 1, ".php": 1, ".phpt": 1, ".lua": 1, ".tcl": 1, ".rc": 1, ".cmake": 1
	},
	"code": {
		".css": 1,
		".js": 1, ".jsm": 1, ".vb": 1, ".vbs": 1, ".bat": 1, ".cmd": 1, ".sh": 1,
		".mak": 1, ".iss": 1, ".nsi": 1, ".nsh": 1, ".bsh": 1, ".sql": 1,
		".as": 1, ".mx": 1, ".ps": 1, ".php": 1, ".phpt": 1, ".lua": 1, ".tcl": 1, ".rc": 1, ".cmake": 1,
		".java": 1, ".jsp": 1, ".asp": 1,
		".asm": 1, ".c": 1, ".h": 1, ".hpp": 1, ".hxx": 1, ".cpp": 1, ".cxx": 1, ".cc": 1, ".cs": 1,
		".go": 1, ".r": 1, ".d": 1, ".pas": 1, ".inc": 1,
		".py": 1, ".pyw": 1, ".pl": 1, ".pm": 1, ".plx": 1, ".rb": 1, ".rbw": 1
	},

	"msoffice": {
		".doc": 1, ".docx": 1, ".xls": 1, ".xlsx": 1, ".ppt": 1, ".pptx": 1, ".vsd": 1
	},
	"openoffice": {
		".odt": 1, ".ods": 1, ".odp": 1, ".rtf": 1, ".abw": 1
	},
	"office": {
		".odt": 1, ".ods": 1, ".odp": 1, ".rtf": 1, ".abw": 1,
		".doc": 1, ".docx": 1, ".xls": 1, ".xlsx": 1, ".ppt": 1, ".pptx": 1, ".vsd": 1
	},

	"archive": {
		".cab": 1, ".zip": 1, ".7z": 1, ".rar": 1, ".rev": 1,
		".jar": 1, ".apk": 1,
		".tar": 1, ".tgz": 1, ".gz": 1, ".bz2": 1
	},
	"disk": {
		".iso": 1, ".isz": 1, ".udf": 1, ".nrg": 1, ".mdf": 1, ".mdx": 1,
		".img": 1, ".ima": 1, ".imz": 1, ".ccd": 1, ".vc4": 1, ".dmg": 1,
		".daa": 1, ".uif": 1, ".vhd": 1, ".vhdx": 1, ".vmdk": 1
	},
	"package": {
		".wpk": 1
	},
	"playlist": {
		".m3u": 1, ".m3u8": 1, ".wpl": 1, ".pls": 1, ".asx": 1, ".xspf": 1
	},

	"image": {
		".tga": 1, ".bmp": 1, ".dib": 1, ".rle": 1, ".dds": 1,
		".tif": 1, ".tiff": 1, ".jpg": 1, ".jpe": 1, ".jpeg": 1, ".jfif": 1,
		".gif": 1, ".png": 1, ".webp": 1, ".psd": 1, ".psb": 1,
		".jp2": 1, ".jpg2": 1, ".jpx": 1, ".jpm": 1, ".jxr": 1
	},
	"audio": {
		".aac": 1, ".m4a": 1, ".alac": 1, ".aif": 1, ".mpa": 1, ".mp3": 1,
		".wav": 1, ".wma": 1, ".weba": 1, ".oga": 1, ".ogg": 1, ".opus": 1,
		".flac": 1, ".mka": 1, ".ra": 1, ".mid": 1, ".midi": 1, ".cda": 1
	},
	"video": {
		".avi": 1, ".mpe": 1, ".mpg": 1, ".mp4": 1, ".webm": 1, ".wmv": 1, ".wmx": 1,
		".flv": 1, ".3gp": 1, ".3g2": 1, ".mkv": 1, ".mov": 1, ".ogv": 1, ".ogx": 1
	},
	"books": {
		".pdf": 1, ".djvu": 1, ".djv": 1,
		".html": 1, ".htm": 1, ".shtml": 1, ".shtm": 1,
		".xhtml": 1, ".phtml": 1, ".hta": 1, ".mht": 1
	},
	"texts": {
		".txt": 1, ".md": 1,
		".css": 1,
		".js": 1, ".jsm": 1, ".vb": 1, ".vbs": 1, ".bat": 1, ".cmd": 1, ".sh": 1,
		".mak": 1, ".iss": 1, ".nsi": 1, ".nsh": 1, ".bsh": 1, ".sql": 1,
		".as": 1, ".mx": 1, ".ps": 1, ".php": 1, ".phpt": 1, ".lua": 1, ".tcl": 1, ".rc": 1, ".cmake": 1,
		".java": 1, ".jsp": 1, ".asp": 1,
		".asm": 1, ".c": 1, ".h": 1, ".hpp": 1, ".hxx": 1, ".cpp": 1, ".cxx": 1, ".cc": 1, ".cs": 1,
		".go": 1, ".r": 1, ".d": 1, ".pas": 1, ".inc": 1,
		".py": 1, ".pyw": 1, ".pl": 1, ".pm": 1, ".plx": 1, ".rb": 1, ".rbw": 1,
		".cfg": 1, ".ini": 1, ".inf": 1, ".reg": 1,
		".xml": 1, ".xsml": 1, ".xsl": 1, ".xsd": 1,
		".kml": 1, ".gpx": 1,
		".wsdl": 1, ".xlf": 1, ".xliff": 1,
		".yml": 1, ".yaml": 1, ".json": 1
	},
	"packs": {
		".cab": 1, ".zip": 1, ".7z": 1, ".rar": 1, ".rev": 1,
		".jar": 1, ".apk": 1,
		".tar": 1, ".tgz": 1, ".gz": 1, ".bz2": 1,
		".iso": 1, ".isz": 1, ".udf": 1, ".nrg": 1, ".mdf": 1, ".mdx": 1,
		".img": 1, ".ima": 1, ".imz": 1, ".ccd": 1, ".vc4": 1, ".dmg": 1,
		".daa": 1, ".uif": 1, ".vhd": 1, ".vhdx": 1, ".vmdk": 1,
		".wpk": 1,
		".m3u": 1, ".m3u8": 1, ".wpl": 1, ".pls": 1, ".asx": 1, ".xspf": 1
	}
};

const extfmtorder = [
	"bitmap", "tiff", "jpeg", "jpeg2000", "psd",
	"component", "exec",
	"text", "html", "config", "datafmt", "script", "code",
	"msoffice", "openoffice", "office",
	"archive", "disk", "package", "playlist",
	"image", "audio", "video", "books", "texts", "packs"
];

const getFileGroup = file => {
	if (file.type) {
		return FG.dir;
	}
	const ext = pathext(file.name);
	if (extfmt.image[ext]) return FG.image;
	else if (extfmt.audio[ext]) return FG.audio;
	else if (extfmt.video[ext]) return FG.video;
	else if (extfmt.books[ext]) return FG.books;
	else if (extfmt.texts[ext]) return FG.texts;
	else if (extfmt.packs[ext]) return FG.packs;
	else return FG.other;
};

const geticonpath = file => {
	const org = file.shared ? iconmapping.shared : iconmapping.private;
	const alt = file.shared ? iconmapping.private : iconmapping.shared;
	switch (file.type) {
		case FT.ctgr:
			return {
				org: org.cid[CID[file.puid]] || org.cid.cid,
				alt: alt.cid[CID[file.puid]] || alt.cid.cid
			};
		case FT.drv:
			if (file.latency < 0) {
				return { org: org.drive.offline, alt: alt.drive.offline };
			} else if (!file.latency || file.latency < DS.yellow) {
				return { org: org.drive.green, alt: alt.drive.green };
			} else if (file.latency < DS.red) {
				return { org: org.drive.yellow, alt: alt.drive.yellow };
			} else {
				return { org: org.drive.red, alt: alt.drive.red };
			}
		case FT.dir:
			if (file.scan) {
				let fnum = 0;
				const fg = file.fgrp;
				for (const n of fg) {
					fnum += n;
				}
				if (!fnum) {
					return { org: org.folder.open, alt: alt.folder.open };
				} else if (fg[FG.other] / fnum > 0.5) {
					return { org: org.folder.other, alt: alt.folder.other };
				} else if (fg[FG.video] / fnum > 0.5) {
					return { org: org.folder.video, alt: alt.folder.video };
				} else if (fg[FG.audio] / fnum > 0.5) {
					return { org: org.folder.audio, alt: alt.folder.audio };
				} else if (fg[FG.image] / fnum > 0.5) {
					return { org: org.folder.image, alt: alt.folder.image };
				} else if (fg[FG.books] / fnum > 0.5) {
					return { org: org.folder.books, alt: alt.folder.books };
				} else if (fg[FG.texts] / fnum > 0.5) {
					return { org: org.folder.texts, alt: alt.folder.texts };
				} else if (fg[FG.packs] / fnum > 0.5) {
					return { org: org.folder.packs, alt: alt.folder.packs };
				} else if (fg[FG.dir] / fnum > 0.5) {
					return { org: org.folder.dir, alt: alt.folder.dir };
				} else if ((fg[FG.audio] + fg[FG.video] + fg[FG.image]) / fnum > 0.5) {
					return { org: org.folder.media, alt: alt.folder.media };
				} else {
					return { org: org.folder.open, alt: alt.folder.open };
				}
			} else {
				return { org: org.folder.close, alt: alt.folder.close };
			}
		default: // file types
			const ext = pathext(file.name);
			const find = t => {
				for (const k of extfmtorder) {
					const icon = t[k];
					if (icon && extfmt[k][ext]) {
						return icon;
					}
				}
			};
			return {
				org: org.ext[ext] || find(org.grp) || org.blank,
				alt: alt.ext[ext] || find(alt.grp) || alt.blank
			};
	}
};

const encode = uri => encodeURI(uri).replace('#', '%23').replace('&', '%26').replace('+', '%2B');

const fileurl = file => `/id${appvm.aid}/file/${file.puid}`;
const mediaurl = (file, media, hd) => `/id${appvm.aid}/file/${file.puid}?media=${media}&hd=${hd}`;

const showmsgbox = (title, message, details) => {
	const el = document.getElementById('msgbox');
	if (el) {
		const dlg = new bootstrap.Modal(el);
		el.querySelector(".modal-title").innerText = title;
		el.querySelector(".message").innerText = message;
		el.querySelector(".details").innerText = details ?? "";
		dlg.show();
	}
};

const ajaxfail = e => {
	console.error(e.name, e);
	if (e instanceof SyntaxError) {
		showmsgbox(
			"Syntax error",
			"Application function failed with syntax error in javascript."
		);
		return;
	} else if (e instanceof HttpError) {
		const msgbox = (title, message) => {
			const el = document.getElementById('msgbox');
			if (el) {
				const dlg = new bootstrap.Modal(el);
				el.querySelector(".modal-title").innerText = title;
				el.querySelector(".message").innerText = message;
				el.querySelector(".errcode").innerText = e.code;
				el.querySelector(".errmsg").innerText = e.what;
				dlg.show();
			}
		};
		switch (e.status) {
			case 400: // Bad Request
				msgbox(
					"Application error",
					"Action is rejected by server. This error is caused by wrong parameters in application ajax-call to server."
				);
				return;
			case 401: // Unauthorized
				msgbox(
					"401 Unauthorized",
					"Action can be done only after authorization."
				);
				return;
			case 403: // Forbidden
				msgbox(
					"403 resource forbidden",
					"Resource referenced by application ajax-call is forbidden. It can be accessible after authorization, or for other authorization."
				);
				return;
			case 404: // Not Found
				msgbox(
					"404 resource not found",
					"Resource referenced by application ajax-call is not found on server."
				);
				return;
			case 500: // Internal Server Error
				msgbox(
					"Internal server error",
					"Action could not be completed due to an internal error on the server side."
				);
				return;
			default:
				msgbox(
					"Error " + e.status,
					`Action is rejected with HTTP status ${e.status}.`
				);
		}
	} else {
		showmsgbox(
			"Server unavailable",
			"Server is currently not available, action can not be done now."
		);
	}
};

const VueMainApp = {
	template: '#app-tpl',
	data() {
		return {
			skinid: "", // ID of skin CSS
			iconid: "", // ID of icons mapping json
			resmodel: resmodel,
			showauth: false, // display authorization form
			isauth: false, // is authorized
			authid: 0, // authorized ID
			aid: 0, // profile ID
			ishome: false, // able to go home

			selfile: null, // current selected item
			diskpath: "", // path to disk to add
			diskpathstate: 0,
			diskadd: null,

			shared: [], // list of shared items

			// history
			histpos: 0, // position in history stack
			histlist: [], // history stack

			// current opened folder data
			flist: [], // list of files and subfolders in in current folder as is
			curscan: new Date(), // time of last scanning of current folder
			curpuid: "", // current folder PUID
			curpath: "", // current folder path and path state
			shrname: "", // current folder path share name
			copied: null, // copied item
			cuted: null, // cuted item

			iid: makestrid(10) // instance ID
		};
	},
	computed: {
		// is it authorized or running on localhost
		isadmin() {
			return this.isauth || window.location.hostname === 'localhost' || window.location.hostname === '127.0.0.1';
		},
		// current page URL
		curshorturl() {
			return `${(devmode ? "/dev" : "")}/id${this.aid}/path/${this.curpuid}`;
		},
		// current page URL
		curlongurl() {
			return `${(devmode ? "/dev" : "")}/id${this.aid}/path/${this.curpath}`;
		},
		// current path base name
		curbasename() {
			if (CP[this.curpuid]) {
				return CP[this.curpuid];
			} else if (this.curpath) {
				const arr = this.curpath.split('/');
				const base = arr.pop() || arr.pop();
				return !arr.length && this.shrname || base;
			} else {
				return "unknown";
			}
		},
		// array of paths to current folder
		curpathway() {
			if (!this.curpath) {
				return [];
			}

			const arr = this.curpath.split('/');
			// remove empty element from separator at the end
			// and remove current name
			if (!arr.pop()) {
				arr.pop();
			}

			const lst = [];
			let path = '';
			for (const fn of arr) {
				if (path) {
					path += '/' + fn;
				} else {
					path = fn;
				}
				lst.push({
					name: fn,
					path: path,
					type: FT.dir
				});
			}
			if (lst.length) {
				lst[0].name = this.shrname || lst[0].name;
			}
			return lst;
		},

		// not cached files
		uncached() {
			const lst = [];
			for (const file of this.flist) {
				if (!file.type && !file.mtmb) {
					lst.push(file);
				}
			}
			return lst;
		},

		// number of subfolders
		pathcount() {
			let n = 0
			for (const file of this.flist) {
				if (file.type === FT.dir) {
					n++;
				}
			}
			return n;
		},
		// number of files
		filecount() {
			let n = 0
			for (const file of this.flist) {
				if (!file.type) {
					n++;
				}
			}
			return n;
		},
		// files sum size
		sumsize() {
			let ss = 0;
			for (const file of this.flist) {
				ss += file.size ?? 0;
			}
			return fmtitemsize(ss);
		},

		// common buttons enablers

		clshome() {
			return { 'disabled': this.curpuid === PUID.home || !(this.isadmin || this.ishome) };
		},
		clsback() {
			return { 'disabled': this.histpos < 2 };
		},
		clsforward() {
			return { 'disabled': this.histpos >= this.histlist.length };
		},
		clsparent() {
			return { 'disabled': !this.curpathway.length };
		},

		clslink() {
			return { 'disabled': !this.selfile || this.selfile.type === FT.ctgr };
		},
		clsshared() {
			return {
				'active': this.selfile && this.selfile.shared,
				'disabled': !this.selfile
			};
		},

		showdiskadd() {
			return this.isadmin && this.curpuid === PUID.drives;
		},
		clsdiskpathedt() {
			return {
				'is-invalid': this.diskpathstate === -1,
				'is-valid': this.diskpathstate == 1
			};
		},
		clsdiskadd() {
			return { 'disabled': !this.diskpath.length };
		},
		clsdiskremove() {
			return { 'disabled': !this.selfile || this.selfile.type !== FT.drv };
		},

		textauthcaret() {
			return this.showauth ? 'arrow_right' : 'arrow_left';
		},

		showcopypaste() {
			return !!this.curpath && this.isadmin;
		},
		showpastego() {
		},
		clscopy() {
			return { 'disabled': !this.selfile };
		},
		clspaste() {
			const sel = this.copied ?? this.cuted;
			return {
				'disabled': !sel || (() => {
					for (const file of this.flist) {
						if (file.puid === sel.puid) {
							return true;
						}
					}
					return false;
				})()
			};
		},
		clspastego() {
			const sel = this.copied ?? this.cuted;
			return {
				'disabled': !sel || !(() => {
					for (const file of this.flist) {
						if (file.name === sel.name) {
							return true;
						}
					}
					return false;
				})()
			};
		},
		clscut() {
			return { 'disabled': !this.selfile };
		},
		clsdelete() {
			return { 'disabled': !this.selfile };
		},
		hintpaste() {
			return `paste: ${this.copied?.name ?? this.cuted?.name}`;
		},
		hintpastego() {
			return `paste new: ${this.copied?.name ?? this.cuted?.name}`;
		},

		// buttons hints
		hintback() {
			if (this.histpos > 1) {
				const hist = this.histlist[this.histpos - 2];
				if (CP[hist.puid]) {
					return `back to "${CP[hist.puid]}"`;
				} else if (hist.path) {
					return `back to /id${hist.aid}/path/${hist.path}`;
				} else {
					return "back to home";
				}
			}
			return "go back";
		},
		hintforward() {
			if (this.histpos < this.histlist.length) {
				const hist = this.histlist[this.histpos];
				if (CP[hist.puid]) {
					return `forward to ${CP[hist.puid]}`;
				} else if (hist.path) {
					return `forward to /id${hist.aid}/path/${hist.path}`;
				} else {
					return "forward to home";
				}
			}
			return "go forward";
		},
		hintparent() {
			if (this.curpathway.length) {
				return this.curpathway.map(e => e.name).join("/");
			} else {
				return "to root folder";
			}
		},
		hintauthcaret() {
			return this.showauth ? "hide login fields" : "show login fields";
		}
	},
	methods: {
		async fetchicons(link) {
			const response = await fetch(link);
			if (!response.ok) {
				throw new HttpError(response.status, { what: "can not load icons mapping file", when: Date.now(), code: 0 });
			}
			iconmapping = await response.json();
			eventHub.emit('iconset', iconmapping);
		},

		async fetchishome() {
			const response = await fetchajaxauth("POST", "/api/res/ishome", {
				aid: this.aid
			});
			traceajax(response);
			if (!response.ok) {
				throw new HttpError(response.status, response.data);
			}
			this.ishome = response.data.ishome;
		},

		async fetchcategory(hist) {
			const response = await fetchajaxauth("POST", "/api/res/ctgr", {
				aid: hist.aid, puid: hist.puid
			});
			traceajax(response);
			if (!response.ok) {
				throw new HttpError(response.status, response.data);
			}

			// update shared
			if (hist.puid === PUID.shares && this.isadmin) {
				this.shared = response.data.list ?? [];
			}

			// current path & state
			this.curscan = new Date(Date.now());
			this.curpuid = hist.puid;
			this.curpath = "";
			this.shrname = "";
			document.title = `hms - ${this.curbasename}`;

			this.newfolder(response.data.list);
		},

		// opens given folder cleary
		async fetchfolder(hist) {
			const response = await fetchajaxauth("POST", "/api/res/folder", {
				aid: hist.aid, puid: hist.puid, path: hist.path
			});
			traceajax(response);
			if (!response.ok) {
				throw new HttpError(response.status, response.data);
			}

			hist.puid = response.data.puid;
			hist.path = response.data.path;
			// current path & state
			this.curscan = new Date(Date.now());
			this.curpuid = response.data.puid;
			this.curpath = response.data.path;
			this.shrname = response.data.shrname;
			document.title = `hms - ${this.curbasename}`;

			this.newfolder(response.data.list);
		},

		async fetchopenroute(hist) {
			if (!hist.puid && !hist.path) {
				hist.puid = PUID.home;
			}
			await this.fetchscanbreak() // stop previous folder scanning
			if (CID[hist.puid]) {
				await this.fetchcategory(hist);
			} else {
				await this.fetchfolder(hist);
			}
			this.fetchscanstart(); // fetch at backround
			this.seturl();
		},

		async fetchrangesearch(arg) {
			const response = await fetchajaxauth("POST", "/api/gps/range", arg);
			traceajax(response);
			if (!response.ok) {
				throw new HttpError(response.status, response.data);
			}

			// current path & state
			this.curscan = new Date(Date.now());
			this.curpuid = PUID.map;
			this.curpath = "";
			this.shrname = "";
			document.title = `hms - ${this.curbasename}`;

			this.newfolder(response.data.list);
		},

		async fetchscanstart() {
			if (!this.uncached.length) {
				return;
			}

			const response = await fetchjsonauth("POST", "/api/tmb/scnstart", {
				aid: this.aid,
				list: this.uncached.map(file => file.puid)
			});
			traceajax(response);
			if (!response.ok) {
				throw new HttpError(response.status, response.data);
			}

			// cache folder thumnails
			const curpuid = this.puid;
			while (curpuid === this.puid && this.uncached.length) {
				// check cached state loop
				const response = await fetchajaxauth("POST", "/api/tmb/chk", {
					puids: this.uncached.map(file => file.puid)
				});
				traceajax(response);
				if (!response.ok) {
					throw new HttpError(response.status, response.data);
				}

				const gpslist = [];
				for (const tp of response.data.tmbs) {
					if (tp.mtmb) {
						for (const file of this.flist) {
							if (file.puid === tp.puid) {
								file.mtmb = tp.mtmb; // Vue.set
								// add gps-item
								if (file.latitude && file.longitude && Number(file.mtmb) > 0) {
									gpslist.push(file);
								}
								break;
							}
						}
					}
				}
				// update map card
				this.$refs.mcard.addmarkers(gpslist);
				// wait and run again
				await new Promise(resolve => setTimeout(resolve, 1500));
			}
		},

		async fetchscanbreak() {
			if (!this.uncached.length) {
				return;
			}

			const response = await fetchjsonauth("POST", "/api/tmb/scnbreak", {
				aid: this.aid,
				list: this.uncached.map(file => file.puid)
			});
			traceajax(response);
			if (!response.ok) {
				throw new HttpError(response.status, response.data);
			}
		},

		async fetchshared() {
			const response = await fetchajaxauth("POST", "/api/res/ctgr", {
				aid: this.aid, puid: PUID.shares
			});
			traceajax(response);
			if (!response.ok) {
				throw new HttpError(response.status, response.data);
			}
			this.shared = response.data.list ?? [];
			this.updateshared();
		},

		async fetchshareadd(file) {
			const response = await fetchajaxauth("POST", "/api/share/add", {
				aid: this.aid,
				puid: file.puid
			});
			traceajax(response);
			if (!response.ok) {
				throw new HttpError(response.status, response.data);
			}
			file.shared = true; // Vue.set
			this.shared.push(file);
		},

		async fetchsharedel(file) {
			const response = await fetchajaxauth("DELETE", "/api/share/del", {
				aid: this.aid,
				puid: file.puid
			});
			traceajax(response);
			if (!response.ok) {
				throw new HttpError(response.status, response.data);
			}

			// update folder settings
			if (response.data.deleted) { // on ok
				file.shared = false; // Vue.set
				for (let i = 0; i < this.shared.length;) {
					if (this.shared[i].puid === file.puid) {
						this.shared.splice(i, 1);
					} else {
						i++;
					}
				}
			}
		},

		updateshared() {
			for (const file of this.flist) {
				let sf = false;
				for (const shr of this.shared) {
					if (shr.puid === file.puid) {
						sf = true;
						break;
					}
				}
				file.shared = sf; // Vue.set
			}
		},

		setskin(skinid) {
			if (skinid !== this.skinid) {
				for (const v of this.resmodel.skinlist) {
					if (v.id === skinid) {
						document.getElementById('skinmodel')?.setAttribute('href', v.link);
						sessionStorage.setItem('skinid', skinid);
						this.skinid = skinid;
					}
				}
			}
		},

		seticon(iconid) {
			if (iconid !== this.iconid) {
				for (const v of this.resmodel.iconlist) {
					if (v.id === iconid) {
						(async () => {
							eventHub.emit('ajax', +1);
							try {
								await this.fetchicons(v.link);
							} catch (e) {
								ajaxfail(e);
							} finally {
								eventHub.emit('ajax', -1);
							}
						})();
						sessionStorage.setItem('iconid', iconid);
						this.iconid = iconid;
					}
				}
			}
		},

		seturl() {
			const url = (() => {
				if (CID[this.curpuid]) {
					return `${(devmode ? "/dev" : "")}/id${this.aid}/ctgr/${CID[this.curpuid]}/`;
				} else if (this.curpath) {
					return this.curlongurl;
				} else {
					return `${(devmode ? "/dev" : "")}/id${this.aid}/home/`;
				}
			})();
			window.history.replaceState(null, this.curpath, url);
		},

		// push item into folders history
		pushhist(hist) {
			this.histlist.splice(this.histpos);
			this.histlist.push(hist);
			this.histpos = this.histlist.length;
		},

		newfolder(list) {
			this.flist = [];
			// update folder settings
			if (this.curpuid === PUID.home) {
				for (const file of list ?? []) {
					this.flist.push(file);
				}
			} else {
				for (const file of list ?? []) {
					if (file && file.type !== FT.ctgr) {
						this.flist.push(file);
					}
				}
			}
			this.updateshared();

			// clear current selected
			eventHub.emit('select', null);

			// init map card
			this.$refs.mcard.new(this.curpuid === PUID.map);
			// update map card
			const gpslist = [];
			for (const file of this.flist) {
				if (file.latitude && file.longitude && Number(file.mtmb) > 0) {
					gpslist.push(file);
				}
				if (pathext(file.name) === ".gpx") {
					this.$refs.mcard.addgpx(file);
				}
			}
			this.$refs.mcard.addmarkers(gpslist);
		},

		onhome() {
			(async () => {
				eventHub.emit('ajax', +1);
				try {
					// open route and push history step
					const hist = { aid: this.aid, puid: PUID.home };
					await this.fetchopenroute(hist);
					this.pushhist(hist);
				} catch (e) {
					ajaxfail(e);
				} finally {
					eventHub.emit('ajax', -1);
				}
			})();
		},

		onback() {
			(async () => {
				eventHub.emit('ajax', +1);
				try {
					this.histpos--;
					const hist = this.histlist[this.histpos - 1];
					await this.fetchopenroute(hist);
				} catch (e) {
					ajaxfail(e);
				} finally {
					eventHub.emit('ajax', -1);
				}
			})();
		},

		onforward() {
			(async () => {
				eventHub.emit('ajax', +1);
				try {
					this.histpos++;
					const hist = this.histlist[this.histpos - 1];
					await this.fetchopenroute(hist);
				} catch (e) {
					ajaxfail(e);
				} finally {
					eventHub.emit('ajax', -1);
				}
			})();
		},

		onparent() {
			(async () => {
				eventHub.emit('ajax', +1);
				try {
					// open route and push history step
					const path = this.curpathway.length
						? this.curpathway[this.curpathway.length - 1].path
						: "";
					const hist = { aid: this.aid, path: path };
					await this.fetchopenroute(hist);
					this.pushhist(hist);
				} catch (e) {
					ajaxfail(e);
				} finally {
					eventHub.emit('ajax', -1);
				}
			})();
		},

		onrefresh() {
			(async () => {
				eventHub.emit('ajax', +1);
				try {
					await this.fetchopenroute({ aid: this.aid, puid: this.curpuid, path: this.curpath });
					if (this.isadmin && this.curpuid !== PUID.shares) {
						await this.fetchshared(); // get shares
					}
					if (!this.isadmin) {
						await this.fetchishome();
					}
				} catch (e) {
					ajaxfail(e);
				} finally {
					eventHub.emit('ajax', -1);
				}
			})();
		},

		onlink() {
			copyTextToClipboard(window.location.origin + fileurl(this.selfile));
		},
		onshare() {
			(async () => {
				eventHub.emit('ajax', +1);
				try {
					if (this.selfile.shared) { // should remove share
						await this.fetchsharedel(this.selfile);
					} else { // should add share
						await this.fetchshareadd(this.selfile);
					}
				} catch (e) {
					ajaxfail(e);
				} finally {
					eventHub.emit('ajax', -1);
				}
			})();
		},
		onauthcaret() {
			this.showauth = !this.showauth;
		},

		ondiskadd() {
			(async () => {
				eventHub.emit('ajax', +1);
				try {
					const response = await fetchajaxauth("POST", "/api/drive/add", {
						aid: this.$root.aid,
						path: this.diskpath
					});
					traceajax(response);
					if (response.ok) {
						const file = response.data;
						if (file) {
							this.flist.push(file);
						}
						this.diskadd?.hide();
					} else {
						this.diskpathstate = -1;
					}
				} catch (e) {
					ajaxfail(e);
				} finally {
					eventHub.emit('ajax', -1);
				}
			})();
		},
		ondiskremove() {
			(async () => {
				eventHub.emit('ajax', +1);
				try {
					const response = await fetchajaxauth("POST", "/api/drive/del", {
						aid: this.$root.aid,
						puid: this.selfile.puid
					});
					traceajax(response);
					if (!response.ok) {
						throw new HttpError(response.status, response.data);
					}

					if (response.data.deleted) {
						this.flist.splice(this.flist.findIndex(elem => elem === this.selfile), 1);
						if (this.selfile.shared) {
							await this.fetchsharedel(this.selfile);
						}
					}
				} catch (e) {
					ajaxfail(e);
				} finally {
					eventHub.emit('ajax', -1);
				}
			})();
		},
		ondiskpathchange(e) {
			(async () => {
				try {
					const response = await fetchajaxauth("POST", "/api/res/ispath", {
						aid: this.$root.aid,
						path: this.diskpath
					});
					if (response.ok) {
						this.diskpathstate = response.data ? 1 : 0;
					} else {
						this.diskpathstate = -1;
					}
				} catch (e) {
					ajaxfail(e);
				}
			})();
		},
		oncopy() {
			this.copied = this.selfile;
			this.cuted = null;
		},
		paste(ovw) {
			(async () => {
				try {
					if (this.copied) {
						const file = this.copied;
						const response = await fetchajaxauth("POST", "/api/edit/copy", {
							aid: this.$root.aid,
							src: file.puid,
							dst: this.curpuid,
							overwrite: ovw
						});
						traceajax(response);
						if (response.ok) {
							// update folder settings
							this.flist.push(response.data);
							this.fetchscanstart(); // fetch at backround
						} else {
							throw new HttpError(response.status, response.data);
						}
					} else {
						const file = this.cuted;
						const response = await fetchajaxauth("POST", "/api/edit/rename", {
							aid: this.$root.aid,
							src: file.puid,
							dst: this.curpuid,
							overwrite: false
						});
						traceajax(response);
						if (response.ok) {
							// update folder settings
							for (let i = 0; i < this.flist.length; i++) {
								if (this.flist[i].puid === file.puid) {
									this.flist.splice(i, 1);
									break;
								}
							}
							this.flist.push(response.data);
							this.fetchscanstart(); // fetch at backround
						} else {
							throw new HttpError(response.status, response.data);
						}
					}
				} catch (e) {
					ajaxfail(e);
				}
			})();
		},
		onpaste() {
			this.paste(true);
		},
		onpastego() {
			this.paste(false);
		},
		oncut() {
			this.cuted = this.selfile;
			this.copied = null;
		},
		ondelete() {
			(async () => {
				const file = this.selfile;
				try {
					const response = await fetchjsonauth("POST", "/api/edit/delete", {
						aid: this.$root.aid,
						puid: file.puid
					});
					traceajax(response);
					if (response.ok) {
						// update folder settings
						for (let i = 0; i < this.flist.length; i++) {
							if (this.flist[i].puid === file.puid) {
								this.flist.splice(i, 1);
								break;
							}
						}
					} else {
						const data = await response.json();
						throw new HttpError(response.status, data);
					}
				} catch (e) {
					ajaxfail(e);
				}
				eventHub.emit('select', null);
			})();
		},

		rangesearch(arg) {
			(async () => {
				try {
					await this.fetchscanbreak() // stop previous folder scanning
					if (this.curpuid !== PUID.map) {
						// open route and push history step
						const hist = { aid: this.aid, puid: PUID.map };
						await this.fetchrangesearch(arg);
						this.pushhist(hist);
					} else {
						await this.fetchrangesearch(arg);
					}
					this.fetchscanstart(); // fetch at backround
					this.seturl();
				} catch (e) {
					ajaxfail(e);
				}
			})();
		},

		authclosure(is) {
			this.isauth = is;
			if (is) {
				const claims = auth.claims();
				if (claims && 'aid' in claims) {
					this.authid = claims.aid;
					this.aid = claims.aid;
				}
			}
		},
		onopen(file) {
			if (!file.type && !file.size) {
				return;
			}
			const ext = pathext(file.name);
			if (file.type || ext === ".iso" || extfmt.playlist[ext]) {
				if (!file.latency || file.latency > 0) {
					(async () => {
						eventHub.emit('ajax', +1);
						try {
							// open route and push history step
							const hist = {
								aid: this.aid,
								puid: file.puid,
								path: file.path
							};
							await this.fetchopenroute(hist);
							this.pushhist(hist);
						} catch (e) {
							ajaxfail(e);
						} finally {
							eventHub.emit('ajax', -1);
						}
					})();
				}
			} else if (extfmt.books[ext] || extfmt.texts[ext]) {
				const url = mediaurl(file, 1, 0);
				window.open(url, file.name);
			}
		},
		onselect(file) {
			// deselect previous
			if (this.selfile) {
				this.selfile.selected = false; // Vue.set
			}
			// select current
			this.selfile = file;
			if (file) {
				file.selected = true; // Vue.set
			}
		},
		onplayback(file, isplay) {
			file.playback = isplay; // Vue.set
		}
	},
	created() {
		eventHub.on('auth', this.authclosure);
		eventHub.on('ajax', viewpreloader);
		eventHub.on('open', this.onopen);
		eventHub.on('select', this.onselect);
		eventHub.on('playback', this.onplayback);

		auth.signload();
		this.login = auth.login;
		if (devmode && this.isauth) {
			console.log("token:", auth.token);
			console.log("login:", auth.login);
		}
	},
	mounted() {
		const chunks = decodeURI(window.location.pathname).split('/');
		// remove first empty element
		chunks.shift();
		// bring it to true path
		if (chunks[chunks.length - 1].length > 0) {
			chunks.push("");
		}
		// cut "dev" prefix
		if (chunks[0] === "dev") {
			chunks.shift();
		}

		// get profile id
		if (chunks[0].substring(0, 2) === "id") {
			this.aid = Number(chunks[0].substring(2));
			chunks.shift();
		} else {
			this.aid = 1;
		}

		// init diskadd dialog
		const el = document.getElementById('diskadd' + this.iid);
		if (el) {
			this.diskadd = new bootstrap.Modal(el);
			el.addEventListener('shown.bs.modal', e => {
				el.querySelector('input').focus();
			});
		}

		const hist = { aid: this.aid };
		// get route
		const route = chunks[0];
		chunks.shift();
		if (route === "ctgr") {
			hist.puid = PUID[chunks[0]];
			chunks.shift();
		} else if (route === "path") {
			hist.path = chunks.join('/');
		} else if (route === "main") {
			hist.puid = PUID.home;
		}
		if (!hist.puid && !hist.path) {
			hist.puid = PUID.home;
		}

		// load resources and open route
		(async () => {
			eventHub.emit('ajax', +1);
			try {
				// load resources model at first
				const response = await fetch("/data/assets/resmodel.json");
				if (!response.ok) {
					throw new HttpError(response.status, { what: "can not load resources model file", when: Date.now(), code: 0 });
				}
				this.resmodel = resmodel = await response.json();

				// set skin
				const skinid = sessionStorage.getItem('skinid') ?? this.resmodel.defskinid;
				for (const v of this.resmodel.skinlist) {
					if (v.id === skinid) {
						document.getElementById('skinmodel')?.setAttribute('href', v.link);
						this.skinid = skinid;
					}
				}

				// load icons
				const iconid = sessionStorage.getItem('iconid') ?? this.resmodel.deficonid;
				for (const v of this.resmodel.iconlist) {
					if (v.id === iconid) {
						await this.fetchicons(v.link);
						this.iconid = iconid;
					}
				}

				// open route and push history step
				await this.fetchopenroute(hist);
				if (this.isadmin && hist.puid !== PUID.shares) {
					await this.fetchshared(); // get shares
				}
				if (!this.isadmin) {
					await this.fetchishome();
				}
				this.pushhist(hist);
			} catch (e) {
				ajaxfail(e);
			} finally {
				eventHub.emit('ajax', -1);
			}
		})();

		// hide start-up preloader
		eventHub.emit('ajax', -1);
	},
	unmounted() {
		eventHub.off('auth', this.authclosure);
		eventHub.off('ajax', viewpreloader);
		eventHub.off('open', this.onopen);
		eventHub.off('select', this.onselect);
		eventHub.off('playback', this.onplayback);

		// erase diskadd dialog
		this.diskadd = null;
	}
};

const VueAuth = {
	template: '#auth-tpl',
	data() {
		return {
			isauth: false, // is authorized
			login: "", // authorization login
			password: "", // authorization password
			namestate: 0, // -1 invalid login, 0 ambiguous, 1 valid login
			passstate: 0 // -1 invalid password, 0 ambiguous, 1 valid password
		};
	},
	computed: {
		clsname() {
			return !this.namestate ? ''
				: this.namestate === -1 ? 'is-invalid' : 'is-valid';
		},
		clspass() {
			return !this.passstate ? ''
				: this.passstate === -1 ? 'is-invalid' : 'is-valid';
		}
	},
	methods: {
		onauthchange() {
			this.namestate = 0;
			this.passstate = 0;
		},
		onlogin() {
			(async () => {
				eventHub.emit('ajax', +1);
				try {
					const resp1 = await fetchjson("POST", "/api/auth/pubkey");
					const data1 = await resp1.json();
					traceajax(resp1, data1);
					if (!resp1.ok) {
						throw new HttpError(resp1.status, data1);
					}

					// github.com/emn178/js-sha256
					const hash = sha256.hmac.create(data1.key);
					hash.update(this.password);

					const resp2 = await fetchjson("POST", "/api/auth/signin", {
						name: this.login,
						pubk: data1.key,
						hash: hash.digest()
					});
					const data2 = await resp2.json();
					traceajax(resp2, data2);

					if (resp2.status === 200) {
						auth.signin(data2, this.login);
						this.namestate = 1;
						this.passstate = 1;
					} else if (resp2.status === 403) { // Forbidden
						auth.signout();
						switch (data2.code) {
							case 13:
								this.namestate = -1;
								this.passstate = 0;
								break;
							case 15:
								this.namestate = 1;
								this.passstate = -1;
								break;
							default:
								this.namestate = -1;
								this.passstate = -1;
						}
					} else {
						throw new HttpError(resp2.status, data2);
					}
				} catch (e) {
					ajaxfail(e);
				} finally {
					eventHub.emit('ajax', -1);
				}
			})();
		},
		onlogout() {
			auth.signout();
			this.namestate = 0;
			this.passstate = 0;
		},

		authclosure(is) {
			this.isauth = is;
			if (is) {
				this.login = auth.login;
			}
		}
	},
	created() {
		eventHub.on('auth', this.authclosure);
	},
	unmounted() {
		eventHub.off('auth', this.authclosure);
	}
};

// Create application view model
const appws = Vue.createApp(VueMainApp)
	.component('auth-tag', VueAuth)
	.component('thumbslider-tag', VueThumbSlider)
	.component('photoslider-tag', VuePhotoSlider)
	.component('mp3-player-tag', VuePlayer)
	.component('dir-card-tag', VueDirCard)
	.component('file-card-tag', VueFileCard)
	.component('tile-card-tag', VueTileCard)
	.component('map-card-tag', VueMapCard)
	.component('icon-tag', VueIcon)
	.component('list-item-tag', VueListItem)
	.component('file-item-tag', VueFileItem)
	.component('img-item-tag', VueImgItem)
	.component('tile-item-tag', VueTileItem);
const appvm = appws.mount('#app');

// The End.
