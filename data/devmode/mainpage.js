"use strict";

const skinlist = [
	{
		name: "daylight",
		link: "/data/skin/daylight.css"
	},
	{
		name: "blue",
		link: "/data/skin/blue.css"
	},
	{
		name: "neon",
		link: "/data/skin/neon.css"
	},
	{
		name: "cup of coffee",
		link: "/data/skin/cup-of-coffee.css"
	},
	{
		name: "coffee beans",
		link: "/data/skin/coffee-beans.css"
	},
	{
		name: "old monitor",
		link: "/data/skin/old-monitor.css"
	}
];

// File types
const FT = {
	cat: -3,
	drive: -2,
	dir: -1,
	file: 0,
	wave: 1,
	flac: 2,
	mp3: 3,
	ogg: 4,
	mp4: 5,
	webm: 6,
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
	rar: 24
};

// File groups
const FG = {
	other: 0,
	music: 1,
	video: 2,
	image: 3,
	books: 4,
	texts: 5,
	store: 6,
	dir: 7
};

const FTtoFG = {
	[FT.cat]: FG.dir,
	[FT.drive]: FG.dir,
	[FT.dir]: FG.dir,
	[FT.file]: FG.other,
	[FT.wave]: FG.music,
	[FT.flac]: FG.music,
	[FT.mp3]: FG.music,
	[FT.ogg]: FG.video,
	[FT.mp4]: FG.video,
	[FT.webm]: FG.video,
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
	[FT.rar]: FG.store
};

// File viewers
const FV = {
	none: 0,
	music: 1,
	video: 2,
	image: 3
};

const FTtoFV = {
	[FT.cat]: FV.none,
	[FT.drive]: FV.none,
	[FT.dir]: FV.none,
	[FT.file]: FV.none,
	[FT.wave]: FV.music,
	[FT.flac]: FV.music,
	[FT.mp3]: FV.music,
	[FT.ogg]: FV.video,
	[FT.mp4]: FV.video,
	[FT.webm]: FV.video,
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
	[FT.rar]: FV.none
};

// Drive state
const DS = {
	yellow: 3000,
	red: 10000
};

const shareprefix = "/file/";

const geticonname = file => {
	switch (file.type) {
		case FT.cat:
			switch (file.cid) {
				case "drives":
					return "drives";
				case "shares":
					return "shares";
				default:
					return "folder-close";
			}
		case FT.drive:
			if (file.latency < 0) {
				return "drive-off";
			} else if (file.latency < DS.yellow) {
				return "drive";
			} else if (file.latency < DS.red) {
				return "drive-yellow";
			} else {
				return "drive-red";
			}
		case FT.dir:
			const suff = app.shrname.length ? "-pub" : "";
			if (file.scan) {
				let fnum = 0;
				const fg = file.fgrp;
				for (let n of fg) {
					fnum += n;
				}
				if (!fnum) {
					return "folder-empty" + suff;
				} else if (fg[FG.music] / fnum > 0.5) {
					return "folder-mp3" + suff;
				} else if (fg[FG.video] / fnum > 0.5) {
					return "folder-movies" + suff;
				} else if (fg[FG.image] / fnum > 0.5) {
					return "folder-photo" + suff;
				} else if (fg[FG.books] / fnum > 0.5) {
					return "folder-doc" + suff;
				} else if (fg[FG.texts] / fnum > 0.5) {
					return "folder-doc" + suff;
				} else if (fg[FG.dir] / fnum > 0.5) {
					return "folder-sub" + suff;
				} else if ((fg[FG.music] + fg[FG.video] + fg[FG.image]) / fnum > 0.5) {
					return "folder-media" + suff;
				} else {
					return "folder-empty" + suff;
				}
			} else {
				return "folder-close" + suff;
			}
		case FT.wave:
			return "doc-wave";
		case FT.flac:
			return "doc-flac";
		case FT.mp3:
			return "doc-mp3";
		case FT.ogg:
			return "doc-music";
		case FT.mp4:
			return "doc-mp4";
		case FT.webm:
			return "doc-movie";
		case FT.tga:
		case FT.bmp:
		case FT.dds:
			return "doc-bitmap";
		case FT.tiff:
		case FT.jpeg:
			return "doc-jpeg";
		case FT.gif:
			return "doc-gif";
		case FT.png:
			return "doc-png";
		case FT.webp:
			return "doc-webp";
		case FT.psd:
			return "doc-psd";
		case FT.pdf:
			return "doc-pdf";
		case FT.html:
			return "doc-html";
		case FT.text:
			return "doc-text";
		case FT.scr:
			return "doc-script";
		case FT.cfg:
			return "doc-config";
		case FT.log:
			return "doc-log";
		case FT.cab:
			return "doc-cab";
		case FT.zip:
			return "doc-zip";
		case FT.rar:
			return "doc-rar";
		default: // File and others
			return "doc-file";
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
		skinlink: "", // URL of skin CSS
		skinlist: skinlist,
		showauth: false, // display authorization form
		isauth: false, // is authorized
		authid: 0, // authorized ID
		aid: 0, // account ID

		loadcount: 0, // ajax working request count
		shared: [], // list of shared folders

		// history
		histpos: 0, // position in history stack
		histlist: [], // history stack

		// current opened folder data
		route: "path", // current route
		pathlist: [], // list of subfolders properties in current folder
		filelist: [], // list of files properties in current folder
		curscan: new Date(), // time of last scanning of current folder
		curpuid: "", // current folder PUID
		curpath: "", // current folder path and path state
		curstate: 2, // current folder path state
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
			switch (this.route) {
				case "path":
					const arr = this.curpath.split('/');
					const base = arr.pop() || arr.pop();
					return !arr.length && this.shrname || base;
				case "home":
					return 'home page';
				case "drive":
					return 'drives list';
				case "share":
					return 'shares list';
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
			return this.route === "home";
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
				switch (hist.route) {
					case "path":
						return `back to /id${hist.aid}/path/${hist.path}`;
					case "home":
						return "back to home";
					case "drive":
						return "back to drives";
					case "share":
						return "back to shares";
				}
			}
			return "go back";
		},
		hintforward() {
			if (this.histpos < this.histlist.length) {
				const hist = this.histlist[this.histpos];
				switch (hist.route) {
					case "path":
						return `forward to /id${hist.aid}/path/${hist.path}`;
					case "home":
						return "forward to home";
					case "drive":
						return "forward to drives";
					case "share":
						return "forward to shares";
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
		// opens given folder cleary
		async fetchfolder(hist) {
			const response = await fetchajaxauth("POST", "/api/card/folder", {
				aid: hist.aid, puid: hist.puid || "", path: hist.path || ""
			});
			traceajax(response);

			this.pathlist = [];
			this.filelist = [];
			// init map card
			this.$refs.mapcard.new();

			if (!response.ok) {
				throw new HttpError(response.status, response.data);
			}
			this.route = "path";
			this.curscan = new Date(Date.now());
			// update folder settings
			for (const fp of response.data.list || []) {
				if (fp) {
					if (fp.type < 0) {
						this.pathlist.push(fp);
					} else {
						this.filelist.push(fp);
					}
				}
			}
			// current path & state
			this.curpuid = hist.puid = response.data.puid;
			this.curpath = hist.path = response.data.path;
			this.curstate = response.data.state;
			this.shrname = response.data.shrname;
			// update map card
			const gpslist = [];
			for (const fp of this.filelist) {
				if (fp.latitude && fp.longitude && fp.ntmb === 1) {
					gpslist.push(fp);
				}
			}
			this.$refs.mapcard.addmarkers(gpslist);
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
			this.shared = response.data;
		},

		async fetchhome() {
			const response = await fetchajaxauth("POST", "/api/card/home", {
				aid: this.aid
			});
			traceajax(response);
			if (!response.ok) {
				throw new HttpError(response.status, response.data);
			}
			this.pathlist = response.data; // all items are categories
			this.filelist = [];
			// init map card
			this.$refs.mapcard.new();

			this.route = "home";
			this.curscan = new Date(Date.now());
			// current path & state
			this.curpuid = "";
			this.curpath = "";
			this.curstate = "";
			this.shrname = "";
		},

		async fetchdrivepage() {
			const response = await fetchajaxauth("POST", "/api/drive/lst", {
				aid: this.aid
			});
			traceajax(response);
			if (!response.ok) {
				throw new HttpError(response.status, response.data);
			}

			this.pathlist = response.data; // all items are drives
			this.filelist = [];
			// init map card
			this.$refs.mapcard.new();

			this.route = "drive";
			this.curscan = new Date(Date.now());
			// current path & state
			this.curpuid = "";
			this.curpath = "";
			this.curstate = "";
			this.shrname = "";
		},

		async fetchsharepage() {
			const response = await fetchajaxauth("POST", "/api/share/lst", {
				aid: this.aid
			});
			traceajax(response);
			if (!response.ok) {
				throw new HttpError(response.status, response.data);
			}

			this.pathlist = [];
			this.filelist = [];
			// init map card
			this.$refs.mapcard.new();

			this.route = "share";
			this.curscan = new Date(Date.now());
			// update folder settings
			for (const fp of response.data) {
				if (fp) {
					if (fp.type < 0) {
						this.pathlist.push(fp);
					} else {
						this.filelist.push(fp);
					}
				}
			}
			// current path & state
			this.curpuid = "";
			this.curpath = "";
			this.curstate = "";
			this.shrname = "";
			// update shared
			this.shared = response.data;
			// update map card
			const gpslist = [];
			for (const fp of this.filelist) {
				if (fp.latitude && fp.longitude && fp.ntmb === 1) {
					gpslist.push(fp);
				}
			}
			this.$refs.mapcard.addmarkers(gpslist);
		},

		async fetchopenroute(hist) {
			switch (hist.route) {
				case "path":
					await this.fetchfolder(hist);
					await this.fetchscanthumbs();
					break;
				case "home":
					await this.fetchhome();
					break;
				case "drive":
					await this.fetchdrivepage();
					break;
				case "share":
					await this.fetchsharepage();
					await this.fetchscanthumbs();
					break;
				default:
					throw new Error("try to open undefined route");
			}
			this.seturl();
		},

		async fetchshareadd(file) {
			const response = await fetchajaxauth("POST", "/api/share/add", {
				aid: this.aid,
				puid: file.puid
			});
			traceajax(response);
			if (response.ok) {
				if (response.data) {
					this.shared.push(file);
				}
			} else if (response.status === 404) { // Not Found
				// remove file from folder
				if (FTtoFG[file.type] === FG.dir) {
					this.pathlist.splice(this.pathlist.findIndex(elem => elem === file), 1);
				} else {
					this.filelist.splice(this.filelist.findIndex(elem => elem === file), 1);
				}
				throw new HttpError(404, response.data);
			} else {
				throw new HttpError(response.status, response.data);
			}
		},

		async fetchsharedel(file) {
			const response = await fetchajaxauth("DELETE", "/api/share/del", {
				aid: this.aid,
				puid: file.puid
			});
			traceajax(response);
			if (response.ok) {
				const ok = response.data;
				// update folder settings
				if (ok) {
					for (let i in this.shared) {
						if (this.shared[i].puid === file.puid) {
							this.shared.splice(i, 1);
							break;
						}
					}

					// remove item from root folder
					if (!this.curpath) {
						if (FTtoFG[file.type] === FG.dir) {
							this.pathlist.splice(this.pathlist.findIndex(elem => elem === file), 1);
						} else {
							this.filelist.splice(this.filelist.findIndex(elem => elem === file), 1);
						}
					}
				}
			} else if (xhr.status === 404) { // Not Found
				// remove file from folder
				if (FTtoFG[file.type] === FG.dir) {
					this.pathlist.splice(this.pathlist.findIndex(elem => elem === file), 1);
				} else {
					this.filelist.splice(this.filelist.findIndex(elem => elem === file), 1);
				}
				throw new HttpError(404, response.data);
			} else {
				throw new HttpError(response.status, response.data);
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

		setskin(skinlink) {
			this.skinlink = skinlink;
			$("#skinlink").attr("href", skinlink);
			sessionStorage.setItem('skinlink', skinlink);
		},

		seturl() {
			const url = (() => {
				switch (this.route) {
					case "path":
						return this.curlongurl;
					case "home":
						return `${(devmode ? "/dev" : "")}/id${this.aid}/home/`;
					case "drive":
						return `${(devmode ? "/dev" : "")}/id${this.aid}/drive/`;
					case "share":
						return `${(devmode ? "/dev" : "")}/id${this.aid}/share/`;
				}
			})();
			window.history.replaceState(null, this.curpath, url);
		},

		// push item into folders history
		pushroute(hist) {
			this.histlist.splice(this.histpos);
			this.histlist.push(hist);
			this.histpos = this.histlist.length;
		},

		onhome() {
			(async () => {
				ajaxcc.emit('ajax', +1);
				try {
					// open route and push history step
					const hist = { route: "home", aid: this.aid };
					await this.fetchopenroute(hist);
					this.pushroute(hist);
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
					const hist = { route: "home", aid: this.aid, path: path };
					await this.fetchopenroute(hist);
					this.pushroute(hist);
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
					await this.fetchopenroute({ route: this.route, aid: this.aid, path: this.curpath });
					if (this.isauth && this.route !== "share") {
						await this.fetchshared(); // get shares
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
			if (!file.offline) {
				(async () => {
					ajaxcc.emit('ajax', +1);
					try {
						// open route and push history step
						const hist = {
							route: "path",
							aid: this.aid,
							puid: file.puid || "",
							path: file.path || ""
						};
						if (file.type === FT.cat) {
							switch (file.cid) {
								case "drives":
									hist.route = "drive";
									break;
								case "shares":
									hist.route = "share";
									break;
								default:
									return;
							}
						}
						await this.fetchopenroute(hist);
						this.pushroute(hist);
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
		this.skinlink = sessionStorage.getItem('skinlink') || "/data/skin/neon.css";
		$("#skinlink").attr("href", this.skinlink);

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
			this.aid = 0;
		}

		// get route
		const route = chunks[0];
		chunks.shift();
		if (!route) {
			route = "home";
		}

		// open route
		(async () => {
			ajaxcc.emit('ajax', +1);
			try {
				// open route and push history step
				const hist = { route: route, aid: this.aid, path: chunks.join('/') };
				await this.fetchopenroute(hist);
				if (this.isauth && route !== "share") {
					await this.fetchshared(); // get shares
				}
				this.pushroute(hist);
			} catch (e) {
				ajaxfail(e);
			} finally {
				ajaxcc.emit('ajax', -1);
			}
		})();
	}
});

$(document).ready(() => {
	$('.preloader').hide("fast");
	$('#app').show("fast");
});

// The End.
