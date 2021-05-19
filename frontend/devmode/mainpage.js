"use strict";

let iconmapping = imempty;
let thumbmode = true;

// icon mapping event model
const iconev = extend({}, makeeventmodel());

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

var extgrp = {
	// Video
	".avi": FG.video,
	".mpe": FG.video,
	".mpg": FG.video,
	".mp4": FG.video,
	".webm": FG.video,
	".wmv": FG.video,
	".wmx": FG.video,
	".flv": FG.video,
	".3gp": FG.video,
	".3g2": FG.video,
	".mkv": FG.video,
	".mov": FG.video,
	".ogv": FG.video,
	".ogx": FG.video,

	// Audio
	".aac": FG.audio,
	".m4a": FG.audio,
	".alac": FG.audio,
	".aif": FG.audio,
	".mpa": FG.audio,
	".mp3": FG.audio,
	".wav": FG.audio,
	".wma": FG.audio,
	".weba": FG.audio,
	".oga": FG.audio,
	".ogg": FG.audio,
	".opus": FG.audio,
	".flac": FG.audio,
	".mka": FG.audio,
	".ra": FG.audio,
	".mid": FG.audio,
	".midi": FG.audio,
	".cda": FG.audio,

	// Images
	".tga": FG.image,
	".bmp": FG.image,
	".dib": FG.image,
	".rle": FG.image,
	".dds": FG.image,
	".tif": FG.image,
	".tiff": FG.image,
	".jpg": FG.image,
	".jpe": FG.image,
	".jpeg": FG.image,
	".jfif": FG.image,
	".gif": FG.image,
	".png": FG.image,
	".webp": FG.image,
	".psd": FG.image,
	".psb": FG.image,
	".jp2": FG.image,
	".jpg2": FG.image,
	".jpx": FG.image,
	".jpm": FG.image,
	".jxr": FG.image,

	// Books
	".pdf": FG.books,
	".djvu": FG.books,
	".djv": FG.books,
	".html": FG.books,
	".htm": FG.books,
	".shtml": FG.books,
	".shtm": FG.books,
	".xhtml": FG.books,
	".phtml": FG.books,
	".hta": FG.books,
	".mht": FG.books,
	// Office
	".odt": FG.books,
	".ods": FG.books,
	".odp": FG.books,
	".rtf": FG.books,
	".abw": FG.books,
	".doc": FG.books,
	".docx": FG.books,
	".xls": FG.books,
	".xlsx": FG.books,
	".ppt": FG.books,
	".pptx": FG.books,
	".vsd": FG.books,

	// Texts
	".txt": FG.texts,
	".md": FG.texts,
	".css": FG.texts,
	".js": FG.texts,
	".jsm": FG.texts,
	".vb": FG.texts,
	".vbs": FG.texts,
	".bat": FG.texts,
	".cmd": FG.texts,
	".sh": FG.texts,
	".mak": FG.texts,
	".iss": FG.texts,
	".nsi": FG.texts,
	".nsh": FG.texts,
	".bsh": FG.texts,
	".sql": FG.texts,
	".as": FG.texts,
	".mx": FG.texts,
	".ps": FG.texts,
	".php": FG.texts,
	".phpt": FG.texts,
	".lua": FG.texts,
	".tcl": FG.texts,
	".rc": FG.texts,
	".cmake": FG.texts,
	".java": FG.texts,
	".jsp": FG.texts,
	".asp": FG.texts,
	".asm": FG.texts,
	".c": FG.texts,
	".h": FG.texts,
	".hpp": FG.texts,
	".hxx": FG.texts,
	".cpp": FG.texts,
	".cxx": FG.texts,
	".cc": FG.texts,
	".cs": FG.texts,
	".go": FG.texts,
	".r": FG.texts,
	".d": FG.texts,
	".pas": FG.texts,
	".inc": FG.texts,
	".py": FG.texts,
	".pyw": FG.texts,
	".pl": FG.texts,
	".pm": FG.texts,
	".plx": FG.texts,
	".rb": FG.texts,
	".rbw": FG.texts,
	".cfg": FG.texts,
	".ini": FG.texts,
	".inf": FG.texts,
	".reg": FG.texts,
	".url": FG.texts,
	".xml": FG.texts,
	".xsml": FG.texts,
	".xsl": FG.texts,
	".xsd": FG.texts,
	".kml": FG.texts,
	".gpx": FG.texts,
	".wsdl": FG.texts,
	".xlf": FG.texts,
	".xliff": FG.texts,
	".yml": FG.texts,
	".yaml": FG.texts,
	".json": FG.texts,
	".log": FG.texts,

	// storage
	".cab": FG.packs,
	".zip": FG.packs,
	".7z": FG.packs,
	".rar": FG.packs,
	".rev": FG.packs,
	".jar": FG.packs,
	".tar": FG.packs,
	".tgz": FG.packs,
	".gz": FG.packs,
	".bz2": FG.packs,
	".iso": FG.packs,
	".isz": FG.packs,
	".udf": FG.packs,
	".nrg": FG.packs,
	".mdf": FG.packs,
	".mdx": FG.packs,
	".img": FG.packs,
	".ima": FG.packs,
	".imz": FG.packs,
	".ccd": FG.packs,
	".vc4": FG.packs,
	".dmg": FG.packs,
	".daa": FG.packs,
	".uif": FG.packs,
	".vhd": FG.packs,
	".vhdx": FG.packs,
	".vmdk": FG.packs,
	".wpk": FG.packs
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
		".py": 1, ".pyw": 1, ".pl": 1, ".pm": 1, ".plx": 1, ".rb": 1, ".rbw":1
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
		".jar": 1, ".tar": 1, ".tgz": 1, ".gz": 1, ".bz2": 1
	},
	"package": {
		".wpk": 1
	},
	"disk": {
		".iso": 1, ".isz": 1, ".udf": 1, ".nrg": 1, ".mdf": 1, ".mdx": 1,
		".img": 1, ".ima": 1, ".imz": 1, ".ccd": 1, ".vc4": 1, ".dmg": 1,
		".daa": 1, ".uif": 1, ".vhd": 1, ".vhdx": 1, ".vmdk": 1
	},

	"image": {
		".tga": 1, ".bmp": 1, ".dib": 1, ".rle": 1, ".dds": 1,
		".tif": 1, ".tiff": 1, ".jpg": 1, ".jpe": 1, ".jpeg": 1, ".jfif": 1,
		".gif": 1, ".png": 1, ".webp": 1, ".psd": 1, ".psb": 1,
		".jp2": 1, ".jpg2": 1, ".jpx": 1, ".jpm": 1, ".jxr": 1
	},
	"music": {
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
		".jar": 1, ".tar": 1, ".tgz": 1, ".gz": 1, ".bz2": 1,
		".wpk": 1,
		".iso": 1, ".isz": 1, ".udf": 1, ".nrg": 1, ".mdf": 1, ".mdx": 1,
		".img": 1, ".ima": 1, ".imz": 1, ".ccd": 1, ".vc4": 1, ".dmg": 1,
		".daa": 1, ".uif": 1, ".vhd": 1, ".vhdx": 1, ".vmdk": 1
	}
};

const extfmtorder = [
	"bitmap", "tiff", "jpeg", "jpeg2000", "psd",
	"component", "exec",
	"text", "html", "config", "datafmt", "script", "code",
	"msoffice", "openoffice", "office",
	"archive", "package", "disk",
	"image", "music", "video", "books", "texts", "packs"
];

const getFileGroup = file => {
	if (file.type) {
		return FG.dir;
	}
	return extgrp[pathext(file.name)] || FG.other;
};

const geticonpath = (file, im, shr) => {
	const org = shr ? im.shared : im.private;
	const alt = shr ? im.private : im.shared;
	switch (file.type) {
		case FT.ctgr:
			return {
				org: org.cid[file.cid] || org.cid.cid,
				alt: alt.cid[file.cid] || alt.cid.cid
			};
		case FT.drv:
			if (file.latency < 0) {
				return { org: org.drive.offline, alt: alt.drive.offline };
			} else if (file.latency < DS.yellow) {
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

const fileurl = file => `/id${app.aid}/file/${file.puid}`;
const pathurl = file => `${(devmode ? "/dev" : "")}/id${app.aid}/path/${file.puid}`;
const mediaurl = file => `/id${app.aid}/media/${file.puid}`;

const showmsgbox = (title, message, details) => {
	const dlg = $("#msgbox");
	dlg.find(".modal-title").text(title);
	dlg.find(".message").text(message);
	dlg.find(".details").text(details || "");
	dlg.modal("show");
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
			const dlg = $("#msgbox");
			dlg.find(".modal-title").text(title);
			dlg.find(".message").text(message);
			dlg.find(".errcode").text(e.code);
			dlg.find(".errmsg").text(e.what);
			dlg.modal("show");
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
	}
	showmsgbox(
		"Server unavailable",
		"Server is currently not available, action can not be done now."
	);
};

Vue.component('auth-tag', {
	template: '#auth-tpl',
	data: function () {
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
				eventHub.$emit('ajax', +1);
				try {
					const resp1 = await fetchjson("POST", "/api/auth/pubkey");
					const data1 = await resp1.json();
					traceajax(resp1, data1);
					if (!resp1.ok) {
						throw new HttpError(resp1.status, data1);
					}

					// github.com/emn178/js-sha256
					const hash = sha256.hmac.create(data1);
					hash.update(this.password);

					const resp2 = await fetchjson("POST", "/api/auth/signin", {
						name: this.login,
						pubk: data1,
						hash: hash.digest()
					});
					const data2 = await resp2.json();
					traceajax(resp2, data2);

					if (resp2.status === 200) {
						auth.signin(data2, this.login);
						this.namestate = 1;
						this.passstate = 1;
						this.$emit('refresh');
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
					eventHub.$emit('ajax', -1);
				}
			})();
		},
		onlogout() {
			auth.signout();
			this.namestate = 0;
			this.passstate = 0;
			this.$emit('refresh');
		},

		authclosure(is) {
			this.isauth = is;
			if (is) {
				this.login = auth.login;
			}
		}
	},
	created() {
		eventHub.$on('auth', this.authclosure);
	},
	beforeDestroy() {
		eventHub.$off('auth', this.authclosure);
	}
});

const app = new Vue({
	el: '#app',
	template: '#app-tpl',
	data: {
		skinid: "", // ID of skin CSS
		iconid: "", // ID of icons mapping json
		resmodel: { skinlist: [], iconlist: [] },
		showauth: false, // display authorization form
		isauth: false, // is authorized
		authid: 0, // authorized ID
		aid: 0, // profile ID
		ishome: false, // able to go home

		loadcount: 0, // ajax working request count
		shared: [], // list of shared folders

		// history
		histpos: 0, // position in history stack
		histlist: [], // history stack

		// current opened folder data
		pathlist: [], // list of subfolders properties in current folder
		filelist: [], // list of files properties in current folder
		curscan: new Date(), // time of last scanning of current folder
		curcid: "", // current category ID
		curpuid: "", // current folder PUID
		curpath: "", // current folder path and path state
		shrname: "" // current folder path share name
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
			if (this.curcid) {
				switch (this.curcid) {
					case "drives":
						return 'Drives list';
					case "shares":
						return 'Shared resources';
					case "media":
						return "Multimedia files";
					case "video":
						return "Movie and video files";
					case "audio":
						return "Music and audio files";
					case "image":
						return "Photos and images";
					case "books":
						return "Books";
					case "texts":
						return "Text files";
					default:
						return this.curcid;
				}
			} else if (this.curpath) {
				const arr = this.curpath.split('/');
				const base = arr.pop() || arr.pop();
				return !arr.length && this.shrname || base;
			} else {
				return 'home page';
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
			for (const file of this.filelist) {
				if (!file.ntmb) {
					lst.push(file);
				}
			}
			return lst;
		},

		// files sum size
		sumsize() {
			let ss = 0;
			for (let file of this.filelist) {
				ss += file.size || 0;
			}
			return fmtitemsize(ss);
		},

		// common buttons enablers

		dishome() {
			return this.curcid === "home" || !(this.isadmin || this.ishome);
		},
		disback() {
			return this.histpos < 2;
		},
		disforward() {
			return this.histpos >= this.histlist.length;
		},
		disparent() {
			return !this.curpathway.length;
		},

		textauthcaret() {
			return this.showauth ? 'arrow_right' : 'arrow_left';
		},

		// buttons hints
		hintback() {
			if (this.histpos > 1) {
				const hist = this.histlist[this.histpos - 2];
				if (hist.cid) {
					return `back to ${hist.cid}`;
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
				if (hist.cid) {
					return `forward to ${hist.cid}`;
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
			iconmapping.iconwebp = this.resmodel.iconwebp;
			iconmapping.iconpng = this.resmodel.iconpng;
			iconev.emit('plug');
		},

		async fetchishome() {
			const response = await fetchajaxauth("POST", "/api/card/ishome", {
				aid: this.aid
			});
			traceajax(response);
			if (!response.ok) {
				throw new HttpError(response.status, response.data);
			}
			this.ishome = response.data;
		},

		async fetchcategory(hist) {
			const response = await fetchajaxauth("POST", "/api/card/ctgr", {
				aid: hist.aid, puid: hist.puid, cid: hist.cid
			});
			traceajax(response);
			if (!response.ok) {
				throw new HttpError(response.status, response.data);
			}

			this.pathlist = [];
			this.filelist = [];
			// update folder settings
			for (const fp of response.data || []) {
				if (fp.type !== FT.ctgr || hist.cid === "home") {
					if (fp.type) {
						this.pathlist.push(fp);
					} else {
						this.filelist.push(fp);
					}
				}
			}
			// update shared
			if (hist.cid === "shares" && this.isadmin) {
				this.shared = response.data || [];
			}

			// current path & state
			this.curscan = new Date(Date.now());
			this.curcid = hist.cid;
			this.curpuid = hist.puid;
			this.curpath = "";
			this.shrname = "";

			// clear current selected
			eventHub.$emit('select', null);
			// init map card
			this.$refs.mcard.new();
			// update map card
			if (this.filelist.length) {
				const gpslist = [];
				for (const fp of this.filelist) {
					if (fp.latitude && fp.longitude && fp.ntmb === 1) {
						gpslist.push(fp);
					}
					if (pathext(fp.name) === ".gpx") {
						this.$refs.mcard.addgpx(fp);
					}
				}
				this.$refs.mcard.addmarkers(gpslist);
			}
		},

		// opens given folder cleary
		async fetchfolder(hist) {
			const response = await fetchajaxauth("POST", "/api/card/folder", {
				aid: hist.aid, puid: hist.puid, path: hist.path
			});
			traceajax(response);
			if (!response.ok) {
				throw new HttpError(response.status, response.data);
			}

			this.pathlist = [];
			this.filelist = [];
			// update folder settings
			for (const fp of response.data.list || []) {
				if (fp && fp.type !== FT.ctgr) {
					if (fp.type) {
						this.pathlist.push(fp);
					} else {
						this.filelist.push(fp);
					}
				}
			}

			hist.puid = response.data.puid;
			hist.path = response.data.path;
			// current path & state
			this.curscan = new Date(Date.now());
			this.curcid = "";
			this.curpuid = hist.puid;
			this.curpath = hist.path;
			this.shrname = response.data.shrname;

			// clear current selected
			eventHub.$emit('select', null);
			// init map card
			this.$refs.mcard.new();
			// update map card
			if (this.filelist.length) {
				const gpslist = [];
				for (const fp of this.filelist) {
					if (fp.latitude && fp.longitude && fp.ntmb === 1) {
						gpslist.push(fp);
					}
					if (pathext(fp.name) === ".gpx") {
						this.$refs.mcard.addgpx(fp);
					}
				}
				this.$refs.mcard.addmarkers(gpslist);
			}
		},

		async fetchopenroute(hist) {
			if (!hist.cid && !hist.puid && !hist.path) {
				hist.cid = "home";
			}
			if (hist.cid) {
				await this.fetchcategory(hist);
				if (hist.cid === "shares") {
					await this.fetchscanthumbs();
				}
			} else {
				await this.fetchfolder(hist);
				await this.fetchscanthumbs();
			}
			this.seturl();
		},

		async fetchscanthumbs() {
			if (!this.uncached.length) {
				return;
			}

			const response = await fetchjsonauth("POST", "/api/tmb/scn", {
				aid: this.aid,
				puids: this.uncached.map(fp => fp.puid)
			});
			traceajax(response);
			if (!response.ok) {
				throw new HttpError(response.status, response.data);
			}

			// cache folder thumnails
			const curpuid = this.puid;
			(async () => {
				try {
					while (curpuid === this.puid && this.uncached.length) {
						// check cached state loop
						const response = await fetchajaxauth("POST", "/api/tmb/chk", {
							tmbs: this.uncached.map(fp => ({ puid: fp.puid }))
						});
						traceajax(response);
						if (!response.ok) {
							throw new HttpError(response.status, response.data);
						}

						const gpslist = [];
						for (const tp of response.data.tmbs) {
							if (tp.ntmb) {
								for (const fp of this.filelist) {
									if (fp.puid === tp.puid) {
										Vue.set(fp, 'ntmb', tp.ntmb);
										// add gps-item
										if (fp.latitude && fp.longitude && fp.ntmb === 1) {
											gpslist.push(fp);
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
				} catch (e) {
					ajaxfail(e);
				}
			})();
		},

		async fetchshared() {
			const response = await fetchajaxauth("POST", "/api/share/lst", {
				aid: this.aid
			});
			traceajax(response);
			if (!response.ok) {
				throw new HttpError(response.status, response.data);
			}
			this.shared = response.data || [];
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
			if (response.data) { // on ok
				for (let i in this.shared) {
					if (this.shared[i].puid === file.puid) {
						this.shared.splice(i, 1);
						break;
					}
				}
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

		setskin(skinid) {
			for (const v of this.resmodel.skinlist) {
				if (v.id === skinid) {
					$("#skinmodel").attr("href", v.link);
					sessionStorage.setItem('skinid', skinid);
					this.skinid = skinid;
				}
			}
		},

		seticon(iconid) {
			(async () => {
				eventHub.$emit('ajax', +1);
				try {
					for (const v of this.resmodel.iconlist) {
						if (v.id === iconid) {
							await this.fetchicons(v.link);
							sessionStorage.setItem('iconid', iconid);
							this.iconid = iconid;
						}
					}
				} catch (e) {
					ajaxfail(e);
				} finally {
					eventHub.$emit('ajax', -1);
				}
			})();
		},

		seturl() {
			const url = (() => {
				if (this.curcid) {
					return `${(devmode ? "/dev" : "")}/id${this.aid}/ctgr/${this.curcid}/`;
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

		onhome() {
			(async () => {
				eventHub.$emit('ajax', +1);
				try {
					// open route and push history step
					const hist = { cid: "home", aid: this.aid };
					await this.fetchopenroute(hist);
					this.pushhist(hist);
				} catch (e) {
					ajaxfail(e);
				} finally {
					eventHub.$emit('ajax', -1);
				}
			})();
		},

		onback() {
			(async () => {
				eventHub.$emit('ajax', +1);
				try {
					this.histpos--;
					const hist = this.histlist[this.histpos - 1];
					await this.fetchopenroute(hist);
				} catch (e) {
					ajaxfail(e);
				} finally {
					eventHub.$emit('ajax', -1);
				}
			})();
		},

		onforward() {
			(async () => {
				eventHub.$emit('ajax', +1);
				try {
					this.histpos++;
					const hist = this.histlist[this.histpos - 1];
					await this.fetchopenroute(hist);
				} catch (e) {
					ajaxfail(e);
				} finally {
					eventHub.$emit('ajax', -1);
				}
			})();
		},

		onparent() {
			(async () => {
				eventHub.$emit('ajax', +1);
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
					eventHub.$emit('ajax', -1);
				}
			})();
		},

		onrefresh() {
			(async () => {
				eventHub.$emit('ajax', +1);
				try {
					await this.fetchopenroute({ cid: this.curcid, aid: this.aid, puid: this.curpuid, path: this.curpath });
					if (this.isadmin && this.curcid !== "shares") {
						await this.fetchshared(); // get shares
					}
					if (!this.isadmin && this.curcid !== "home") {
						await this.fetchishome();
					}
				} catch (e) {
					ajaxfail(e);
				} finally {
					eventHub.$emit('ajax', -1);
				}
			})();
		},

		onshare(file) {
			(async () => {
				eventHub.$emit('ajax', +1);
				try {
					if (this.isshared(file)) { // should remove share
						await this.fetchsharedel(file);
					} else { // should add share
						await this.fetchshareadd(file);
					}
				} catch (e) {
					ajaxfail(e);
				} finally {
					eventHub.$emit('ajax', -1);
				}
			})();
		},
		onauthcaret() {
			this.showauth = !this.showauth;
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
		incloadcount(count) {
			this.loadcount += count;
		},
		onopen(file) {
			switch (getFileGroup(file)) {
				case FG.dir:
				case FG.packs:
					if (!file.type && pathext(file.name) !== ".iso") { // skip non-iso images
						break;
					}
					if (!file.latency || file.latency > 0) {
						(async () => {
							eventHub.$emit('ajax', +1);
							try {
								// open route and push history step
								const hist = { aid: this.aid };
								if (file.cid) {
									hist.cid = file.cid;
								}
								if (file.puid) {
									hist.puid = file.puid;
								}
								if (file.path) {
									hist.path = file.path;
								}
								await this.fetchopenroute(hist);
								this.pushhist(hist);
							} catch (e) {
								ajaxfail(e);
							} finally {
								eventHub.$emit('ajax', -1);
							}
						})();
					}
					break;
				case FG.image:
					this.$refs.slider.popup(file, this.$refs.fcard.playlist);
					break;
				default:
					const url = mediaurl(file);
					window.open(url, file.name);
			}
		}
	},
	created() {
		eventHub.$on('auth', this.authclosure);
		eventHub.$on('ajax', this.incloadcount);
		eventHub.$on('open', this.onopen);

		auth.signload();
		this.login = auth.login;
		if (devmode && this.isauth) {
			console.log("token:", auth.token);
			console.log("login:", auth.login);
		}
	},
	beforeDestroy() {
		eventHub.$off('auth', this.authclosure);
		eventHub.$off('ajax', this.incloadcount);
		eventHub.$off('open', this.onopen);
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
		if (chunks[0].substr(0, 2) === "id") {
			this.aid = Number(chunks[0].substr(2));
			chunks.shift();
		} else {
			this.aid = 1;
		}

		const hist = { aid: this.aid };
		// get route
		const route = chunks[0];
		chunks.shift();
		if (route === "ctgr") {
			hist.cid = chunks[0];
			chunks.shift();
		} else if (route === "path") {
			hist.path = chunks.join('/');
		} else if (route === "main") {
			hist.cid = "home";
		}
		if (!hist.cid && !hist.path) {
			hist.cid = "home";
		}

		// open route
		(async () => {
			eventHub.$emit('ajax', +1);
			try {
				// open route and push history step
				await this.fetchopenroute(hist);
				if (this.isadmin && hist.cid !== "share") {
					await this.fetchshared(); // get shares
				}
				if (!this.isadmin && hist.cid !== "home") {
					await this.fetchishome();
				}
				this.pushhist(hist);
			} catch (e) {
				ajaxfail(e);
			} finally {
				eventHub.$emit('ajax', -1);
			}
		})();

		// load resources
		(async () => {
			eventHub.$emit('ajax', +1);
			try {
				// load model at first to give an opportunity switch to another skin/iconset on failure
				const response = await fetch("/data/assets/resmodel.json");
				if (!response.ok) {
					throw new HttpError(response.status, { what: "can not load resources model file", when: Date.now(), code: 0 });
				}
				this.resmodel = await response.json();

				const skinid = sessionStorage.getItem('skinid') || this.resmodel.defskinid;
				for (const v of this.resmodel.skinlist) {
					if (v.id === skinid) {
						$("#skinmodel").attr("href", v.link);
						this.skinid = skinid;
					}
				}

				const iconid = sessionStorage.getItem('iconid') || this.resmodel.deficonid;
				for (const v of this.resmodel.iconlist) {
					if (v.id === iconid) {
						await this.fetchicons(v.link);
						this.iconid = iconid;
					}
				}
			} catch (e) {
				ajaxfail(e);
			} finally {
				eventHub.$emit('ajax', -1);
			}
		})();
	}
});

$(document).ready(() => {
	$('.preloader-lock').hide("fast");
	$('#app').show("fast");
});

// The End.
