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
const Photo = 6;
const Bitmap = 7;
const GIF = 8;
const PNG = 9;
const JPEG = 10;
const WebP = 11;
const PDF = 12;
const HTML = 13;
const Text = 14;
const Script = 15;
const Config = 16;
const Log = 17;

// File groups
const FGOther = 0;
const FGMusic = 1;
const FGVideo = 2;
const FGImage = 3;
const FGBooks = 4;
const FGTexts = 5;
const FGDir = 6;

// File viewers
const FVNone = 0;
const FVMusic = 1;
const FVVideo = 2;
const FVImage = 3;

const root = { name: "", path: "", size: 0, time: 0, type: Dir };
let folderhist = [];

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
					} else if (fg[FGMusic] / fnum > 0.5) {
						return "folder-mp3" + suff;
					} else if (fg[FGVideo] / fnum > 0.5) {
						return "folder-movies" + suff;
					} else if (fg[FGImage] / fnum > 0.5) {
						return "folder-photo" + suff;
					} else if (fg[FGBooks] / fnum > 0.5) {
						return "folder-doc" + suff;
					} else if (fg[FGTexts] / fnum > 0.5) {
						return "folder-doc" + suff;
					} else if (fg[FGDir] / fnum > 0.5) {
						return "folder-sub" + suff;
					} else if ((fg[FGMusic] + fg[FGVideo] + fg[FGImage]) / fnum > 0.5) {
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
			order: true, sortmode: sortbyalpha
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
		folderpath: function () {
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
				const fp = {
					name: fn,
					path: path,
					size: 0,
					time: 0,
					type: Dir
				};
				pathlist.push(fp);
			}
			return pathlist;
		},

		// sorted subfolders list
		sortedsubfld: function () {
			return this.subfldlist.slice().sort((v1, v2) => {
				return v1.name.toLowerCase() > v2.name.toLowerCase() ? 1 : -1;
			});
		},

		// sorted files list
		sortedfiles: function () {
			if (this.filter.sortmode === sortbyalpha) {
				return this.filelist.slice().sort((v1, v2) => {
					return v1.name.toLowerCase() > v2.name.toLowerCase() ? 1 : -1;
				});
			} else if (this.filter.sortmode === sortbysize) {
				return this.filelist.slice().sort((v1, v2) => {
					if (v1.size === v2.size) {
						return v1.name.toLowerCase() > v2.name.toLowerCase() ? 1 : -1;
					} else {
						return v1.size > v2.size ? 1 : -1;
					}
				});
			} else { // remains unsorted
				return this.filelist.slice();
			}
		},

		// display filtered playlist
		playlist: function () {
			const pl = [];
			for (const file of this.sortedfiles) {
				if (this.showitem(file)) {
					pl.push(file);
				}
			}
			return pl;
		},

		// files sum size
		sumsize: function () {
			let ss = 0;
			for (let file of this.filelist) {
				ss += file.size;
			}
			return this.fmtsize(ss);
		},

		// get all folder shares

		foldershares: function () {
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

		ishome: function () {
			return { disabled: this.loadcount || !this.folderinfo.name };
		},
		isback: function () {
			return { disabled: this.loadcount || this.folderhistpos < 2 };
		},
		isforward: function () {
			return { disabled: this.loadcount || this.folderhistpos > folderhist.length - 1 };
		},
		isparent: function () {
			return { disabled: this.loadcount || !this.folderpath.length };
		},
		isshared: function () {
			return {
				active: this.selected && this.selected.pref,
				disabled: !this.selected
			};
		},

		ismusic: function () {
			return { active: this.filter.music };
		},
		isvideo: function () {
			return { active: this.filter.video };
		},
		isphoto: function () {
			return { active: this.filter.photo };
		},
		ispdf: function () {
			return { active: this.filter.pdf };
		},
		isbooks: function () {
			return { active: this.filter.books };
		},
		isother: function () {
			return { active: this.filter.other };
		},

		isorder: function () {
			return this.filter.order ? 'arrow_upward' : 'arrow_downward';
		},
		issortmode: function () {
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
		hintback: function () {
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
		hintforward: function () {
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
		hintsortmode: function () {
			switch (this.filter.sortmode) {
				case sortbyalpha:
					return "sort by size";
				case sortbysize:
					return "as is unsorted";
				case unsorted:
					return "sort by alpha";
			}
		},

		// styles changers
		atorder: function () {
			return {
				'flex-wrap': this.filter.order ? 'wrap' : 'wrap-reverse',
				'flex-direction': this.filter.order ? 'row' : 'row-reverse'
			};
		},

		// music buttons

		hintplay: function () {
			let c = true;
			return c ? 'play' : 'pause';
		},
		isrepeat: function () {
			return 'repeat';
		},
		hintrepeat: function () {
			let c = true;
			return c ? 'repeat' : 'repeat one';
		}
	},
	methods: {
		// events responders
		gohome: function () {
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

		goback: function () {
			if (folderhist.length < 2 || this.folderhistpos < 2) {
				return;
			}
			this.folderhistpos--;
			this.gofolder(folderhist[this.folderhistpos - 1]);
		},

		goforward: function () {
			if (folderhist.length < 2 || this.folderhistpos > folderhist.length - 1) {
				return;
			}
			this.folderhistpos++;
			this.gofolder(folderhist[this.folderhistpos - 1]);
		},

		goparent: function () {
			const fl = this.folderpath;
			if (fl.length) {
				this.openfolder(fl[fl.length - 1]);
			}
		},

		openfolder: function (file) {
			this.gofolder(file);

			// update folder history
			if (this.folderhistpos < folderhist.length) {
				folderhist.splice(this.folderhistpos, folderhist.length - this.folderhistpos);
			}
			folderhist.push(file);
			this.folderhistpos = folderhist.length;
		},

		onshare: function () {
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
						folderhist = [];
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
						folderhist = [];
						this.folderhistpos = 0;
					}
				});
			}
		},

		onrefresh: function () {
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

		onsettings: function () {
		},

		onstatistics: function () {
		},

		onorder: function () {
			this.filter.order = !this.filter.order;
		},

		onsortmode: function () {
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

		onlist: function () {
		},

		onmusic: function () {
			this.filter.music = !this.filter.music;
		},

		onvideo: function () {
			this.filter.video = !this.filter.video;
		},

		onphoto: function () {
			this.filter.photo = !this.filter.photo;
		},

		onpdf: function () {
			this.filter.pdf = !this.filter.pdf;
		},

		onbooks: function () {
			this.filter.books = !this.filter.books;
		},

		onother: function () {
			this.filter.other = !this.filter.other;
		},

		ondelsel: function () {
			this.selected = null;
			if (this.viewer) {
				this.viewer.hide();
				this.viewer = null;
			}
		},

		onfilesel: function (e, file) {
			if (this.loadcount) { // no ajax request in progress
				return;
			}
			this.selected = file;
			e.stopPropagation(); // prevent deselect item by parent widget

			// Run viewer/player
			switch (file.type) {
				case Dir:
					if (this.viewer) {
						this.viewer.hide();
						this.viewer = null;
					}
					break;
				case Wave:
				case FLAC:
				case MP3:
				case OGG:
					if (this.viewer !== mp3viewer) {
						if (this.viewer) {
							this.viewer.hide();
						}
						this.viewer = mp3viewer;
						this.viewer.show();
					}
					this.viewer.setfile(file, mp3viewer.isplay());
					break;
				case MP4:
					if (this.viewer) {
						this.viewer.hide();
						this.viewer = null;
					}
					break;
				case Photo:
				case Bitmap:
				case GIF:
				case PNG:
				case JPEG:
				case WebP:
					if (this.viewer) {
						this.viewer.hide();
						this.viewer = null;
					}
					break;
				case PDF:
				case HTML:
					if (this.viewer) {
						this.viewer.hide();
						this.viewer = null;
					}
					break;
				case Text:
				case Script:
				case Config:
				case Log:
					if (this.viewer) {
						this.viewer.hide();
						this.viewer = null;
					}
					break;
				default:
					if (this.viewer) {
						this.viewer.hide();
						this.viewer = null;
					}
					break;
			}
		},

		onfilerun: function (file) {
			if (this.loadcount) { // no ajax request in progress
				return;
			}
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

		gofolder: function (file) {
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
							folderhist = [];
							this.folderhistpos = 0;
						}
					});
				}
			});
		},

		fmtsize: function (size) {
			if (size < 1536) {
				return fmtfilesize(size);
			} else {
				return "%s (%d bytes)".printf(fmtfilesize(size), size);
			}
		},

		fmttitle: function (file) {
			let title = file.name;
			if (file.pref) {
				title += '\nshare: ' + shareprefix + file.pref;
			}
			if (file.type !== Dir) {
				title += '\nsize: ' + this.fmtsize(file.size);
			}
			return title;
		},

		getwebpicon: function (file) {
			return '/asst/file-webp/' + geticonname(file) + '.webp';
		},

		getpngicon: function (file) {
			return '/asst/file-png/' + geticonname(file) + '.png';
		},

		// manage items classes
		itemview: function (file) {
			return { selected: this.selected && this.selected.name === file.name };
		},

		// show/hide functions
		showitem: function (file) {
			switch (file.type) {
				case Dir:
					return true;
				case Wave:
				case FLAC:
				case MP3:
				case OGG:
					return this.filter.music;
				case MP4:
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
		},

		isplay: function (file) {
			return this.selected && file.path === this.selected.path && this.playbackmode;
		}
	}
});

class Viewer {
	show() { }
	hide() { }
	setfile(file) {
		this.file = file;
	}
}

class MP3Player extends Viewer {
	constructor() {
		super();
		this.file = {};
		this.rate = 1.00;
		this.volume = 1.00;
		this.repeatmode = 0; // 0 - no any repeat, 1 - repeat single, 2 - repeat playlist
		this.seeking = false;
		this.playonshow = false;

		this.media = null;

		const frame = $("#music-footer");
		this.frame = frame;
		this.ratemenu = frame.find("#rate");
		this.curbar = frame.find(".timescale > .progress > .current");
		this.bufbar = frame.find(".timescale > .progress > .buffer");
		this.timer = frame.find(".timescale .time-pos");
		this.seeker = frame.find(".timescale > .progress > .seeker");

		this.seeker.on('change', () => {
			this.media.currentTime = Number(this.seeker.val());
			this.seeking = false;
		});
		this.seeker.on('input', () => {
			this.seeking = true;
			this.timer.text(fmttime(Number(this.seeker.val()), this.media.duration));
		});
	}

	show() {
		this.frame.show("fast");
		if (this.playonshow) {
			this.play();
			this.playonshow = false;
		}
	}

	hide() {
		this.frame.hide("fast");
		if (this.isplay()) {
			this.media.pause();
			this.playonshow = true;
		}
	}

	setfile(file, start) {
		if (this.file.path === file.path) { // do not set again same file
			return;
		}
		if (this.isplay()) { // stop previous
			this.media.pause();
		}
		this.file = file;
		this.media = new Audio(getfileurl(file)); // API HTMLMediaElement, HTMLAudioElement
		this.media.playbackRate = this.rate;
		this.media.loop = this.repeatmode === 1;

		this.frame.find(".timescale > div:last-child").text(file.name);

		// disable UI for not ready media
		this.frame.find(".play").addClass('disabled');
		this.seeker.prop('disabled', true);

		// media interface responders
		this.media.addEventListener('loadedmetadata', () => {
			this.updateprogress();
		});
		this.media.addEventListener('canplay', () => {
			const len = this.media.duration;
			const cur = this.media.currentTime;
			// enable UI
			this.frame.find(".play").removeClass('disabled');
			this.seeker.prop('disabled', false);
			this.seeker.attr('min', "0");
			this.seeker.attr('max', len.toString());
			this.seeker.val(cur.toString());
			this.frame.find(".timescale .time-end").text(fmttime(len, len));
			if (start) {
				this.media.play();
			}
		});
		this.media.addEventListener('timeupdate', () => {
			this.updateprogress();
		});
		this.media.addEventListener('seeked', () => {
			this.updateprogress();
		});
		this.media.addEventListener('progress', () => {
			this.updateprogress();
		});
		this.media.addEventListener('play', () => {
			this.frame.find(".play > i").html('pause');
			app.playbackmode = true;
		});
		this.media.addEventListener('pause', () => {
			this.frame.find(".play > i").html('play_arrow');
			app.playbackmode = false;
		});
		this.media.addEventListener('ended', () => {
			const pls = app.playlist;
			const filepos = () => {
				for (const i in pls) {
					const file = pls[i];
					if (this.file.path === file.path) {
						return Number(i);
					}
				}
			};
			const nextpos = (pos) => {
				for (let i = pos + 1; i < pls.length; i++) {
					const file = pls[i];
					if (file.type === Wave || file.type === FLAC ||
						file.type === MP3 || file.type === OGG ||
						file.type === MP4) {
						return file;
					}
				}
			};
			const next1 = nextpos(filepos());
			if (next1) {
				app.selected = next1;
				this.setfile(next1, true);
				return;
			} else if (this.repeatmode === 2) {
				const next2 = nextpos(-1);
				if (next2) {
					app.selected = next2;
					this.setfile(next2, true);
					return;
				}
			}
			this.frame.find(".play > i").html('play_arrow');
			app.playbackmode = false;
		});
	}

	setrate(rate) {
		this.ratemenu.find(".dropdown-item").removeClass("active");
		const str = rate.toFixed(2);
		this.ratemenu.find(".speed_" + str.substr(0, 1) + "_" + str.substr(2, 2)).addClass("active");
		this.rate = rate;
		if (this.media) {
			this.media.playbackRate = rate;
		}
	}

	play() {
		if (this.media.paused) {
			this.media.play();
		} else {
			this.media.pause();
		}
	}

	isplay() {
		return this.media && !this.media.paused;
	}

	updateprogress() {
		const len = this.media.duration;
		const cur = this.media.currentTime;
		{
			let percent;
			if (len === Infinity) { // streamed
				percent = 95;
			} else if (isNaN(len)) { // unknown length
				percent = 5;
			} else {
				percent = cur / len * 100;
			}
			this.curbar.css("width", percent + "%");
		}

		if (this.media.buffered.length > 0) {
			const pos1 = this.media.buffered.start(0);
			const pos2 = this.media.buffered.end(0);
			let percent;
			if (pos1 <= cur && pos2 - cur > 0) { // buffered in current pos
				percent = (pos2 - cur) / len * 100;
			} else { // not buffered or buffered outside
				percent = 0;
			}
			this.bufbar.css("width", percent + "%");
		}

		if (!this.seeking) {
			this.timer.text(fmttime(cur, len));
			this.seeker.val(cur.toString());
		}
	}

	// user events responders

	onprev() {
	}

	onnext() {
	}

	onrepeat() {
		this.repeatmode = (this.repeatmode + 1) % 3;
		if (this.media) {
			this.media.loop = this.repeatmode === 1;
		}

		this.frame.find(".repeat > i").html(this.repeatmode === 1 ? 'repeat_one' : 'repeat');
		if (this.repeatmode) {
			this.frame.find(".repeat").addClass('active');
		} else {
			this.frame.find(".repeat").removeClass('active');
		}
	}
}

/////////////
// Startup //
/////////////

// Widgets
let mp3viewer = undefined;

const initwidgets = () => {
	mp3viewer = new MP3Player();
};

$(document).ready(() => {
	$("nav.footer").hide();
	initwidgets();
	app.gohome();

	$('.preloader').hide("fast");
	$('#app').show("fast");
});

// The End.
