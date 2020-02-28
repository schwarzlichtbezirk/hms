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
	bmp: 8,
	tiff: 9,
	gif: 10,
	png: 11,
	jpeg: 12,
	webp: 13,
	pdf: 14,
	html: 15,
	text: 16,
	scr: 17,
	cfg: 18,
	log: 19
};

// File groups
const FG = {
	other: 0,
	music: 1,
	video: 2,
	image: 3,
	books: 4,
	texts: 5,
	dir: 6
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
	[FT.bmp]: FV.image,
	[FT.tiff]: FV.image,
	[FT.gif]: FV.image,
	[FT.png]: FV.image,
	[FT.jpeg]: FV.image,
	[FT.webp]: FV.image,
	[FT.pdf]: FV.none,
	[FT.html]: FV.none,
	[FT.text]: FV.none,
	[FT.scr]: FV.none,
	[FT.cfg]: FV.none,
	[FT.log]: FV.none
};

const root = { name: "", path: "", size: 0, time: 0, type: FT.dir };
const folderhist = [];

const shareprefix = "/share/";

const sortbyalpha = "name";
const sortbysize = "size";
const unsorted = "";

const geticonname = (file) => {
	switch (file.type) {
		case FT.dir:
			if (file.path.length > 3) {
				let suff = app.foldershares.length ? "-pub" : "";
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
		case FT.bmp:
		case FT.tiff:
			return "doc-bitmap";
		case FT.gif:
			return "doc-gif";
		case FT.png:
			return "doc-png";
		case FT.jpeg:
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
		default: // File and others
			return "doc-file";
	}
};

const getfileurl = (file) => {
	let url = undefined;
	if (file.pref) {
		url = shareprefix + file.pref;
	} else {
		if (app.foldershares.length) {
			const shr = app.foldershares[0]; // use any first available share
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
		isadmin: false, // is it running on localhost
		shared: [], // list of shared folders and files
		filter: { // main menu buttons flags
			music: true, video: true, photo: true, pdf: true, books: true, other: false,
			order: false, sortmode: sortbyalpha
		},
		loadcount: 0, // ajax working request count

		// current opened folder data
		subfldlist: [], // list of subfolders properties in current folder
		filelist: [], // list of files properties in current folder
		folderinfo: root, // current folder properties
		selected: null, // selected file properties
		folderscan: new Date(), // time of last scanning of current folder
		folderhistpos: 0, // position in history stack

		// file viewers
		viewer: null,
		playbackfile: null
	},
	computed: {

		// array of paths to current folder
		folderpath() {
			if (!this.folderinfo.name) {
				return [];
			}

			const arr = this.folderinfo.path.split('/');
			arr.pop(); // remove empty element from separator at the end
			arr.pop(); // remove current name

			const pathlist = [];
			let path = '';
			for (const fn of arr) {
				path += fn + '/';
				pathlist.push({
					name: fn,
					path: path,
					size: 0,
					time: 0,
					type: FT.dir
				});
			}
			return pathlist;
		},

		// sorted subfolders list
		sortedsubfld() {
			return this.subfldlist.slice().sort((v1, v2) => {
				return v1.name.toLowerCase() > v2.name.toLowerCase() ? 1 : -1;
			});
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

		// display filtered sorted playlist
		playlist() {
			const res = [];
			for (const file of this.filelist) {
				if (this.showitem(file)) {
					res.push(file);
				}
			}
			if (this.filter.sortmode === sortbyalpha) {
				res.sort((v1, v2) => {
					return v1.name.toLowerCase() > v2.name.toLowerCase() ? 1 : -1;
				});
			} else if (this.filter.sortmode === sortbysize) {
				res.sort((v1, v2) => {
					if (v1.size === v2.size) {
						return v1.name.toLowerCase() > v2.name.toLowerCase() ? 1 : -1;
					} else {
						return v1.size > v2.size ? 1 : -1;
					}
				});
			}
			if (this.filter.order) {
				res.reverse();
			}
			return res;
		},

		// files sum size
		sumsize() {
			let ss = 0;
			for (let file of this.filelist) {
				ss += file.size;
			}
			return fmtitemsize(ss);
		},

		// get all folder shares

		foldershares() {
			let shares = [];
			let fldpath = this.folderinfo.path;
			for (let fp of this.shared) {
				let shr = Object.assign({}, fp);
				if (shr.path.length <= fldpath.length && fldpath.substr(0, shr.path.length) === shr.path) {
					shr.suff = fldpath.substr(shr.path.length, fldpath.length);
					shares.push(shr);
				}
			}
			return shares;
		},

		// common buttons enablers

		dishome() {
			return !this.folderinfo.name;
		},
		disback() {
			return this.folderhistpos < 2;
		},
		disforward() {
			return this.folderhistpos > folderhist.length - 1;
		},
		disparent() {
			return !this.folderpath.length;
		},
		disshared() {
			return !this.selected;
		},
		clsshared() {
			return { active: this.selected && this.selected.pref };
		},

		clsmusic() {
			return { active: this.filter.music };
		},
		clsvideo() {
			return { active: this.filter.video };
		},
		clsphoto() {
			return { active: this.filter.photo };
		},
		clspdf() {
			return { active: this.filter.pdf };
		},
		clsbooks() {
			return { active: this.filter.books };
		},
		clsother() {
			return { active: this.filter.other };
		},

		clsorder() {
			return this.filter.order ? 'arrow_upward' : 'arrow_downward';
		},
		clssortmode() {
			switch (this.filter.sortmode) {
				case sortbyalpha:
					return "sort";
				case sortbysize:
					return "reorder";
				case unsorted:
					return "sort_by_alpha";
			}
		},

		// buttons hints
		hintback() {
			if (this.folderhistpos < 2) {
				return "go back";
			} else {
				let name = folderhist[this.folderhistpos - 2].name;
				if (!name) {
					name = "root folder";
				}
				return "go back to " + name;
			}
		},
		hintforward() {
			if (this.folderhistpos > folderhist.length - 1) {
				return "go forward";
			} else {
				let name = folderhist[this.folderhistpos].name;
				if (!name) {
					name = "root folder";
				}
				return "go forward to " + name;
			}
		},
		hintsortmode() {
			switch (this.filter.sortmode) {
				case sortbyalpha:
					return "sort by size";
				case sortbysize:
					return "as is unsorted";
				case unsorted:
					return "sort by alpha";
			}
		},

		isshowmp3() {
			return this.selected && FTtoFV[this.selected.type] === FV.music;
		}
	},
	methods: {
		// events responders
		gohome() {
			// remove selected state before request for any result
			this.ondelsel();

			ajaxjson("GET", "/api/getdrv", xhr => {
				traceresponse(xhr);
				if (xhr.status === 200) {
					this.isadmin = true;
					this.folderscan = new Date(Date.now());
					// update folder settings
					this.subfldlist = xhr.response || [];
					this.filelist = [];
					this.folderinfo = root;
				} else if (xhr.status === 403) { // Forbidden
					this.isadmin = false;
					this.subfldlist = [];
					this.filelist = [];
					this.folderinfo = root;
				}

				// get shares for root
				ajaxjson("GET", "/api/share/lst", xhr => {
					traceresponse(xhr);
					if (xhr.status === 200) {
						// update folder settings
						this.shared = xhr.response || [];
						for (const file of this.shared) {
							if (file.type === FT.dir) {
								this.subfldlist.push(file);
							} else {
								this.filelist.push(file);
							}
						}
					}
				});

				// update folder history
				if (this.folderhistpos < folderhist.length) {
					folderhist.splice(this.folderhistpos, folderhist.length - this.folderhistpos);
				}
				folderhist.push(root);
				this.folderhistpos = folderhist.length;
			});
		},

		goback() {
			this.folderhistpos--;
			const file = folderhist[this.folderhistpos - 1];
			if (file.path) {
				this.gofolder(file);
			} else {
				this.gohome();
			}
		},

		goforward() {
			this.folderhistpos++;
			this.gofolder(folderhist[this.folderhistpos - 1]);
		},

		goparent() {
			this.openfolder(this.folderpath[this.folderpath.length - 1]);
		},

		openfolder(file) {
			this.gofolder(file);

			// update folder history
			if (this.folderhistpos < folderhist.length) {
				folderhist.splice(this.folderhistpos, folderhist.length - this.folderhistpos);
			}
			folderhist.push(file);
			this.folderhistpos = folderhist.length;
		},

		closeviewer() {
			if (this.viewer) {
				this.viewer.close();
				this.viewer = null;
			}
		},

		onshare() {
			if (!this.selected) {
				return;
			}
			if (!this.selected.pref) { // should add share
				ajaxjson("PUT", "/api/share/add?" + $.param({
					path: this.selected.path
				}), xhr => {
					traceresponse(xhr);
					if (xhr.status === 200) {
						let shr = xhr.response;
						if (shr) {
							// update folder settings
							Vue.set(this.selected, 'pref', shr.pref);
							this.shared.push(shr);
						}
					} else if (xhr.status === 404) { // Not Found
						onerr404();
						// clear folder history
						folderhist.splice(0, folderhist.length);
						this.folderhistpos = 0;
					}
				});
			} else { // should remove share
				ajaxjson("DELETE", "/api/share/del?" + $.param({
					pref: this.selected.pref
				}), xhr => {
					traceresponse(xhr);
					if (xhr.status === 200) {
						let ok = xhr.response;
						// update folder settings
						if (ok) {
							for (let i in this.shared) {
								if (this.shared[i].pref === this.selected.pref) {
									this.shared.splice(i, 1);
									break;
								}
							}
							Vue.delete(this.selected, 'pref');
						}
					} else if (xhr.status === 404) { // Not Found
						onerr404();
						// clear folder history
						folderhist.splice(0, folderhist.length);
						this.folderhistpos = 0;
					}
				});
			}
		},

		onrefresh() {
			let file = this.folderinfo;
			this.gofolder(file);

			// get shares on any case
			if (file.name) {
				ajaxjson("GET", "/api/share/lst", xhr => {
					traceresponse(xhr);
					if (xhr.status === 200) {
						// update folder settings
						this.shared = xhr.response;
					}
				});
			}
		},

		onsettings() {
		},

		onstatistics() {
		},

		onorder() {
			this.filter.order = !this.filter.order;
		},

		onsortmode() {
			switch (this.filter.sortmode) {
				case sortbyalpha:
					this.filter.sortmode = sortbysize;
					break;
				case sortbysize:
					this.filter.sortmode = unsorted;
					break;
				case unsorted:
					this.filter.sortmode = sortbyalpha;
					break;
			}
		},

		onlist() {
		},

		onmusic() {
			this.filter.music = !this.filter.music;
		},

		onvideo() {
			this.filter.video = !this.filter.video;
		},

		onphoto() {
			this.filter.photo = !this.filter.photo;
		},

		onpdf() {
			this.filter.pdf = !this.filter.pdf;
		},

		onbooks() {
			this.filter.books = !this.filter.books;
		},

		onother() {
			this.filter.other = !this.filter.other;
		},

		ondelsel() {
			this.selected = null;
			this.closeviewer();
		},

		onfilesel(file) {
			this.selected = file;

			// Run viewer/player
			switch (FTtoFV[file.type]) {
				case FV.none:
					this.closeviewer();
					break;
				case FV.music:
					this.viewer = this.$refs.mp3player;
					this.viewer.setfile(file);
					break;
				case FV.video:
					this.closeviewer();
					break;
				case FV.image:
					this.closeviewer();
					break;
				default:
					this.closeviewer();
					break;
			}
		},

		onfilerun(file) {
			if (file.type === FT.dir) {
				this.gofolder(file);

				// update folder history
				if (this.folderhistpos < folderhist.length) {
					folderhist.splice(this.folderhistpos, folderhist.length - this.folderhistpos);
				}
				folderhist.push(file);
				this.folderhistpos = folderhist.length;
			} else if (file.type !== FT.file) {
				let url = getfileurl(file);
				window.open(url, file.name);
			}
		},

		onplayback(file, playback) {
			this.playbackfile = playback && file;
		},

		// helper functions

		gofolder(file) {
			// remove selected state before request for any result
			this.ondelsel();

			ajaxjson("GET", "/api/folder?" + $.param({
				path: file.path
			}), xhr => {
				traceresponse(xhr);
				if (xhr.status === 200) {
					this.folderscan = new Date(Date.now());
					// update folder settings
					this.subfldlist = xhr.response.paths || [];
					this.filelist = xhr.response.files || [];
					this.folderinfo = file;

					// cache folder thumnails
					if (this.uncached.length) {
						ajaxjson("POST", "/api/tmb/scn", xhr => { }, {
							itmbs: this.uncached,
							force: false
						}, true);
					}

					// check cached state loop
					let chktmb;
					chktmb = () => {
						if (!this.uncached.length || this.folderinfo !== file) {
							return;
						}
						ajaxjson("POST", "/api/tmb/chk", xhr => {
							traceresponse(xhr);
							if (xhr.status === 200) {
								for (const itmb of xhr.response.itmbs) {
									if (itmb.ntmb) {
										for (const file of this.filelist) {
											if (file.ktmb === itmb.ktmb) {
												Vue.set(file, 'ntmb', itmb.ntmb);
												break;
											}
										}
									}
								}
								setTimeout(chktmb, 1500);
							}
						}, { itmbs: this.uncached }, true);
					};
					setTimeout(chktmb, 600);
				}
			});
		},

		// show/hide functions
		showitem(file) {
			switch (file.type) {
				case FT.dir:
					return true;
				case FT.wave:
				case FT.flac:
				case FT.mp3:
					return this.filter.music;
				case FT.ogg:
				case FT.mp4:
				case FT.webm:
					return this.filter.video;
				case FT.photo:
				case FT.bmp:
				case FT.tiff:
				case FT.gif:
				case FT.png:
				case FT.jpeg:
				case FT.webp:
					return this.filter.photo;
				case FT.pdf:
				case FT.html:
					return this.filter.pdf;
				case FT.text:
				case FT.scr:
				case FT.cfg:
				case FT.log:
					return this.filter.books;
				default:
					return this.filter.other;
			}
		}
	}
});

/////////////
// Startup //
/////////////

$(document).ready(() => {
	app.gohome();

	$('.preloader').hide("fast");
	$('#app').show("fast");
});

// The End.
