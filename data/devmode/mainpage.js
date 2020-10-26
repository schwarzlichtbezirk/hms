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
	drive: -2,
	dir: -1,
	file: 0,
	wave: 1,
	flac: 2,
	mp3: 3,
	ogg: 4,
	mp4: 5,
	webm: 6,
	photo: 7,
	tga: 8,
	bmp: 9,
	dds: 10,
	tiff: 11,
	jpeg: 12,
	gif: 13,
	png: 14,
	webp: 15,
	psd: 16,
	pdf: 17,
	html: 18,
	text: 19,
	scr: 20,
	cfg: 21,
	log: 22,
	cab: 23,
	zip: 24,
	rar: 25
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
	[FT.drive]: FG.dir,
	[FT.dir]: FG.dir,
	[FT.file]: FG.other,
	[FT.wave]: FG.music,
	[FT.flac]: FG.music,
	[FT.mp3]: FG.music,
	[FT.ogg]: FG.video,
	[FT.mp4]: FG.video,
	[FT.webm]: FG.video,
	[FT.photo]: FG.image,
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
	[FT.drive]: FV.none,
	[FT.dir]: FV.none,
	[FT.file]: FV.none,
	[FT.wave]: FV.music,
	[FT.flac]: FV.music,
	[FT.mp3]: FV.music,
	[FT.ogg]: FV.video,
	[FT.mp4]: FV.video,
	[FT.webm]: FV.video,
	[FT.photo]: FV.image,
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

const root = { name: "", path: "", size: 0, time: 0, type: FT.dir, puid: "" };

const shareprefix = "/file/";

const geticonname = file => {
	switch (file.type) {
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
		case FT.photo:
			return "doc-photo";
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

const hashfileurl = file => `/id${app.aid}/file/${file.puid}`;
const hashpathurl = file => `/id${app.aid}/path/${file.puid}`;

const showmsgbox = (title, body) => {
	const dlg = $("#msgbox");
	dlg.find(".modal-title").html(title);
	dlg.find(".modal-body").html(body);
	dlg.modal("show");
};

const ajaxfail = what => {
	showmsgbox(
		"Server unavailable",
		"Server is currently not available, action can not be done now."
	);
	console.error(what);
};

const onerr404 = () => {
	showmsgbox(
		"Invalid path",
		"Specified path cannot be accessed now."
	);
};

let app = new Vue({
	el: '#app',
	template: '#app-tpl',
	data: {
		skinlink: "", // URL of skin CSS
		skinlist: skinlist,
		showauth: false, // display authorization form
		isauth: false, // is authorized
		authid: 0, // authorized ID
		aid: 0, // account ID
		login: "", // authorization login
		password: "", // authorization password
		namestate: 0, // -1 invalid login, 0 ambiguous, 1 valid login
		passstate: 0, // -1 invalid password, 0 ambiguous, 1 valid password

		loadcount: 0, // ajax working request count
		shared: [], // list of shared folders

		// current opened folder data
		pathlist: [], // list of subfolders properties in current folder
		filelist: [], // list of files properties in current folder
		curprop: root, // current folder properties
		curscan: new Date(), // time of last scanning of current folder
		curpath: "", // current path and path state
		curstate: 2, // current path state
		shrname: "", // current path share name
		histpos: 0, // position in history stack
		histlist: [] // history stack
	},
	computed: {
		// is it authorized or running on localhost
		isadmin() {
			return this.isauth || window.location.hostname === 'localhost' || window.location.hostname === '127.0.0.1';
		},
		// current page URL
		cururl() {
			return `${(devmode ? "/dev" : "")}/id${this.aid}/path/${this.curpath}`;
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
				lst[0].name = this.shrname;
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
			return !this.curpath;
		},
		disback() {
			return this.histpos < 2;
		},
		disforward() {
			return this.histpos > this.histlist.length - 1;
		},
		disparent() {
			return !this.curpathway.length;
		},

		textauthcaret() {
			return this.showauth ? 'arrow_right' : 'arrow_left';
		},
		clsname() {
			return !this.namestate ? ''
				: this.namestate === -1 ? 'is-invalid' : 'is-valid';
		},
		clspass() {
			return !this.passstate ? ''
				: this.passstate === -1 ? 'is-invalid' : 'is-valid';
		},

		// buttons hints
		hintback() {
			if (this.histpos < 2) {
				return "go back";
			} else {
				let name = this.histlist[this.histpos - 2].name;
				if (!name) {
					name = "root folder";
				}
				return "go back to " + name;
			}
		},
		hintforward() {
			if (this.histpos > this.histlist.length - 1) {
				return "go forward";
			} else {
				let name = this.histlist[this.histpos].name;
				if (!name) {
					name = "root folder";
				}
				return "go forward to " + name;
			}
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
		fetchfolder(file) {
			return fetchajaxauth("POST", "/api/folder", {
				aid: this.aid,
				puid: file.puid
			}).then(response => {
				traceajax(response);

				this.pathlist = [];
				this.filelist = [];
				this.curprop = file;
				// init map card
				this.$refs.mapcard.new();

				if (response.ok) {
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
					this.curpath = response.data.path;
					this.curstate = response.data.state;
					this.shrname = response.data.shrname;
					this.seturl();
					// update map card
					const gpslist = [];
					for (const fp of this.filelist) {
						if (fp.latitude && fp.longitude && fp.ntmb === 1) {
							gpslist.push(fp);
						}
					}
					this.$refs.mapcard.addmarkers(gpslist);
				} else if (response.status === 401) { // Unauthorized
					onerr404();
				} else if (response.status === 404) { // Not Found
					onerr404();
				}

				// cache folder thumnails
				if (this.uncached.length) {
					// check cached state loop
					let chktmb;
					chktmb = () => {
						const tmbs = [];
						for (const fp of this.uncached) {
							tmbs.push({ puid: fp.puid });
						}
						fetchajaxauth("POST", "/api/tmb/chk", {
							tmbs: tmbs
						}).then(response => {
							traceajax(response);
							if (response.ok) {
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
								this.$refs.mapcard.addmarkers(gpslist); // update map card
								if (this.uncached.length) {
									setTimeout(chktmb, 1500); // wait and run again
								}
							}
						});
					};
					// gets thumbs
					setTimeout(chktmb, 600);

					const puids = [];
					for (const fp of this.uncached) {
						puids.push(fp.puid);
					}
					fetchajaxauth("POST", "/api/tmb/scn", {
						aid: this.aid,
						puids: puids
					}).then(response => {
						traceajax(response);
					});
				}
			});
		},

		// opens given folder and push history step
		fetchopenfolder(file) {
			return this.fetchfolder(file)
				.then(() => {
					// update folder history
					if (this.histpos < this.histlist.length) {
						this.histlist.splice(this.histpos, this.histlist.length - this.histpos);
					}
					this.histlist.push(file);
					this.histpos = this.histlist.length;
				});
		},

		fetchsharelist() {
			return fetchajaxauth("POST", "/api/share/lst", {
				aid: this.aid
			}).then(response => {
				traceajax(response);
				if (response.ok) {
					this.shared = response.data || [];
				}
			});
		},

		fetchshareadd(file) {
			return fetchajaxauth("POST", "/api/share/add", {
				aid: this.aid,
				puid: file.puid
			}).then(response => {
				traceajax(response);
				if (response.ok) {
					if (response.data) {
						this.shared.push(file);
					}
				} else if (response.status === 404) { // Not Found
					onerr404();
					// remove file from folder
					if (FTtoFG[file.type] === FG.dir) {
						this.pathlist.splice(this.pathlist.findIndex(elem => elem === file), 1);
					} else {
						this.filelist.splice(this.filelist.findIndex(elem => elem === file), 1);
					}
				}
			});
		},

		fetchsharedel(file) {
			return fetchajaxauth("DELETE", "/api/share/del", {
				aid: this.aid,
				puid: file.puid
			}).then(response => {
				traceajax(response);
				if (response.ok) {
					const ok = response.data;
					// update folder settings
					if (ok) {
						const isdir = FTtoFG[file.type] === FG.dir;
						if (isdir) {
							for (let i in this.shared) {
								if (this.shared[i].puid === file.puid) {
									this.shared.splice(i, 1);
									break;
								}
							}
						}

						if (this.curprop.path) {
							// adjust file path to current path
							file.path = this.curprop.path + file.name;
							if (isdir) {
								file.path += '/';
							}
						} else {
							// remove item from root folder
							if (isdir) {
								this.pathlist.splice(this.pathlist.findIndex(elem => elem === file), 1);
							} else {
								this.filelist.splice(this.filelist.findIndex(elem => elem === file), 1);
							}
						}
					}
				} else if (xhr.status === 404) { // Not Found
					onerr404();
					// remove file from folder
					if (FTtoFG[file.type] === FG.dir) {
						this.pathlist.splice(this.pathlist.findIndex(elem => elem === file), 1);
					} else {
						this.filelist.splice(this.filelist.findIndex(elem => elem === file), 1);
					}
				}
			});
		},

		isshared(file) {
			for (const shr of this.shared) {
				if (shr.puid === file.puid) {
					return true;
				}
			}
			return false;
		},

		seturl() {
			window.history.replaceState(null, this.curpath, this.cururl);
		},

		setskin(skinlink) {
			this.skinlink = skinlink;
			$("#skinlink").attr("href", skinlink);
			sessionStorage.setItem('skinlink', skinlink);
		},

		onhome() {
			ajaxcc.emit('ajax', +1);
			this.fetchopenfolder(root)
				.catch(ajaxfail).finally(() => ajaxcc.emit('ajax', -1));
		},

		onback() {
			this.histpos--;
			const file = this.histlist[this.histpos - 1];

			ajaxcc.emit('ajax', +1);
			this.fetchfolder(file)
				.catch(ajaxfail).finally(() => ajaxcc.emit('ajax', -1));
		},

		onforward() {
			this.histpos++;
			const file = this.histlist[this.histpos - 1];

			ajaxcc.emit('ajax', +1);
			this.fetchfolder(file)
				.catch(ajaxfail).finally(() => ajaxcc.emit('ajax', -1));
		},

		onparent() {
			const file = this.curpathway.length
				? this.curpathway[this.curpathway.length - 1]
				: root;

			ajaxcc.emit('ajax', +1);
			this.fetchopenfolder(file)
				.catch(ajaxfail).finally(() => ajaxcc.emit('ajax', -1));
		},

		onrefresh() {
			const file = this.curprop;

			ajaxcc.emit('ajax', +1);
			this.fetchfolder(file)
				.then(() => this.fetchsharelist()) // get shares
				.catch(ajaxfail).finally(() => ajaxcc.emit('ajax', -1));
		},

		onshare(file) {
			if (this.isshared(file)) { // should remove share
				ajaxcc.emit('ajax', +1);
				this.fetchsharedel(file)
					.catch(ajaxfail).finally(() => ajaxcc.emit('ajax', -1));
			} else { // should add share
				ajaxcc.emit('ajax', +1);
				this.fetchshareadd(file)
					.catch(ajaxfail).finally(() => ajaxcc.emit('ajax', -1));
			}
		},

		onpathopen(file) {
			if (!file.offline) {
				ajaxcc.emit('ajax', +1);
				this.fetchopenfolder(file)
					.catch(ajaxfail).finally(() => ajaxcc.emit('ajax', -1));
			}
		},

		onauthcaret() {
			this.showauth = !this.showauth;
		},
		onauthchange() {
			this.namestate = 0;
			this.passstate = 0;
		},
		onlogin() {
			ajaxcc.emit('ajax', +1);
			fetchajax("POST", "/api/pubkey").then(response => {
				traceajax(response);
				if (response.ok) {
					// github.com/emn178/js-sha256
					const hash = sha256.hmac.create(response.data);
					hash.update(this.password);
					return fetchajax("POST", "/api/signin", {
						name: this.login,
						pubk: response.data,
						hash: hash.digest()
					});
				}
				return Promise.reject();
			}).then(response => {
				traceajax(response);
				if (response.status === 200) {
					auth.signin(response.data, this.login);
					this.namestate = 1;
					this.passstate = 1;
					this.onrefresh();
					this.seturl();
				} else if (response.status === 403) { // Forbidden
					auth.signout();
					switch (response.data.code) {
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
				}
			}).catch(ajaxfail).finally(() => ajaxcc.emit('ajax', -1));
		},
		onlogout() {
			auth.signout();
			this.namestate = 0;
			this.passstate = 0;
			this.onrefresh();
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
		chunks.shift(); // remove first empty element
		if (chunks[0] === "dev") {
			chunks.shift(); // cut "dev" prefix
		}

		// get account id
		if (chunks[0].substr(0, 2) === "id") {
			this.aid = Number(chunks[0].substr(2));
			chunks.shift();
		} else {
			this.aid = 0;
		}

		// open path
		if (chunks[0] === "path") {
			chunks.shift();
			if (chunks[chunks.length - 1].length > 0) {
				chunks.push(""); // bring it to true path
			}
			const file = {
				name: chunks[chunks.length - 2],
				path: chunks.join("/"),
				size: 0, time: 0, type: FT.dir
			};

			ajaxcc.emit('ajax', +1);
			this.fetchopenfolder(file)
				.then(() => this.fetchsharelist()) // get shares
				.catch(ajaxfail).finally(() => ajaxcc.emit('ajax', -1));
		} else {
			ajaxcc.emit('ajax', +1);
			this.fetchopenfolder(root)
				.catch(ajaxfail).finally(() => ajaxcc.emit('ajax', -1));
		}
	}
});

$(document).ready(() => {
	$('.preloader').hide("fast");
	$('#app').show("fast");
});

// The End.
