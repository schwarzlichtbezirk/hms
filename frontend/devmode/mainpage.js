"use strict";

let iconmapping = imempty;
let thumbmode = true;

// icon mapping event model
const iconev = extend({}, makeeventmodel());

// File types
const FT = {
	ctgr: -3,
	drive: -2,
	dir: -1,
	file: 0,
	mp4: 1,
	webm: 2,
	wave: 3,
	flac: 4,
	mp3: 5,
	ogg: 6,
	tga: 7,
	bmp: 8,
	dds: 9,
	tiff: 10,
	jpeg: 11,
	gif: 12,
	png: 13,
	webp: 14,
	psd: 15,
	pdf: 16,
	html: 17,
	text: 18,
	scr: 19,
	cfg: 20,
	log: 21,
	cab: 22,
	zip: 23,
	rar: 24,
	tar: 25,
	disk: 26
};

const FTN = [
	"file", // 0
	"mp4", // 1
	"webm", // 2
	"wave", // 3
	"flac", // 4
	"mp3", // 5
	"ogg", // 6
	"tga", // 7
	"bmp", // 8
	"dds", // 9
	"tiff", // 10
	"jpeg", // 11
	"gif", // 12
	"png", // 13
	"webp", // 14
	"psd", // 15
	"pdf", // 16
	"html", // 17
	"text", // 18
	"scr", // 19
	"cfg", // 20
	"log", // 21
	"cab", // 22
	"zip", // 23
	"rar", // 24
	"tar", // 25
	"disk" // 26
];

// File groups
const FG = {
	other: 0,
	video: 1,
	audio: 2,
	image: 3,
	books: 4,
	texts: 5,
	store: 6,
	dir: 7
};

const FTtoFG = {
	[FT.ctgr]: FG.dir,
	[FT.drive]: FG.dir,
	[FT.dir]: FG.dir,
	[FT.file]: FG.other,
	[FT.ogg]: FG.video,
	[FT.mp4]: FG.video,
	[FT.webm]: FG.video,
	[FT.wave]: FG.audio,
	[FT.flac]: FG.audio,
	[FT.mp3]: FG.audio,
	[FT.tga]: FG.image,
	[FT.bmp]: FG.image,
	[FT.dds]: FG.image,
	[FT.tiff]: FG.image,
	[FT.jpeg]: FG.image,
	[FT.gif]: FG.image,
	[FT.png]: FG.image,
	[FT.webp]: FG.image,
	[FT.psd]: FG.image,
	[FT.pdf]: FG.books,
	[FT.html]: FG.books,
	[FT.text]: FG.texts,
	[FT.scr]: FG.texts,
	[FT.cfg]: FG.texts,
	[FT.log]: FG.texts,
	[FT.cab]: FG.store,
	[FT.zip]: FG.store,
	[FT.rar]: FG.store,
	[FT.tar]: FG.store,
	[FT.disk]: FG.store
};

// File viewers
const FV = {
	none: 0,
	video: 1,
	audio: 2,
	image: 3
};

const FTtoFV = {
	[FT.ctgr]: FV.none,
	[FT.drive]: FV.none,
	[FT.dir]: FV.none,
	[FT.file]: FV.none,
	[FT.ogg]: FV.video,
	[FT.mp4]: FV.video,
	[FT.webm]: FV.video,
	[FT.wave]: FV.audio,
	[FT.flac]: FV.audio,
	[FT.mp3]: FV.audio,
	[FT.tga]: FV.image,
	[FT.bmp]: FV.image,
	[FT.dds]: FV.image,
	[FT.tiff]: FV.image,
	[FT.jpeg]: FV.image,
	[FT.gif]: FV.image,
	[FT.png]: FV.image,
	[FT.webp]: FV.image,
	[FT.psd]: FV.image,
	[FT.pdf]: FV.none,
	[FT.html]: FV.none,
	[FT.text]: FV.none,
	[FT.scr]: FV.none,
	[FT.cfg]: FV.none,
	[FT.log]: FV.none,
	[FT.cab]: FV.none,
	[FT.zip]: FV.none,
	[FT.rar]: FV.none,
	[FT.tar]: FV.none,
	[FT.disk]: FV.none
};

// Drive state
const DS = {
	yellow: 3000,
	red: 10000
};

const shareprefix = "/file/";

const geticonpath = (file, im, shr) => {
	const org = shr ? im.shared : im.private;
	const alt = shr ? im.private : im.shared;
	switch (file.type) {
		case FT.ctgr:
			return {
				org: org.cid[file.cid] || org.cid.cid,
				alt: alt.cid[file.cid] || alt.cid.cid
			};
		case FT.drive:
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
				} else if (fg[FG.store] / fnum > 0.5) {
					return { org: org.folder.store, alt: alt.folder.store };
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
			return {
				org: org.file[FTN[file.type]] || org.file.file,
				alt: alt.file[FTN[file.type]] || alt.file.file
			};
	}
};

const encode = (uri) => encodeURI(uri).replace('#', '%23').replace('&', '%26').replace('+', '%2B');

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
					"Unauthorized",
					"Action can be done only after authorization."
				);
				return;
			case 403: // Forbidden
				msgbox(
					"404 resource forbidden",
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
				ajaxcc.emit('ajax', +1);
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
					ajaxcc.emit('ajax', -1);
				}
			})();
		},
		onlogout() {
			auth.signout();
			this.namestate = 0;
			this.passstate = 0;
			this.$emit('refresh');
		}
	},
	mounted() {
		this._authclosure = is => {
			this.isauth = is;
			if (is) {
				this.login = auth.login;
			}
		};
		auth.on('auth', this._authclosure);
	},
	beforeDestroy() {
		auth.off('auth', this._authclosure);
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
		aid: 0, // account ID
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
				path += fn + '/';
				lst.push({
					name: fn,
					path: path
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
					if (fp.type < 0) {
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

			// init map card
			this.$refs.mapcard.new();
			// update map card
			if (this.filelist.length) {
				const gpslist = [];
				for (const fp of this.filelist) {
					if (fp.latitude && fp.longitude && fp.ntmb === 1) {
						gpslist.push(fp);
					}
				}
				this.$refs.mapcard.addmarkers(gpslist);
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
					if (fp.type < 0) {
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

			// init map card
			this.$refs.mapcard.new();
			// update map card
			if (this.filelist.length) {
				const gpslist = [];
				for (const fp of this.filelist) {
					if (fp.latitude && fp.longitude && fp.ntmb === 1) {
						gpslist.push(fp);
					}
				}
				this.$refs.mapcard.addmarkers(gpslist);
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
						this.$refs.mapcard.addmarkers(gpslist);
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
				ajaxcc.emit('ajax', +1);
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
					ajaxcc.emit('ajax', -1);
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
				ajaxcc.emit('ajax', +1);
				try {
					// open route and push history step
					const hist = { cid: "home", aid: this.aid };
					await this.fetchopenroute(hist);
					this.pushhist(hist);
				} catch (e) {
					ajaxfail(e);
				} finally {
					ajaxcc.emit('ajax', -1);
				}
			})();
		},

		onback() {
			(async () => {
				ajaxcc.emit('ajax', +1);
				try {
					this.histpos--;
					const hist = this.histlist[this.histpos - 1];
					await this.fetchopenroute(hist);
				} catch (e) {
					ajaxfail(e);
				} finally {
					ajaxcc.emit('ajax', -1);
				}
			})();
		},

		onforward() {
			(async () => {
				ajaxcc.emit('ajax', +1);
				try {
					this.histpos++;
					const hist = this.histlist[this.histpos - 1];
					await this.fetchopenroute(hist);
				} catch (e) {
					ajaxfail(e);
				} finally {
					ajaxcc.emit('ajax', -1);
				}
			})();
		},

		onparent() {
			(async () => {
				ajaxcc.emit('ajax', +1);
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
					ajaxcc.emit('ajax', -1);
				}
			})();
		},

		onrefresh() {
			(async () => {
				ajaxcc.emit('ajax', +1);
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
					ajaxcc.emit('ajax', -1);
				}
			})();
		},

		onshare(file) {
			(async () => {
				ajaxcc.emit('ajax', +1);
				try {
					if (this.isshared(file)) { // should remove share
						await this.fetchsharedel(file);
					} else { // should add share
						await this.fetchshareadd(file);
					}
				} catch (e) {
					ajaxfail(e);
				} finally {
					ajaxcc.emit('ajax', -1);
				}
			})();
		},

		onpathopen(file) {
			if (!file.latency || file.latency > 0) {
				(async () => {
					ajaxcc.emit('ajax', +1);
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
						ajaxcc.emit('ajax', -1);
					}
				})();
			}
		},
		onauthcaret() {
			this.showauth = !this.showauth;
		}
	},
	mounted() {
		auth.on('auth', is => {
			this.isauth = is;
			if (is) {
				const claims = auth.claims();
				if (claims && 'aid' in claims) {
					this.authid = claims.aid;
					this.aid = claims.aid;
				}
			}
		});
		ajaxcc.on('ajax', count => this.loadcount += count);

		auth.signload();
		this.login = auth.login;
		if (devmode && this.isauth) {
			console.log("token:", auth.token);
			console.log("login:", auth.login);
		}

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

		// get account id
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
			ajaxcc.emit('ajax', +1);
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
				ajaxcc.emit('ajax', -1);
			}
		})();

		// load resources
		(async () => {
			ajaxcc.emit('ajax', +1);
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
				ajaxcc.emit('ajax', -1);
			}
		})();
	}
});

$(document).ready(() => {
	$('.preloader-lock').hide("fast");
	$('#app').show("fast");
});

// The End.
