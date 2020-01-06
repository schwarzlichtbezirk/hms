"use strict";

//@ sourceMappingURL=mainpage.min.map

// File types
const Dir = -1;
const File = 0;
const Wave = 1;
const FLAC = 2;
const MP3 = 3;
const OGG = 4;
const MP4 = 5;
const WebM = 6;
const Photo = 7;
const Bitmap = 8;
const GIF = 9;
const PNG = 10;
const JPEG = 11;
const WebP = 12;
const PDF = 13;
const HTML = 14;
const Text = 15;
const Script = 16;
const Config = 17;
const Log = 18;

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

const root = { name: "", path: "", size: 0, time: 0, type: Dir };
const folderhist = [];

const shareprefix = "/share/";

const sortbyalpha = "name";
const sortbysize = "size";
const unsorted = "";

const geticonname = (file) => {
	switch (file.type) {
		case Dir:
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
		case Wave:
			return "doc-wave";
		case FLAC:
			return "doc-flac";
		case MP3:
			return "doc-mp3";
		case OGG:
			return "doc-music";
		case MP4:
			return "doc-mp4";
		case WebM:
			return "doc-movie";
		case Photo:
			return "doc-photo";
		case Bitmap:
			return "doc-bitmap";
		case GIF:
			return "doc-gif";
		case PNG:
			return "doc-png";
		case JPEG:
			return "doc-jpeg";
		case WebP:
			return "doc-webp";
		case PDF:
			return "doc-pdf";
		case HTML:
			return "doc-html";
		case Text:
			return "doc-text";
		case Script:
			return "doc-script";
		case Config:
			return "doc-config";
		case Log:
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
			if (file.type === Dir) {
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

const splitfilelist = (list, subfld, files) => {
	for (const file of list) {
		if (file.type === Dir) {
			subfld.push(file);
		} else {
			files.push(file);
		}
	}
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
		playbackmode: false
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
					type: Dir
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
			return this.selected && (this.selected.type === MP3 || this.selected.type === OGG);
		}
	},
	methods: {
		// events responders
		gohome() {
			// remove selected state before request for any result
			this.ondelsel();

			ajaxjson("GET", "/api/getdrv", (xhr) => {
				traceresponse(xhr);
				if (xhr.status === 200) {
					this.isadmin = true;
					this.folderscan = new Date(Date.now());
					// update folder settings
					const dir = xhr.response;
					this.subfldlist = [];
					this.filelist = [];
					splitfilelist(dir, this.subfldlist, this.filelist);
					this.folderinfo = root;
				} else if (xhr.status === 401) { // Unauthorized
					this.isadmin = false;
					this.subfldlist = [];
					this.filelist = [];
					this.folderinfo = root;
				}

				// get shares for root
				ajaxjson("GET", "/api/shared", (xhr) => {
					traceresponse(xhr);
					if (xhr.status === 200) {
						// update folder settings
						this.shared = xhr.response;
						splitfilelist(this.shared, this.subfldlist, this.filelist);
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
			this.gofolder(folderhist[this.folderhistpos - 1]);
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
				ajaxjson("PUT", "/api/addshr?" + $.param({
					path: this.selected.path
				}), (xhr) => {
					traceresponse(xhr);
					if (xhr.status === 200) {
						let shr = xhr.response;
						if (shr) {
							// update folder settings
							this.selected.pref = shr.pref;
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
				ajaxjson("DELETE", "/api/delshr?" + $.param({
					pref: this.selected.pref
				}), (xhr) => {
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
							this.selected.pref = "";
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
				ajaxjson("GET", "/api/shared", (xhr) => {
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
			switch (file.type) {
				case Dir:
					this.closeviewer();
					break;
				case Wave:
				case FLAC:
				case MP3:
				case OGG:
					if (this.viewer !== this.$refs.mp3player) {
						this.closeviewer();
						this.viewer = this.$refs.mp3player;
						this.viewer.setup();
					}
					mp3player.setfile(file, mp3player.isplay());
					break;
				case MP4:
				case WebM:
					this.closeviewer();
					break;
				case Photo:
				case Bitmap:
				case GIF:
				case PNG:
				case JPEG:
				case WebP:
					this.closeviewer();
					break;
				case PDF:
				case HTML:
					this.closeviewer();
					break;
				case Text:
				case Script:
				case Config:
				case Log:
					this.closeviewer();
					break;
				default:
					this.closeviewer();
					break;
			}
		},

		onfilerun(file) {
			if (file.type === Dir) {
				this.gofolder(file);

				// update folder history
				if (this.folderhistpos < folderhist.length) {
					folderhist.splice(this.folderhistpos, folderhist.length - this.folderhistpos);
				}
				folderhist.push(file);
				this.folderhistpos = folderhist.length;
			} else if (file.type !== File) {
				let url = getfileurl(file);
				window.open(url, file.name);
			}
		},

		// helper functions

		gofolder(file) {
			// remove selected state before request for any result
			this.ondelsel();

			ajaxjson("GET", "/api/folder?" + $.param({
				path: file.path
			}), (xhr) => {
				traceresponse(xhr);
				if (xhr.status === 200) {
					this.folderscan = new Date(Date.now());
					// update folder settings
					const dir = xhr.response;
					this.subfldlist = [];
					this.filelist = [];
					splitfilelist(dir, this.subfldlist, this.filelist);
					this.folderinfo = file;
				}

				// get shares only for root
				if (!file.name) {
					ajaxjson("GET", "/api/shared", (xhr) => {
						traceresponse(xhr);
						if (xhr.status === 200) {
							// update folder settings
							this.shared = xhr.response;
							splitfilelist(this.shared, this.subfldlist, this.filelist);
						} else if (xhr.status === 404) { // Not Found
							onerr404();
							// clear folder history
							folderhist.splice(0, folderhist.length);
							this.folderhistpos = 0;
						}
					});
				}
			});
		},

		// show/hide functions
		showitem(file) {
			switch (file.type) {
				case Dir:
					return true;
				case Wave:
				case FLAC:
				case MP3:
				case OGG:
					return this.filter.music;
				case MP4:
				case WebM:
					return this.filter.video;
				case Photo:
				case Bitmap:
				case GIF:
				case PNG:
				case JPEG:
				case WebP:
					return this.filter.photo;
				case PDF:
				case HTML:
					return this.filter.pdf;
				case Text:
				case Script:
				case Config:
				case Log:
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
