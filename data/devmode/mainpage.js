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
		isadmin: true, // is it running on localhost
		shared: [], // list of shared folders and files
		filter: { // main menu buttons flags
			music: true, video: true, photo: true, pdf: true, books: true, other: false,
			order: false, sortmode: sortbyalpha
		},
		listmode: "mdicon",
		loadcount: 0, // ajax working request count

		// current opened folder data
		pathlist: [], // list of subfolders properties in current folder
		filelist: [], // list of files properties in current folder
		curpath: root, // current folder properties
		selfile: null, // selected file properties
		curscan: new Date(), // time of last scanning of current folder
		histpos: 0, // position in history stack
		histlist: [], // history stack

		// file viewers
		viewer: null,
		playbackfile: null
	},
	computed: {

		// array of paths to current folder
		folderpath() {
			if (!this.curpath.name) {
				return [];
			}

			const arr = this.curpath.path.split('/');
			arr.pop(); // remove empty element from separator at the end
			arr.pop(); // remove current name

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

		// sorted subfolders list
		sortedpathlist() {
			return this.pathlist.slice().sort((v1, v2) => {
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

		// file list with GPS tags
		gpslist() {
			const lst = [];
			for (const file of this.filelist) {
				if (file.latitude && file.longitude) {
					lst.push(file);
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

		// get all folder shares
		foldershares() {
			let shares = [];
			let fldpath = this.curpath.path;
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
			return !this.curpath.name;
		},
		disback() {
			return this.histpos < 2;
		},
		disforward() {
			return this.histpos > this.histlist.length - 1;
		},
		disparent() {
			return !this.folderpath.length;
		},
		disshared() {
			return !this.selfile;
		},
		clsshared() {
			return { active: this.selfile && this.selfile.pref };
		},

		clsfolderlist() {
			switch (this.listmode) {
				case "lgicon":
					return 'align-items-center';
				case "mdicon":
					return 'align-items-start';
			}
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
			return this.filter.order
				? 'arrow_upward'
				: 'arrow_downward';
		},
		clssortmode() {
			switch (this.filter.sortmode) {
				case sortbyalpha:
					return "sort_by_alpha";
				case sortbysize:
					return "sort";
				case unsorted:
					return "reorder";
			}
		},
		clslistmode() {
			switch (this.listmode) {
				case "lgicon":
					return 'view_module';
				case "mdicon":
					return 'subject';
			}
		},

		iconmodetag() {
			switch (this.listmode) {
				case "lgicon":
					return 'img-icon-tag';
				case "mdicon":
					return 'file-icon-tag';
			}
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
		hintorder() {
			return this.filter.order
				? "reverse order"
				: "direct order";
		},
		hintsortmode() {
			switch (this.filter.sortmode) {
				case sortbyalpha:
					return "sort by alpha";
				case sortbysize:
					return "sort by size";
				case unsorted:
					return "as is unsorted";
			}
		},
		hintlist() {
			switch (this.listmode) {
				case "lgicon":
					return "large icons";
				case "mdicon":
					return "middle icons";
			}
		},

		isshowmp3() {
			return this.selfile && FTtoFV[this.selfile.type] === FV.music;
		}
	},
	methods: {
		// opens given folder cleary
		gofolder(file) {
			// remove selected state before request for any result
			this.ondelsel();

			ajaxjson("GET", "/api/folder?" + $.param({
				path: file.path
			}), xhr => {
				traceresponse(xhr);

				this.pathlist = [];
				this.filelist = [];
				this.curpath = file;

				if (xhr.status === 200) {
					this.curscan = new Date(Date.now());
					// update folder settings
					this.pathlist = xhr.response.paths || [];
					this.filelist = xhr.response.files || [];
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
								for (const tp of xhr.response.tmbs) {
									if (tp.ntmb) {
										for (const file of this.filelist) {
											if (file.ktmb === tp.ktmb) {
												Vue.set(file, 'ntmb', tp.ntmb);
												break;
											}
										}
									}
								}
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

		closeviewer() {
			if (this.viewer) {
				this.viewer.close();
				this.viewer = null;
			}
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
				case FT.tga:
				case FT.bmp:
				case FT.gif:
				case FT.png:
				case FT.jpeg:
				case FT.tiff:
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
			this.openfolder(this.folderpath[this.folderpath.length - 1]);
		},

		onshare() {
			if (!this.selfile) {
				return;
			}
			if (!this.selfile.pref) { // should add share
				ajaxjson("PUT", "/api/share/add?" + $.param({
					path: this.selfile.path
				}), xhr => {
					traceresponse(xhr);
					if (xhr.status === 200) {
						let shr = xhr.response;
						if (shr) {
							// update folder settings
							Vue.set(this.selfile, 'pref', shr.pref);
							this.shared.push(shr);
						}
					} else if (xhr.status === 403) { // Forbidden
						this.isadmin = false;
					} else if (xhr.status === 404) { // Not Found
						onerr404();
						// clear folder history
						this.histlist.splice(0, this.histlist.length);
						this.histpos = 0;
					}
				});
			} else { // should remove share
				ajaxjson("DELETE", "/api/share/del?" + $.param({
					pref: this.selfile.pref
				}), xhr => {
					traceresponse(xhr);
					if (xhr.status === 200) {
						let ok = xhr.response;
						// update folder settings
						if (ok) {
							for (let i in this.shared) {
								if (this.shared[i].pref === this.selfile.pref) {
									this.shared.splice(i, 1);
									break;
								}
							}
							Vue.delete(this.selfile, 'pref');
						}
					} else if (xhr.status === 403) { // Forbidden
						this.isadmin = false;
					} else if (xhr.status === 404) { // Not Found
						onerr404();
						// clear folder history
						this.histlist.splice(0, this.histlist.length);
						this.histpos = 0;
					}
				});
			}
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
			switch (this.listmode) {
				case "lgicon":
					this.listmode = 'mdicon';
					break;
				case "mdicon":
					this.listmode = 'lgicon';
					break;
			}
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
			this.selfile = null;
			this.closeviewer();
		},

		onfilesel(file) {
			this.selfile = file;

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
				this.openfolder(file);
			} else if (file.type !== FT.file) {
				let url = getfileurl(file);
				window.open(url, file.name);
			}
		},

		onplayback(file, playback) {
			this.playbackfile = playback && file;
		}
	}
});

/////////////
// Startup //
/////////////

$(document).ready(() => {
	app.openfolder(root);

	$('.preloader').hide("fast");
	$('#app').show("fast");
});

// The End.
