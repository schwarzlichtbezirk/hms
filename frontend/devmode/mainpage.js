"use strict";

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
	wavebars: "",
	iconfmt: []
};
let thumbmode = storageGetItem("thumbmode", true);

// Category identifiers by names.
const CNID = {
	home: "04",
	local: "08",
	remote: "0C",
	shares: "0G",
	media: "0K",
	video: "0O",
	audio: "0S",
	image: "10",
	books: "14",
	texts: "18",
	map: "1C",

	reserved: 32
};

// Category names by identifiers.
const CIDN = {
	"04": "home",
	"08": "local",
	"0C": "remote",
	"0G": "shares",
	"0K": "media",
	"0O": "video",
	"0S": "audio",
	"10": "image",
	"14": "books",
	"18": "texts",
	"1C": "map",
};

// Category properties.
const CP = {
	"04": "Home",
	"08": "Drives list",
	"0C": "Network",
	"0G": "Shared resources",
	"0K": "Multimedia files",
	"0O": "Movie and video files",
	"0S": "Music and audio files",
	"10": "Photos and images",
	"14": "Books",
	"18": "Text files",
	"1C": "Map",
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
	unk: 0, // unknown file type
	file: 1,
	dir: 2,
	drv: 3,
	cld: 4,
	ctgr: 5,
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
	group: 7
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
		".tiff": 1, ".tif": 1, ".dng": 1
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

	"font": {
		".otf": 1, ".otc": 1, ".ttc": 1,
		".pcf": 1, ".ttf": 1, ".tte": 1,
		".woff": 1, ".woff2": 1, ".eot": 1,
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
		".tif": 1, ".tiff": 1, ".dng": 1, ".jpg": 1, ".jpe": 1, ".jpeg": 1, ".jfif": 1,
		".gif": 1, ".png": 1, ".webp": 1, ".avif": 1, ".psd": 1, ".psb": 1,
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
	"msoffice", "openoffice", "office", "font",
	"archive", "disk", "package", "playlist",
	"image", "audio", "video", "books", "texts", "packs"
];

const getFileGroup = file => {
	if (file.type !== FT.file) {
		return FG.group;
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

const geticonpath = (im, file) => {
	const org = file.shared ? im.shared : im.private;
	const alt = file.shared ? im.private : im.shared;
	switch (file.type) {
		case FT.ctgr:
			return {
				org: org.cid[CIDN[file.puid]] || org.cid.cid,
				alt: alt.cid[CIDN[file.puid]] || alt.cid.cid
			};
		case FT.cld:
			if (file.latency < 0) {
				return { org: org.drive.offline, alt: alt.drive.offline };
			} else {
				return { org: org.drive.network, alt: alt.drive.network };
			}
		case FT.drv:
			if (file.latency < 0) {
				return { org: org.drive.gray, alt: alt.drive.gray };
			} else if (!file.latency || file.latency < DS.yellow) {
				return { org: org.drive.green, alt: alt.drive.green };
			} else if (file.latency < DS.red) {
				return { org: org.drive.yellow, alt: alt.drive.yellow };
			} else {
				return { org: org.drive.red, alt: alt.drive.red };
			}
		case FT.dir:
			if (file.scan > "0001-01-01T00:00:00Z") {
				let fnum = 0;
				const fg = file.fgrp;
				for (const n in fg) {
					fnum += fg[n];
				}
				if (!fnum) {
					return { org: org.folder.open, alt: alt.folder.open };
				} else if (fg.other / fnum > 0.5) {
					return { org: org.folder.other, alt: alt.folder.other };
				} else if (fg.video / fnum > 0.5) {
					return { org: org.folder.video, alt: alt.folder.video };
				} else if (fg.audio / fnum > 0.5) {
					return { org: org.folder.audio, alt: alt.folder.audio };
				} else if (fg.image / fnum > 0.5) {
					return { org: org.folder.image, alt: alt.folder.image };
				} else if (fg.books / fnum > 0.5) {
					return { org: org.folder.books, alt: alt.folder.books };
				} else if (fg.texts / fnum > 0.5) {
					return { org: org.folder.texts, alt: alt.folder.texts };
				} else if (fg.packs / fnum > 0.5) {
					return { org: org.folder.packs, alt: alt.folder.packs };
				} else if (fg.group / fnum > 0.5) {
					return { org: org.folder.group, alt: alt.folder.group };
				} else if ((fg.audio + fg.video + fg.image) / fnum > 0.5) {
					return { org: org.folder.media, alt: alt.folder.media };
				} else {
					return { org: org.folder.open, alt: alt.folder.open };
				}
			} else {
				return { org: org.folder.close, alt: alt.folder.close };
			}
		case FT.file:
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
			resmodel: { skinlist: [], iconlist: [] },
			showauth: false, // display authorization form
			isauth: false, // is authorized
			aid: 0, // profile access ID
			hashome: false, // able to go home
			access: false, // this client can have access to not shared files

			selfile: null, // current selected item
			delfile: null, // file to delete
			delensured: false, // deletion request

			// history
			histpos: 0, // position in history stack
			histlist: [], // history stack

			// current opened folder data
			flist: [], // list of files and subfolders in in current folder as is
			skipped: 0, // number of skipped files in current folder
			curscan: new Date(), // time of last scanning of current folder
			curpuid: "", // current folder PUID
			sharepath: "", // current folder share path
			sharename: "", // current folder share name
			rootpath: "", // current folder root path
			rootname: "", // current folder root name
			static: false, // content of current folder can bs modified
			copied: null, // copied item
			cuted: null, // cuted item

			iid: makestrid(10) // instance ID
		};
	},
	computed: {
		// is it authorized or running on localhost
		isadmin() {
			return this.isauth || this.access;
		},
		// current page URL
		curshorturl() {
			return `${(devmode ? "/dev" : "")}/id${this.aid}/path/${this.curpuid}`;
		},
		// current page URL
		curlongurl() {
			if (CIDN[this.curpuid]) {
				return `${(devmode ? "/dev" : "")}/id${this.aid}/ctgr/${CIDN[this.curpuid]}/`;
			} else {
				return `${(devmode ? "/dev" : "")}/id${this.aid}/path/${this.curpath}`;
			}
		},
		// current folder path, to share or to disk
		curpath() {
			return this.sharepath || this.rootpath;
		},
		// current path base name
		curbasename() {
			if (CP[this.curpuid]) {
				return CP[this.curpuid];
			} else {
				const arr = this.curpath.split('/');
				if (arr.length > 1) {
					return arr.pop();
				} else {
					return this.sharename || this.rootname;
				}
			}
		},
		// array of paths to current folder
		curpathway() {
			const lst = [];
			if (!this.curpath) {
				return [];
			}

			const arr = this.curpath.split('/');
			arr.pop(); // remove current name

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
				lst[0].name = this.sharename || this.rootname;
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
				if (file.type === FT.file) {
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
			return { 'disabled': this.curpuid === CNID.home || !this.hashome };
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

		delfilename() {
			if (!this.delfile) {
				return 'N/A';
			}
			return this.delfile.name;
		},
		deltypename() {
			if (!this.delfile) {
				return 'null';
			}
			switch (this.delfile.type) {
				case FT.file: return 'file';
				case FT.dir: return 'folder';
				case FT.drv: return 'drive';
				case FT.ctgr: return 'category';
			}
		},

		textauthcaret() {
			return this.showauth ? 'arrow_right' : 'arrow_left';
		},

		showcopypaste() {
			return !!this.rootpath && this.isadmin;
		},
		showpastego() {
		},
		clscopy() {
			return { 'disabled': !this.selfile || !(this.selfile.type === FT.file || this.selfile.type === FT.dir) };
		},
		clspaste() {
			const sel = this.copied ?? this.cuted;
			return {
				'disabled': !sel || this.static || (() => {
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
				'disabled': !sel || this.static || !(() => {
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
			return { 'disabled': !this.selfile || this.selfile.static };
		},
		clsdelete() {
			return { 'disabled': !this.selfile || this.selfile.static };
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
					return `back to /id${this.aid}/path/${hist.path}`;
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
					return `forward to /id${this.aid}/path/${hist.path}`;
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

		// opens given folder cleary
		async fetchfolder(arg) {
			const response = await fetchjsonauth("POST", `/id${this.aid}/api/res/folder`, arg);
			const data = await response.json();
			traceajax(response, data);
			if (!response.ok) {
				throw new HttpError(response.status, data);
			}

			// current path & state
			this.skipped = data.skipped;
			this.curpuid = data.puid;
			this.sharepath = data.sharepath;
			this.sharename = data.sharename;
			this.rootpath = data.rootpath;
			this.rootname = data.rootname;
			this.static = data.static;
			this.hashome = data.hashome;
			this.access = data.access;

			await this.newfolder(data.list);
		},

		async fetchrangesearch(arg) {
			const response = await fetchjsonauth("POST", `/id${this.$root.aid}/api/gps/range`, arg);
			const data = await response.json();
			traceajax(response, data);
			if (!response.ok) {
				throw new HttpError(response.status, data);
			}

			// current path & state
			this.skipped = 0;
			this.curpuid = CNID.map;
			this.sharepath = "";
			this.sharename = "";
			this.rootpath = "";
			this.rootname = "";
			this.static = true;
			this.hashome = data.hashome;
			this.access = true;

			await this.newfolder(data.list);
		},

		async newfolder(newlist) {
			// clear current selected
			eventHub.emit('select', null);

			// update folder settings
			this.flist = newlist ?? [];

			// update page data
			this.curscan = new Date(Date.now());
			window.history.replaceState(null, "", this.curlongurl);
			document.title = `hms - ${this.curbasename}`;
			// scroll page to top
			this.$refs.page.scrollTop = 0;
		},

		async fetchshareadd(file) {
			const response = await fetchjsonauth("POST", `/id${this.aid}/api/share/add`, {
				path: file.puid
			});
			const data = await response.json();
			traceajax(response, data);
			if (!response.ok) {
				throw new HttpError(response.status, data);
			}
			file.shared = true; // Vue.set
		},

		async fetchsharedel(file) {
			const response = await fetchjsonauth("DELETE", `/id${this.aid}/api/share/del`, {
				path: file.puid
			});
			const data = await response.json();
			traceajax(response, data);
			if (!response.ok) {
				throw new HttpError(response.status, data);
			}

			// update folder settings
			if (data.deleted) { // on ok
				file.shared = false; // Vue.set
			}
		},

		setskin(skinid) {
			if (skinid !== this.skinid) {
				for (const v of this.resmodel.skinlist) {
					if (v.id === skinid) {
						document.getElementById('skinmodel')?.setAttribute('href', v.link);
						storageSetItem('skinid', skinid);
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
						storageSetItem('iconid', iconid);
						this.iconid = iconid;
					}
				}
			}
		},

		// push item into folders history
		pushhist(hist) {
			this.histlist.splice(this.histpos);
			this.histlist.push(hist);
			this.histpos = this.histlist.length;
		},

		onhome() {
			(async () => {
				eventHub.emit('ajax', +1);
				try {
					// open route and push history step
					const hist = { path: CNID.home, scan: true };
					await this.fetchfolder(hist);
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
					await this.fetchfolder(hist);
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
					await this.fetchfolder(hist);
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
					const path = this.curpathway[this.curpathway.length - 1].path || CNID.home;
					const hist = { path: path, scan: true };
					await this.fetchfolder(hist);
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
					await this.fetchfolder({
						path: this.curpuid ?? this.curpath,
						scan: true,
					});
				} catch (e) {
					ajaxfail(e);
				} finally {
					eventHub.emit('ajax', -1);
				}
			})();
		},

		onlink() {
			navigator.clipboard.writeText(window.location.origin + `/id${this.aid}/file/${this.selfile.puid}`);
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

		oncopy() {
			this.copied = this.selfile;
			this.cuted = null;
		},
		paste(ovw) {
			(async () => {
				try {
					const file = this.copied || this.cuted;
					const response = await fetchjsonauth("POST", `/id${this.$root.aid}/api/edit/copy`, {
						src: file.puid,
						dst: this.curpuid,
						overwrite: ovw && !!this.copied
					});
					const data = await response.json();
					traceajax(response, data);
					if (!response.ok) {
						throw new HttpError(response.status, data);
					}
					// update folder settings
					if (this.cuted) {
						for (let i = 0; i < this.flist.length; i++) {
							if (this.flist[i].puid === file.puid) {
								this.flist.splice(i, 1);
								break;
							}
						}
					}
					this.flist.push(data);
					await this.$refs.fcard.fetchtmbscan(); // fetch at backround
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
		ondelask() {
			this.delfile = this.selfile;
			if (this.delensured) {
				this.ondelete();
			} else {
				const dlg = new bootstrap.Modal('#delask');
				dlg.show();
			}
		},
		ondelete() {
			if (!this.delfile) {
				return;
			}
			(async () => {
				try {
					const response = await fetchjsonauth("POST", `/id${this.$root.aid}/api/edit/delete`, {
						src: this.delfile.puid
					});
					traceajax(response);
					if (response.ok) {
						// update folder settings
						for (let i = 0; i < this.flist.length; i++) {
							if (this.flist[i].puid === this.delfile.puid) {
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
				if (this.delfile === this.selfile) {
					eventHub.emit('select', null);
				}
				this.delfile = null;
			})();
		},

		rangesearch(arg) {
			(async () => {
				try {
					await this.fetchrangesearch(arg);
					if (this.curpuid !== CNID.map) {
						// open route and push history step
						const hist = { aid: this.aid, puid: CNID.map };
						this.pushhist(hist);
					}
				} catch (e) {
					ajaxfail(e);
				}
			})();
		},

		authclosure(auth) {
			this.isauth = auth.signed;
			if (auth.signed) {
				const claims = auth.claims();
				if (claims && 'uid' in claims) {
					this.aid = claims.uid;
				}
			}
		},
		onopen(file) {
			if (file.type === FT.file && !file.size) {
				return;
			}
			const ext = pathext(file.name);
			if (file.type !== FT.file || ext === ".iso" || extfmt.playlist[ext]) {
				if (!file.latency || file.latency > 0) {
					(async () => {
						eventHub.emit('ajax', +1);
						try {
							// open route and push history step
							const hist = {
								path: file.puid ?? file.path,
								scan: true,
							};
							await this.fetchfolder(hist);
							this.pushhist(hist);
						} catch (e) {
							ajaxfail(e);
						} finally {
							eventHub.emit('ajax', -1);
						}
					})();
				}
			} else if (extfmt.books[ext] || extfmt.texts[ext]) {
				const url = `/id${this.aid}/file/${file.puid}?media=1&hd=0`;
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
		auth.signload(); // run it after handler set
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

		const hist = { scan: true };
		// get route
		const route = chunks[0];
		chunks.shift();
		if (route === "ctgr") {
			hist.path = CNID[chunks[0]];
			chunks.shift();
		} else if (route === "path") {
			hist.path = chunks.join('/');
		} else if (route === "main") {
			hist.path = CNID.home;
		} else {
			hist.path = CNID.home;
		}

		// load resources and open route
		(async () => {
			eventHub.emit('ajax', +1);
			try {
				// load resources model at first
				const response = await fetch("/fs/assets/resmodel.json");
				if (!response.ok) {
					throw new HttpError(response.status, { what: "can not load resources model file", when: Date.now(), code: 0 });
				}
				this.resmodel = await response.json();

				// set skin
				const skinid = storageGetItem('skinid', this.resmodel.defskinid);
				for (const v of this.resmodel.skinlist) {
					if (v.id === skinid) {
						document.getElementById('skinmodel')?.setAttribute('href', v.link);
						this.skinid = skinid;
					}
				}

				// load icons
				const iconid = storageGetItem('iconid', this.resmodel.deficonid);
				for (const v of this.resmodel.iconlist) {
					if (v.id === iconid) {
						await this.fetchicons(v.link);
						this.iconid = iconid;
					}
				}

				// open route and push history step
				await this.fetchfolder(hist);
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
		},
		clslogin() {
			return { 'disabled': !this.login || !this.password };
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
						auth.signin(data2.access, data2.refrsh, this.login);
						this.namestate = 1;
						this.passstate = 1;
					} else if (resp2.status === 403) { // Forbidden
						auth.signout();
						switch (data2.code) {
							case 61: // SEC_signin_noacc
								this.namestate = -1;
								this.passstate = 0;
								break;
							case 63: // SEC_signin_deny
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

		authclosure(auth) {
			this.isauth = auth.signed;
			if (auth.signed) {
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
Vue.createApp(VueMainApp)
	.component('auth-tag', VueAuth)
	.component('thumbslider-tag', VueThumbSlider)
	.component('photoslider-tag', VuePhotoSlider)
	.component('mp3-player-tag', VuePlayer)
	.component('ctgr-card-tag', VueCtgrCard)
	.component('cloud-card-tag', VueCloudCard)
	.component('drive-card-tag', VueDriveCard)
	.component('dir-card-tag', VueDirCard)
	.component('file-card-tag', VueFileCard)
	.component('tile-card-tag', VueTileCard)
	.component('map-card-tag', VueMapCard)
	.component('icon-tag', VueIcon)
	.component('iconmenu-tag', VueIconMenu)
	.component('list-item-tag', VueListItem)
	.component('file-item-tag', VueFileItem)
	.component('img-item-tag', VueImgItem)
	.component('tile-item-tag', VueTileItem)
	.mount('#app');

// The End.
