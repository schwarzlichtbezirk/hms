"use strict";

//@ sourceMappingURL=mainpage.min.map

// File types
const FT = {
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
	gif: 10,
	png: 11,
	jpeg: 12,
	tiff: 13,
	webp: 14,
	pdf: 15,
	html: 16,
	text: 17,
	scr: 18,
	cfg: 19,
	log: 20,
	cab: 21,
	zip: 22,
	rar: 23
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

// File viewers
const FV = {
	none: 0,
	music: 1,
	video: 2,
	image: 3
};

const FTtoFV = {
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
	[FT.gif]: FV.image,
	[FT.png]: FV.image,
	[FT.jpeg]: FV.image,
	[FT.tiff]: FV.image,
	[FT.webp]: FV.image,
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

const root = { name: "", path: "", size: 0, time: 0, type: FT.dir };

const shareprefix = "/share/";

const geticonname = (file) => {
	switch (file.type) {
		case FT.dir:
			if (file.path.length > 3) {
				let suff = app.curpathshares.length ? "-pub" : "";
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
			} else {
				return "drive";
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
			return "doc-bitmap";
		case FT.gif:
			return "doc-gif";
		case FT.png:
			return "doc-png";
		case FT.jpeg:
		case FT.tiff:
			return "doc-jpeg";
		case FT.webp:
			return "doc-webp";
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

const getfileurl = (file) => {
	let url = undefined;
	if (file.pref) {
		url = shareprefix + file.pref;
	} else {
		if (app.curpathshares.length) {
			const shr = app.curpathshares[0]; // use any first available share
			url = shareprefix + shr.pref + '/' + shr.suff + file.name;
			if (file.type === FT.dir) {
				url += '/';
			}
		} else {
			url = "/local?" + $.param({
				path: file.path
			});
		}
	}
	return url;
};

const showmsgbox = (title, body) => {
	const dlg = $("#msgbox");
	dlg.find(".modal-title").html(title);
	dlg.find(".modal-body").html(body);
	dlg.modal("show");
};

const onerr404 = () => {
	showmsgbox("Invalid path", "Specified path cannot be accessed now.");
};

let app = new Vue({
	el: '#app',
	template: '#app-tpl',
	data: {
		isadmin: true, // is it running on localhost
		shared: [], // list of shared folders and files
		loadcount: 0, // ajax working request count

		// current opened folder data
		pathlist: [], // list of subfolders properties in current folder
		filelist: [], // list of files properties in current folder
		curpath: root, // current folder properties
		curscan: new Date(), // time of last scanning of current folder
		histpos: 0, // position in history stack
		histlist: [] // history stack
	},
	computed: {
		// array of paths to current folder
		curpathway() {
			if (!this.curpath.name) {
				return [];
			}

			const arr = this.curpath.path.split('/');
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
					path: path,
					size: 0,
					time: 0,
					type: FT.dir
				});
			}
			return lst;
		},

		// get all folder shares
		curpathshares() {
			const lst = [];
			const fldpath = this.curpath.path;
			for (const fp of this.shared) {
				if (fp.path.length <= fldpath.length && fldpath.substr(0, fp.path.length) === fp.path) {
					const shr = Object.assign({}, fp);
					shr.suff = fldpath.substr(shr.path.length, fldpath.length);
					lst.push(shr);
				}
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
				ss += file.size;
			}
			return fmtitemsize(ss);
		},

		// common buttons enablers

		dishome() {
			return !this.curpath.name;
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
				return "to parent folder";
			}
		}
	},
	methods: {
		// opens given folder cleary
		gofolder(file) {
			ajaxjson("GET", "/api/folder?" + $.param({
				path: file.path
			}), xhr => {
				traceresponse(xhr);

				this.pathlist = [];
				this.filelist = [];
				this.curpath = file;
				//window.history.replaceState(null, file.path, "/path/" + file.path);

				if (xhr.status === 200) {
					this.curscan = new Date(Date.now());
					// update path for each item
					const pathlist = xhr.response.paths || [];
					for (const fp of pathlist) {
						fp.path = fp.pref ? fp.pref : file.path + fp.name;
						fp.path += '/';
					}
					const filelist = xhr.response.files || [];
					for (const fp of filelist) {
						fp.path = fp.pref ? fp.pref : file.path + fp.name;
					}
					// update folder settings
					this.pathlist = pathlist;
					this.filelist = filelist;
					// update shares
					if (!file.path) { // shares only at root
						this.shared = [];
						for (const fp of this.pathlist) {
							if (fp.pref) {
								this.shared.push(fp);
							}
						}
					}
					// update map card
					const gpslist = [];
					for (const fp of filelist) {
						if (fp.latitude && fp.longitude && fp.ntmb === 1) {
							gpslist.push(fp);
						}
					}
					this.$refs.mapcard.new();
					this.$refs.mapcard.addmarkers(gpslist);
				} else if (xhr.status === 403) { // Forbidden
					this.isadmin = false;
				}

				// cache folder thumnails
				if (this.uncached.length) {
					const paths = [];
					for (const fp of this.uncached) {
						paths.push(fp.path);
					}
					ajaxjson("POST", "/api/tmb/scn", xhr => { }, {
						paths: paths,
						force: false
					}, true);
					// check cached state loop
					let chktmb;
					chktmb = () => {
						const tmbs = [];
						for (const fp of this.uncached) {
							tmbs.push({ ktmb: fp.ktmb });
						}
						ajaxjson("POST", "/api/tmb/chk", xhr => {
							traceresponse(xhr);
							if (xhr.status === 200) {
								const gpslist = [];
								for (const tp of xhr.response.tmbs) {
									if (tp.ntmb) {
										for (const fp of this.filelist) {
											if (fp.ktmb === tp.ktmb) {
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
								if (this.uncached.length && this.curpath === file) {
									setTimeout(chktmb, 1500); // wait and run again
								}
							}
						}, { tmbs: tmbs }, true);
					};
					// gets thumbs
					setTimeout(chktmb, 600);
				}
			});
		},

		// opens given folder and push history step
		openfolder(file) {
			this.gofolder(file);

			// update folder history
			if (this.histpos < this.histlist.length) {
				this.histlist.splice(this.histpos, this.histlist.length - this.histpos);
			}
			this.histlist.push(file);
			this.histpos = this.histlist.length;
		},

		onhome() {
			this.openfolder(root);
		},

		onback() {
			this.histpos--;
			const file = this.histlist[this.histpos - 1];
			this.gofolder(file);
		},

		onforward() {
			this.histpos++;
			this.gofolder(this.histlist[this.histpos - 1]);
		},

		onparent() {
			this.openfolder(this.curpathway[this.curpathway.length - 1]);
		},

		onrefresh() {
			ajaxjson("POST", "/api/purge", xhr => {
				traceresponse(xhr);
				if (xhr.status === 200) {
					let file = this.curpath;
					this.gofolder(file);
				} else if (xhr.status === 403) { // Forbidden
					this.isadmin = false;
				}
			});
		},

		onshare(file) {
			if (!file.pref) { // should add share
				ajaxjson("PUT", "/api/share/add?" + $.param({
					path: file.path
				}), xhr => {
					traceresponse(xhr);
					if (xhr.status === 200) {
						let shr = xhr.response;
						if (shr) {
							// update folder settings
							Vue.set(file, 'pref', shr.pref);
							if (shr.type === FT.dir) {
								this.shared.push(shr);
							}
						}
					} else if (xhr.status === 403) { // Forbidden
						this.isadmin = false;
					} else if (xhr.status === 404) { // Not Found
						onerr404();
						// remove file from list
						for (const i in this.list) {
							if (this.list[i] === file) {
								this.list.splice(i, 1);
								break;
							}
						}
					}
				});
			} else { // should remove share
				ajaxjson("DELETE", "/api/share/del?" + $.param({
					pref: file.pref
				}), xhr => {
					traceresponse(xhr);
					if (xhr.status === 200) {
						let ok = xhr.response;
						// update folder settings
						if (ok) {
							if (file.type === FT.dir) {
								for (let i in this.shared) {
									if (this.shared[i].pref === file.pref) {
										this.shared.splice(i, 1);
										break;
									}
								}
							}
							Vue.delete(file, 'pref');
						}
					} else if (xhr.status === 403) { // Forbidden
						this.isadmin = false;
					} else if (xhr.status === 404) { // Not Found
						onerr404();
						// remove file from list
						for (const i in this.list) {
							if (this.list[i] === file) {
								this.list.splice(i, 1);
								break;
							}
						}
					}
				});
			}
		},

		onpathopen(file) {
			this.openfolder(file);
		}
	}
});

/////////////
// Startup //
/////////////

$(document).ready(() => {
	console.log(window.location);
	if (window.location.pathname.substr(0, 6) === "/path/") {
		const path = window.location.pathname.substr(6);
		const arr = path.split('/');
		if (!arr[arr.length-1]) {
			arr.pop();
		}
		app.openfolder({
			name: arr[arr.length - 1],
			path: path,
			size: 0, time: 0, type: FT.dir
		});
	} else {
		app.openfolder(root);
	}

	$('.preloader').hide("fast");
	$('#app').show("fast");
});

// The End.
