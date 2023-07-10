"use strict";

const sortmodeicon = {
	byalpha: 'sort_by_alpha',
	bytime: 'schedule',
	bysize: 'sort',
	unsorted: 'filter_alt_off',
};
const sortmodehint = {
	byalpha: "sort by alpha",
	bytime: "sort by time",
	bysize: "sort by size",
	unsorted: "as is unsorted",
};

const listmodetag = {
	xs: 'list-item-tag',
	sm: 'file-item-tag',
	md: 'file-item-tag',
	lg: 'img-item-tag',
};
const listmoderow = {
	xs: 'align-items-start',
	sm: 'align-items-start',
	md: 'align-items-start',
	lg: 'align-items-center',
};
const listmodeicon = {
	xs: 'format_align_justify',
	sm: 'view_comfy',
	md: 'view_module',
	lg: 'widgets',
};
const listmodehint = {
	xs: "list",
	sm: "small icons",
	md: "middle icons",
	lg: "large icons",
};

const noderadius = 15;

const circulartimeout = 3 * 1000;
const circularmindist = 5; // minimum distance in points
const circularmaxradius = 112 * 1000;

// map modes states
const mm = {
	view: 'view',
	draw: 'draw',
	remove: 'remove',
};

// typeEXIF checks that file extension belongs to images with EXIF tags.
const typeEXIF = {
	".tif": true, ".tiff": true, ".dng": true,
	".jpg": true, ".jpe": true, ".jpeg": true, ".jfif": true,
	".png": true, ".webp": true,
};

// typeTile checks that file with this extension can be used to build tiles sheet.
const typeTile = {
	".jpg": true, ".jpe": true, ".jpeg": true, ".jfif": true,
	/*".avif": true,*/ ".png": true, ".webp": true, ".gif": true,
}

const gpxcolors = [
	'#6495ED', // CornflowerBlue
	'#DA70D6', // Orchid
	'#DAA520', // GoldenRod
	'#9370DB', // MediumPurple
	'#DB7093', // PaleVioletRed
	'#8FBC8F', // DarkSeaGreen
	'#FF7F50', // Coral
	'#9ACD32', // YellowGreen
	'#BC8F8F', // RosyBrown
	'#5F9EA0', // CadetBlue
	'#778899', // LightSlateGray
	'#F4A460', // SandyBrown
	'#FA8072', // Salmon
	'#20B2AA', // LightSeaGreen
	'#4169E1', // RoyalBlue
];

const copyTextToClipboard = text => {
	const elem = document.createElement("textarea");
	elem.value = text;
	elem.style.position = 'fixed'; // prevent to scroll content to the end
	elem.style.left = '0px';
	elem.style.top = '0px';
	document.body.appendChild(elem);
	elem.focus();
	elem.select();

	try {
		return !document.execCommand('copy') && "copying text command was failed";
	} catch (err) {
		return err;
	} finally {
		document.body.removeChild(elem);
	}
};

///////////////////////////
// Full screen functions //
///////////////////////////

const isFullscreen = () => document.fullscreenElement ||
	document.webkitFullscreenElement ||
	document.mozFullScreenElement ||
	document.msFullscreenElement;

const openFullscreen = e => {
	if (e.requestFullscreen) {
		e.requestFullscreen();
	} else if (e.mozRequestFullScreen) {
		e.mozRequestFullScreen();
	} else if (e.webkitRequestFullscreen) {
		e.webkitRequestFullscreen(Element.ALLOW_KEYBOARD_INPUT);
	} else if (e.msRequestFullscreen) {
		e.msRequestFullscreen();
	}
};

const closeFullscreen = () => {
	if (document.exitFullscreen) {
		document.exitFullscreen();
	} else if (document.mozCancelFullScreen) {
		document.mozCancelFullScreen();
	} else if (document.webkitExitFullscreen) {
		document.webkitExitFullscreen();
	} else if (document.msExitFullscreen) {
		document.msExitFullscreen();
	}
};

// leaflet html addons

const imagetmpl = src => `
<div class="position-relative">
	<picture>
		${src}
		<img class="position-absolute top-50 start-50 translate-middle w-100">
	</picture>
</div>
`;

const photoinfotmpl = (file, src) => `
<div class="photoinfo">
	<ul class="nav nav-tabs" role="tablist">
		<li class="nav-item"><a class="nav-link active" data-bs-toggle="tab" href="#pict">Thumbnail</a></li>
		<li class="nav-item"><a class="nav-link" data-bs-toggle="tab" href="#prop">Properties</a></li>
	</ul>
	<div class="tab-content">
		<div class="tab-pane active" id="pict">
			<picture>
				${src}
				<img class="rounded thumb" alt="${file.name}">
			</picture>
			<ul class="d-flex flex-wrap list-inline latlng">
				<li><div class="name">lat:</div> <div class="value">${file.latitude.toFixed(6)}</div></li>
				<li><div class="name">lon:</div> <div class="value">${file.longitude.toFixed(6)}</div></li>
				${file.altitude ? `<li><div class="name">alt:</div> <div class="value">` + file.altitude + `</div></li>` : ""}
			</ul>
		</div>
		<div class="tab-pane fade" id="prop">
			<ul class="list-unstyled prop"><li>${
				fileinfo(file).map(e => `<div class="name">${e[0]}:</div> <div class="value">${e[1]}</div>`).join("</li><li>")
			}</li></ul>
		</div>
	</div>
</div>
`;

const VueCtgrCard = {
	template: '#ctgr-card-tpl',
	props: ["flist"],
	data() {
		return {
			listmode: 'sm',
			pinned: false,
			expanded: true,
			iid: makestrid(10), // instance ID
		};
	},
	watch: {
		flist: {
			handler(newlist, oldlist) {
				this.pinned = this.$root.curpuid === CNID.home;
			}
		},
	},
	computed: {
		ctgrlist() {
			const l = [];
			for (const file of this.flist) {
				if (file.type === FT.ctgr) {
					l.push(file);
				}
			}
			return l;
		},
		isvisible() {
			return this.pinned || this.ctgrlist.length > 0;
		},
		sortedlist() {
			return this.ctgrlist;
		},

		clsfilelist() {
			return listmoderow[this.listmode];
		},

		clslistmode() {
			return listmodeicon[this.listmode];
		},

		iconmodetag() {
			return listmodetag[this.listmode];
		},

		hintlist() {
			return listmodehint[this.listmode];
		},
		expandchevron() {
			return this.expanded ? 'expand_more' : 'chevron_right';
		},
	},
	methods: {
		onlistmodels() {
			this.listmode = 'xs';
			storageSetItem("card.ctgr.listmode", this.listmode);
		},
		onlistmodesm() {
			this.listmode = 'sm';
			storageSetItem("card.ctgr.listmode", this.listmode);
		},
		onlistmodemd() {
			this.listmode = 'md';
			storageSetItem("card.ctgr.listmode", this.listmode);
		},
		onlistmodelg() {
			this.listmode = 'lg';
			storageSetItem("card.ctgr.listmode", this.listmode);
		},

		onselect(file) {
			eventHub.emit('select', file);
		},
		onopen(file) {
			eventHub.emit('open', file);
		},
		onunselect() {
			eventHub.emit('select', null);
		},
		onexpand(e) {
			this.expanded = true;
			storageSetItem("card.ctgr.expanded", this.expanded);
			this.expand();
		},
		oncollapse(e) {
			this.expanded = false;
			storageSetItem("card.ctgr.expanded", this.expanded);
			this.collapse();
		},
		expand() {
		},
		collapse() {
		},
	},
	created() {
		this.listmode = storageGetItem("card.ctgr.listmode", this.listmode);
		this.expanded = storageGetItem("card.ctgr.expanded", this.expanded);
	},
	mounted() {
		const el = document.getElementById('card' + this.iid);
		if (el) {
			if (this.expanded) { 
				el.classList.add('show');
				this.expand();
			} else {
				this.collapse();
			}
			el.addEventListener('shown.bs.collapse', this.onexpand);
			el.addEventListener('hidden.bs.collapse', this.oncollapse);
		}
	},
	unmounted() {
		const el = document.getElementById('card' + this.iid);
		if (el) {
			el.removeEventListener('shown.bs.collapse', this.onexpand);
			el.removeEventListener('hidden.bs.collapse', this.oncollapse);
		}
	},
};

const defcloudport = {
	ftp: "21",
	sftp: "22",
	http: "",
	https: "",
};

const VueCloudCard = {
	template: '#cloud-card-tpl',
	props: ["flist"],
	data() {
		return {
			selfile: null, // current selected cloud
			sortorder: 1,
			listmode: 'sm',

			scheme: 'ftp',
			host: "", // path to disk to add
			port: 21,
			login: "",
			password: "",
			showpswd: false,
			name: "",
			hoststate: 0,
			portstate: 0,
			loginstate: 0,
			passstate: 0,
			rootadd: null,

			pinned: false,
			expanded: true,
			iid: makestrid(10), // instance ID
		};
	},
	watch: {
		flist: {
			handler(newlist, oldlist) {
				this.pinned = this.$root.curpuid === CNID.remote;
			}
		},
		scheme: {
			handler(newval, oldval) {
				this.port = defcloudport[newval];
			}
		}
	},
	computed: {
		drvlist() {
			const l = [];
			for (const file of this.flist) {
				if (file.type === FT.cld) {
					l.push(file);
				}
			}
			return l;
		},
		isvisible() {
			return this.pinned || this.drvlist.length > 0;
		},
		sortedlist() {
			return this.drvlist.slice().sort((v1, v2) => {
				return this.sortorder * (v1.name.toLowerCase() > v2.name.toLowerCase() ? 1 : -1);
			});
		},

		pswdstate() {
			return this.showpswd ? 'visibility' : 'visibility_off';
		},
		clspswdstate() {
			return this.showpswd ? 'text' : 'password';
		},

		clsfilelist() {
			return listmoderow[this.listmode];
		},
		clsorder() {
			return this.sortorder > 0
				? 'arrow_downward'
				: 'arrow_upward';
		},
		clslistmode() {
			return listmodeicon[this.listmode];
		},

		clshostedt() {
			return {
				'is-invalid': this.hoststate === -1,
				'is-valid': this.hoststate == 1
			};
		},
		clsportedt() {
			return {
				'is-invalid': this.portstate === -1,
				'is-valid': this.portstate == 1
			};
		},
		clsloginedt() {
			return {
				'is-invalid': this.loginstate === -1,
				'is-valid': this.loginstate == 1
			};
		},
		clspassedt() {
			return {
				'is-invalid': this.portstate === -1,
				'is-valid': this.portstate == 1
			};
		},
		clsadd() {
			return {
				'disabled': !this.host.length || !(this.port || !defcloudport[this.scheme]) ||
					this.hoststate < 0 || this.portstate < 0
			};
		},
		clsremove() {
			return { 'disabled': !this.selfile };
		},

		iconmodetag() {
			return listmodetag[this.listmode];
		},

		hintorder() {
			return this.sortorder > 0
				? "direct order"
				: "reverse order";
		},
		hintlist() {
			return listmodehint[this.listmode];
		},
		expandchevron() {
			return this.expanded ? 'expand_more' : 'chevron_right';
		},
	},
	methods: {
		onorder() {
			this.sortorder = -this.sortorder;
			storageSetItem("card.cloud.sortorder", this.sortorder);
		},
		onlistmodels() {
			this.listmode = 'xs';
			storageSetItem("card.cloud.listmode", this.listmode);
		},
		onlistmodesm() {
			this.listmode = 'sm';
			storageSetItem("card.cloud.listmode", this.listmode);
		},
		onlistmodemd() {
			this.listmode = 'md';
			storageSetItem("card.cloud.listmode", this.listmode);
		},
		onlistmodelg() {
			this.listmode = 'lg';
			storageSetItem("card.cloud.listmode", this.listmode);
		},

		onadd() {
			(async () => {
				eventHub.emit('ajax', +1);
				try {
					const response = await fetchjsonauth("POST", `/id${this.$root.aid}/api/cloud/add`, {
						scheme: this.scheme,
						host: this.host,
						port: this.port,
						login: this.login,
						password: this.password,
						name: this.name,
					});
					const data = await response.json();
					traceajax(response, data);
					if (response.ok) {
						if (data.added) {
							this.flist.push(data.fp);
						}
						this.rootadd?.hide();
					} else {
						this.rootpathstate = -1;
					}
				} catch (e) {
					ajaxfail(e);
				} finally {
					eventHub.emit('ajax', -1);
				}
			})();
		},
		onremove() {
			(async () => {
				eventHub.emit('ajax', +1);
				try {
					const response = await fetchjsonauth("POST", `/id${this.$root.aid}/api/cloud/del`, {
						puid: this.selfile.puid
					});
					const data = await response.json();
					traceajax(response, data);
					if (!response.ok) {
						throw new HttpError(response.status, data);
					}

					if (data.deleted) {
						this.flist.splice(this.flist.findIndex(elem => elem === this.selfile), 1);
						if (this.selfile.shared) {
							await this.$root.fetchsharedel(this.selfile);
						}
						this.selfile = null;
					}
				} catch (e) {
					ajaxfail(e);
				} finally {
					eventHub.emit('ajax', -1);
				}
			})();
		},
		onshowpswd() {
			this.showpswd = !this.showpswd;
		},
		onchange(e) {
			if (!this.host.length) {
				this.hoststate = -1;
			} else {
				this.hoststate = 0;
			}
			if (defcloudport[this.scheme] && !this.port) {
				this.portstate = -1;
			} else {
				this.portstate = 0;
			}
			this.loginstate = 0;
			this.passstate = 0;
		},

		onselect(file) {
			eventHub.emit('select', file);
		},
		onopen(file) {
			eventHub.emit('open', file);
		},
		onunselect() {
			eventHub.emit('select', null);
		},
		onexpand(e) {
			this.expanded = true;
			storageSetItem("card.ctgr.expanded", this.expanded);
			this.expand();
		},
		oncollapse(e) {
			this.expanded = false;
			storageSetItem("card.ctgr.expanded", this.expanded);
			this.collapse();
		},
		expand() {
		},
		collapse() {
		},

		onanyselect(file) {
			if (file && file.type === FT.cld) {
				this.selfile = file
			} else {
				this.selfile = null;
			}
		},
	},
	created() {
		this.sortorder = storageGetItem("card.cloud.sortorder", this.sortorder);
		this.listmode = storageGetItem("card.cloud.listmode", this.listmode);
		this.expanded = storageGetItem("card.cloud.expanded", this.expanded);
	},
	mounted() {
		eventHub.on('select', this.onanyselect);
		{
			const el = document.getElementById('card' + this.iid);
			if (el) {
				if (this.expanded) {
					el.classList.add('show');
					this.expand();
				} else {
					this.collapse();
				}
				el.addEventListener('shown.bs.collapse', this.onexpand);
				el.addEventListener('hidden.bs.collapse', this.oncollapse);
			}
		}

		// init rootadd dialog
		{
			const el = document.getElementById('rootadd' + this.iid);
			if (el) {
				this.rootadd = new bootstrap.Modal(el);
				el.addEventListener('shown.bs.modal', e => {
					el.querySelector('input').focus();
				});
			}
		}
	},
	unmounted() {
		eventHub.off('select', this.onanyselect);
		{
			const el = document.getElementById('card' + this.iid);
			if (el) {
				el.removeEventListener('shown.bs.collapse', this.onexpand);
				el.removeEventListener('hidden.bs.collapse', this.oncollapse);
			}
		}

		// erase rootadd dialog
		this.rootadd = null;
	},
};

const VueDriveCard = {
	template: '#drive-card-tpl',
	props: ["flist"],
	data() {
		return {
			selfile: null, // current selected drive
			sortorder: 1,
			listmode: 'sm',

			rootpath: "", // path to disk to add
			name: "",
			rootpathstate: 0,
			rootadd: null,

			pinned: false,
			expanded: true,
			iid: makestrid(10), // instance ID
		};
	},
	watch: {
		flist: {
			handler(newlist, oldlist) {
				this.pinned = this.$root.curpuid === CNID.local;
			}
		},
	},
	computed: {
		drvlist() {
			const l = [];
			for (const file of this.flist) {
				if (file.type === FT.drv) {
					l.push(file);
				}
			}
			return l;
		},
		isvisible() {
			return this.pinned || this.drvlist.length > 0;
		},
		sortedlist() {
			return this.drvlist.slice().sort((v1, v2) => {
				return this.sortorder * (v1.name.toLowerCase() > v2.name.toLowerCase() ? 1 : -1);
			});
		},

		clsfilelist() {
			return listmoderow[this.listmode];
		},

		clsorder() {
			return this.sortorder > 0
				? 'arrow_downward'
				: 'arrow_upward';
		},
		clslistmode() {
			return listmodeicon[this.listmode];
		},

		clsrootpathedt() {
			return {
				'is-invalid': this.rootpathstate === -1,
				'is-valid': this.rootpathstate == 1
			};
		},
		clsadd() {
			return { 'disabled': !this.rootpath.length || this.rootpathstate < 0 };
		},
		clsremove() {
			return { 'disabled': !this.selfile };
		},

		iconmodetag() {
			return listmodetag[this.listmode];
		},

		hintorder() {
			return this.sortorder > 0
				? "direct order"
				: "reverse order";
		},
		hintlist() {
			return listmodehint[this.listmode];
		},
		expandchevron() {
			return this.expanded ? 'expand_more' : 'chevron_right';
		},
	},
	methods: {
		onorder() {
			this.sortorder = -this.sortorder;
			storageSetItem("card.drive.sortorder", this.sortorder);
		},
		onlistmodels() {
			this.listmode = 'xs';
			storageSetItem("card.drive.listmode", this.listmode);
		},
		onlistmodesm() {
			this.listmode = 'sm';
			storageSetItem("card.drive.listmode", this.listmode);
		},
		onlistmodemd() {
			this.listmode = 'md';
			storageSetItem("card.drive.listmode", this.listmode);
		},
		onlistmodelg() {
			this.listmode = 'lg';
			storageSetItem("card.drive.listmode", this.listmode);
		},

		onadd() {
			(async () => {
				eventHub.emit('ajax', +1);
				try {
					const response = await fetchjsonauth("POST", `/id${this.$root.aid}/api/drive/add`, {
						path: this.rootpath,
						name: this.name,
					});
					const data = await response.json();
					traceajax(response, data);
					if (response.ok) {
						if (data.added) {
							this.flist.push(data.fp);
						}
						this.rootadd?.hide();
					} else {
						this.rootpathstate = -1;
					}
				} catch (e) {
					ajaxfail(e);
				} finally {
					eventHub.emit('ajax', -1);
				}
			})();
		},
		onremove() {
			(async () => {
				eventHub.emit('ajax', +1);
				try {
					const response = await fetchjsonauth("POST", `/id${this.$root.aid}/api/drive/del`, {
						puid: this.selfile.puid
					});
					const data = await response.json();
					traceajax(response, data);
					if (!response.ok) {
						throw new HttpError(response.status, data);
					}

					if (data.deleted) {
						this.flist.splice(this.flist.findIndex(elem => elem === this.selfile), 1);
						if (this.selfile.shared) {
							await this.$root.fetchsharedel(this.selfile);
						}
						this.selfile = null;
					}
				} catch (e) {
					ajaxfail(e);
				} finally {
					eventHub.emit('ajax', -1);
				}
			})();
		},
		onpathchange(e) {
			(async () => {
				try {
					const response = await fetchjsonauth("POST", `/id${this.$root.aid}/api/res/ispath`, {
						path: this.rootpath
					});
					const data = await response.json();
					if (response.ok) {
						if (data.valid) {
							// makes green if path is not present
							// at another drives of profile
							this.rootpathstate = data.space ? 0 : 1;
						} else {
							this.rootpathstate = -1;
						}
					} else {
						this.rootpathstate = -1;
					}
				} catch (e) {
					ajaxfail(e);
				}
			})();
		},

		onselect(file) {
			eventHub.emit('select', file);
		},
		onopen(file) {
			eventHub.emit('open', file);
		},
		onunselect() {
			eventHub.emit('select', null);
		},
		onexpand(e) {
			this.expanded = true;
			storageSetItem("card.ctgr.expanded", this.expanded);
			this.expand();
		},
		oncollapse(e) {
			this.expanded = false;
			storageSetItem("card.ctgr.expanded", this.expanded);
			this.collapse();
		},
		expand() {
		},
		collapse() {
		},

		onanyselect(file) {
			if (file && file.type === FT.drv) {
				this.selfile = file
			} else {
				this.selfile = null;
			}
		},
	},
	created() {
		this.sortorder = storageGetItem("card.drive.sortorder", this.sortorder);
		this.listmode = storageGetItem("card.drive.listmode", this.listmode);
		this.expanded = storageGetItem("card.drive.expanded", this.expanded);
	},
	mounted() {
		eventHub.on('select', this.onanyselect);
		{
			const el = document.getElementById('card' + this.iid);
			if (el) {
				if (this.expanded) {
					el.classList.add('show');
					this.expand();
				} else {
					this.collapse();
				}
				el.addEventListener('shown.bs.collapse', this.onexpand);
				el.addEventListener('hidden.bs.collapse', this.oncollapse);
			}
		}

		// init rootadd dialog
		{
			const el = document.getElementById('rootadd' + this.iid);
			if (el) {
				this.rootadd = new bootstrap.Modal(el);
				el.addEventListener('shown.bs.modal', e => {
					el.querySelector('input').focus();
				});
			}
		}
	},
	unmounted() {
		eventHub.off('select', this.onanyselect);
		{
			const el = document.getElementById('card' + this.iid);
			if (el) {
				el.removeEventListener('shown.bs.collapse', this.onexpand);
				el.removeEventListener('hidden.bs.collapse', this.oncollapse);
			}
		}

		// erase rootadd dialog
		this.rootadd = null;
	},
};

const VueDirCard = {
	template: '#dir-card-tpl',
	props: ["flist"],
	data() {
		return {
			sortorder: 1,
			listmode: 'sm',
			expanded: true,
			iid: makestrid(10), // instance ID
		};
	},
	computed: {
		dirlist() {
			const l = [];
			for (const file of this.flist) {
				if (file.type === FT.dir) {
					l.push(file);
				}
			}
			return l;
		},
		isvisible() {
			return this.dirlist.length > 0;
		},
		sortedlist() {
			return this.dirlist.slice().sort((v1, v2) => {
				return this.sortorder * (v1.name.toLowerCase() > v2.name.toLowerCase() ? 1 : -1);
			});
		},

		clsfilelist() {
			return listmoderow[this.listmode];
		},

		clsorder() {
			return this.sortorder > 0
				? 'arrow_downward'
				: 'arrow_upward';
		},
		clslistmode() {
			return listmodeicon[this.listmode];
		},

		iconmodetag() {
			return listmodetag[this.listmode];
		},

		hintorder() {
			return this.sortorder > 0
				? "direct order"
				: "reverse order";
		},
		hintlist() {
			return listmodehint[this.listmode];
		},
		expandchevron() {
			return this.expanded ? 'expand_more' : 'chevron_right';
		},
	},
	methods: {
		onorder() {
			this.sortorder = -this.sortorder;
			storageSetItem("card.dir.sortorder", this.sortorder);
		},
		onlistmodels() {
			this.listmode = 'xs';
			storageSetItem("card.dir.listmode", this.listmode);
		},
		onlistmodesm() {
			this.listmode = 'sm';
			storageSetItem("card.dir.listmode", this.listmode);
		},
		onlistmodemd() {
			this.listmode = 'md';
			storageSetItem("card.dir.listmode", this.listmode);
		},
		onlistmodelg() {
			this.listmode = 'lg';
			storageSetItem("card.dir.listmode", this.listmode);
		},

		onselect(file) {
			eventHub.emit('select', file);
		},
		onopen(file) {
			eventHub.emit('open', file);
		},
		onunselect() {
			eventHub.emit('select', null);
		},
		onexpand(e) {
			this.expanded = true;
			storageSetItem("card.ctgr.expanded", this.expanded);
			this.expand();
		},
		oncollapse(e) {
			this.expanded = false;
			storageSetItem("card.ctgr.expanded", this.expanded);
			this.collapse();
		},
		expand() {
		},
		collapse() {
		},
	},
	created() {
		this.sortorder = storageGetItem("card.dir.sortorder", this.sortorder);
		this.listmode = storageGetItem("card.dir.listmode", this.listmode);
		this.expanded = storageGetItem("card.dir.expanded", this.expanded);
	},
	mounted() {
		const el = document.getElementById('card' + this.iid);
		if (el) {
			if (this.expanded) { 
				el.classList.add('show');
				this.expand();
			} else {
				this.collapse();
			}
			el.addEventListener('shown.bs.collapse', this.onexpand);
			el.addEventListener('hidden.bs.collapse', this.oncollapse);
		}
	},
	unmounted() {
		const el = document.getElementById('card' + this.iid);
		if (el) {
			el.removeEventListener('shown.bs.collapse', this.onexpand);
			el.removeEventListener('hidden.bs.collapse', this.oncollapse);
		}
	},
};

const VueFileCard = {
	template: '#file-card-tpl',
	props: ["flist"],
	data() {
		return {
			sortorder: 1,
			sortmode: 'byalpha',
			listmode: 'sm',
			thumbmode: true,
			audioonly: false,
			fgshow: [
				false, // other
				true, // video
				true, // audio
				true, // image
				true, // books
				true, // texts
				true, // packs
				true // dir
			],

			flisthub: makeeventhub(),
			expanded: true,
			iid: makestrid(10), // instance ID
		};
	},
	watch: {
		flist: {
			handler(newlist, oldlist) {
				this.flisthub.emit(null);
				this.onnewlist(newlist, oldlist);
			}
		},
		sortedlist: {
			handler(val, oldval) {
				eventHub.emit('sortedlist', val);
			}
		},
	},
	computed: {
		filelist() {
			const l = [];
			for (const file of this.flist) {
				if (file.type === FT.file) {
					l.push(file);
				}
			}
			return l;
		},
		isvisible() {
			return this.filelist.length > 0;
		},
		sortedlist() {
			const res = [];
			for (const file of this.filelist) {
				if (this.fgshow[getFileGroup(file)]) {
					res.push(file);
				}
			}
			if (this.sortmode === 'byalpha') {
				res.sort((v1, v2) => {
					return this.sortorder * (v1.name.toLowerCase() > v2.name.toLowerCase() ? 1 : -1);
				});
			} else if (this.sortmode === 'bytime') {
				res.sort((v1, v2) => {
					const t1 = v1.datetime ?? v1.time;
					const t2 = v2.datetime ?? v2.time;
					if (t1 !== t2) {
						return this.sortorder * (t1 > t2 ? 1 : -1);
					}
					return this.sortorder * (v1.name.toLowerCase() > v2.name.toLowerCase() ? 1 : -1);
				});
			} else if (this.sortmode === 'bysize') {
				res.sort((v1, v2) => {
					const s1 = v1.size ?? 0;
					const s2 = v2.size ?? 0;
					if (s1 !== s2) {
						return this.sortorder * (s1 > s2 ? 1 : -1);
					}
					return this.sortorder * (v1.name.toLowerCase() > v2.name.toLowerCase() ? 1 : -1);
				});
			}
			return res;
		},

		clsfilelist() {
			return listmoderow[this.listmode];
		},

		clsorder() {
			return this.sortorder > 0
				? 'arrow_downward'
				: 'arrow_upward';
		},
		clssortmode() {
			return sortmodeicon[this.sortmode];
		},
		clslistmode() {
			return listmodeicon[this.listmode];
		},

		showmusic() {
			return !!this.filelist.find(file => extfmt.audio[pathext(file.name)]);
		},
		showvideo() {
			return !!this.filelist.find(file => extfmt.video[pathext(file.name)]);
		},
		showphoto() {
			return !!this.filelist.find(file => extfmt.image[pathext(file.name)]);
		},
		showbooks() {
			return !!this.filelist.find(file => extfmt.books[pathext(file.name)]);
		},
		showtexts() {
			return !!this.filelist.find(file => extfmt.texts[pathext(file.name)]);
		},
		showpacks() {
			return !!this.filelist.find(file => extfmt.packs[pathext(file.name)]);
		},
		showother() {
			return !!this.filelist.find(file => getFileGroup(file) === FG.other);
		},

		clsthumbmode() {
			return { active: this.thumbmode };
		},
		clsheadset() {
			return { active: this.audioonly };
		},
		clsaudio() {
			return { active: this.fgshow[FG.audio] };
		},
		clsvideo() {
			return { active: this.fgshow[FG.video] };
		},
		clsphoto() {
			return { active: this.fgshow[FG.image] };
		},
		clsbooks() {
			return { active: this.fgshow[FG.books] };
		},
		clstexts() {
			return { active: this.fgshow[FG.texts] };
		},
		clspacks() {
			return { active: this.fgshow[FG.packs] };
		},
		clsother() {
			return { active: this.fgshow[FG.other] };
		},

		iconmodetag() {
			return listmodetag[this.listmode];
		},

		hintorder() {
			return this.sortorder > 0
				? "direct order"
				: "reverse order";
		},
		hintsortmode() {
			return sortmodehint[this.sortmode];
		},
		hintlist() {
			return listmodehint[this.listmode];
		},
		expandchevron() {
			return this.expanded ? 'expand_more' : 'chevron_right';
		},
	},
	methods: {
		async fetchtmbscan(flist) {
			// not cached thumbnails
			const uncached = [];
			// get embedded thumbnails at first
			for (const file of flist) {
				if ((this.$root.access || file.free) && file.type === FT.file && !file.etmb && !file.mtmb) {
					uncached.push({ puid: file.puid, tm: -1 });
				}
			}
			// then get rendered thumbnails
			for (const file of flist) {
				if ((this.$root.access || file.free) && file.type === FT.file && !file.mtmb) {
					uncached.push({ puid: file.puid, tm: 0 });
				}
			}
			if (!uncached.length) {
				return;
			}

			let stop = false;
			const onnewlist = () => {
				stop = true;
			}

			const self = this;
			const gen = (async function* () {
				const response = await fetchjsonauth("POST", `/id${self.$root.aid}/api/tile/scnstart`, {
					list: uncached,
				});
				traceajax(response);
				if (!response.ok) {
					const data = await response.json();
					throw new HttpError(response.status, data);
				}

				yield;

				let ul = uncached.length;
				let uc = 0;
				// cache folder thumnails
				while (true) {
					if (stop || !self.expanded) {
						const response = await fetchjsonauth("POST", `/id${self.$root.aid}/api/tile/scnbreak`, {
							list: uncached,
						});
						traceajax(response);
						if (!response.ok) {
							const data = await response.json();
							throw new HttpError(response.status, data);
						}
						return;
					}
					const response = await fetchjsonauth("POST", `/id${self.$root.aid}/api/tile/chk`, {
						list: uncached
					});
					const data = await response.json();
					traceajax(response, data);
					if (!response.ok) {
						throw new HttpError(response.status, data);
					}

					uncached.splice(0); // refill uncached
					for (const tp of data.list) {
						if (tp.mime) {
							for (const file of flist) {
								if (file.puid === tp.puid) {
									if (tp.tm == -1) {
										file.etmb = tp.mime; // Vue.set
									} else {
										file.mtmb = tp.mime; // Vue.set
									}
									break;
								}
							}
						} else {
							uncached.push(tp)
						}
					}
					// check cached state loop
					if (!uncached.length) {
						return;
					}
					// check for some files remains uncached
					if (uncached.length >= ul) {
						uc++;
						if (uc > 9) {
							return;
						}
					} else {
						ul = uncached.length;
						uc = 0;
					}

					yield;
				}
			})();

			this.flisthub.on(null, onnewlist);
			await (async () => {
				while (true) {
					const ret = await gen.next();
					if (ret.done) {
						return;
					}
					// waits before new checkup iteration
					await new Promise(resolve => setTimeout(resolve, 1500));
				}
			})()
			this.flisthub.off(null, onnewlist);
		},

		onnewlist(newlist, oldlist) {
			(async () => {
				try {
					await this.fetchtmbscan(newlist); // fetch at backround
				} catch (e) {
					ajaxfail(e);
				}
			})();
		},

		onorder() {
			this.sortorder = -this.sortorder;
			storageSetItem("card.file.sortorder", this.sortorder);
		},
		onsortalpha() {
			this.sortmode = 'byalpha';
			storageSetItem("card.file.sortmode", this.sortmode);
		},
		onsorttime() {
			this.sortmode = 'bytime';
			storageSetItem("card.file.sortmode", this.sortmode);
		},
		onsortsize() {
			this.sortmode = 'bysize';
			storageSetItem("card.file.sortmode", this.sortmode);
		},
		onsortunsorted() {
			this.sortmode = 'unsorted';
			storageSetItem("card.file.sortmode", this.sortmode);
		},
		onlistmodels() {
			this.listmode = 'xs';
			storageSetItem("card.file.listmode", this.listmode);
		},
		onlistmodesm() {
			this.listmode = 'sm';
			storageSetItem("card.file.listmode", this.listmode);
		},
		onlistmodemd() {
			this.listmode = 'md';
			storageSetItem("card.file.listmode", this.listmode);
		},
		onlistmodelg() {
			this.listmode = 'lg';
			storageSetItem("card.file.listmode", this.listmode);
		},

		onthumbmode() {
			thumbmode = !thumbmode;
			storageSetItem('thumbmode', thumbmode);
			eventHub.emit('thumbmode', thumbmode);
		},
		onheadset() {
			storageSetItem('audioonly', !this.audioonly);
			eventHub.emit('audioonly', !this.audioonly);
		},

		onaudio() {
			this.fgshow[FG.audio] = !this.fgshow[FG.audio];
			storageSetItem('card.file.fgshow', this.fgshow);
		},
		onvideo() {
			this.fgshow[FG.video] = !this.fgshow[FG.video];
			storageSetItem('card.file.fgshow', this.fgshow);
		},
		onphoto() {
			this.fgshow[FG.image] = !this.fgshow[FG.image];
			storageSetItem('card.file.fgshow', this.fgshow);
		},
		onbooks() {
			this.fgshow[FG.books] = !this.fgshow[FG.books];
			storageSetItem('card.file.fgshow', this.fgshow);
		},
		ontexts() {
			this.fgshow[FG.texts] = !this.fgshow[FG.texts];
			storageSetItem('card.file.fgshow', this.fgshow);
		},
		onpacks() {
			this.fgshow[FG.packs] = !this.fgshow[FG.packs];
			storageSetItem('card.file.fgshow', this.fgshow);
		},
		onother() {
			this.fgshow[FG.other] = !this.fgshow[FG.other];
			storageSetItem('card.file.fgshow', this.fgshow);
		},

		onselect(file) {
			eventHub.emit('select', file);
		},
		onopen(file) {
			eventHub.emit('open', file, this.sortedlist);
		},
		onunselect() {
			eventHub.emit('select', null);
		},

		onexpand(e) {
			this.expanded = true;
			storageSetItem("card.ctgr.expanded", this.expanded);
			this.expand();
		},
		oncollapse(e) {
			this.expanded = false;
			storageSetItem("card.ctgr.expanded", this.expanded);
			this.collapse();
		},
		expand() {
			(async () => {
				try {
					await this.fetchtmbscan(this.flist); // fetch at backround
				} catch (e) {
					ajaxfail(e);
				}
			})();
		},
		collapse() {
		},

		_thumbmode(val) {
			this.thumbmode = val;
		},
		_audioonly(val) {
			this.audioonly = val;
		},
	},
	created() {
		this.sortorder = storageGetItem("card.file.sortorder", this.sortorder);
		this.sortmode = storageGetItem("card.file.sortmode", this.sortmode);
		this.listmode = storageGetItem("card.file.listmode", this.listmode);
		this.expanded = storageGetItem("card.file.expanded", this.expanded);
		this.thumbmode = storageGetItem("thumbmode", this.thumbmode);
		this.audioonly = storageGetItem('audioonly', this.audioonly);
		this.fgshow = storageGetItem('card.file.fgshow', this.fgshow);
	},
	mounted() {
		eventHub.on('thumbmode', this._thumbmode);
		eventHub.on('audioonly', this._audioonly);
		const el = document.getElementById('card' + this.iid);
		if (el) {
			if (this.expanded) { 
				el.classList.add('show');
				this.expand();
			} else {
				this.collapse();
			}
			el.addEventListener('shown.bs.collapse', this.onexpand);
			el.addEventListener('hidden.bs.collapse', this.oncollapse);
		}
	},
	unmounted() {
		eventHub.off('thumbmode', this._thumbmode);
		eventHub.off('audioonly', this._audioonly);
		const el = document.getElementById('card' + this.iid);
		if (el) {
			el.removeEventListener('shown.bs.collapse', this.onexpand);
			el.removeEventListener('hidden.bs.collapse', this.oncollapse);
		}
	},
};

// set of possible sizes for horizontal images
const htiles = [
	[2, 2], // 0
	[3, 3], // 1
	[4, 4], // 2
	[5, 5], // 3
	[6, 6], // 4
];
// set of possible sizes for vertical images
const vtiles = [
	[2, 4],
	[3, 6],
];
// indexes in htiles depended on free space size
const tilemode246 = [
	[0, 2, 4], // for all
	[], // 1
	[0], // 2
	[], // 3
	[0, 2], // 4
	[], // 5
	[0, 2, 4], // 6
];
const tilemode234 = [
	[0, 1, 2], // for all
	[], // 1
	[0], // 2
	[1], // 3
	[0, 2], // 4
	[0, 1], // 5
];
const tilemode26 = [
	[0, 4], // for all
	[], // 1
	[0], // 2
	[], // 3
	[0], // 4
	[], // 5
	[0, 4], // 6
];
const tilemode2346 = [
	[0, 1, 2, 4], // for all
	[], // 1
	[0], // 2
	[1], // 3
	[0, 2], // 4
	[0, 1], // 5
	[0, 1, 2, 4], // 6
	[0, 1, 2], // 7
];
const tilemode444 = [
	[2], // for all
	[], // 1
	[], // 2
	[], // 3
	[2], // 4
	[], // 5
	[], // 6
];
const tilemode3333 = [
	[1], // for all
	[], // 1
	[], // 2
	[1], // 3
	[], // 4
	[], // 5
	[1], // 6
];
const tilemodetype = {
	'mode-246': tilemode246,
	'mode-234': tilemode234,
	'mode-26': tilemode26,
	'mode-2346': tilemode2346,
	'mode-444': tilemode444,
	'mode-3333': tilemode3333,
};
const rollbacktiles = 11;

const maketileslide = (list, tilemode) => {
	const tiles = [];
	const sheet = [];
	let fill = 0; // filled line
	const randomint = max => {
		return Math.floor(Math.random() * max);
	};
	const addline = () => {
		sheet.push([0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0]);
	};
	const zeropos = y => {
		for (let x = 0; x < 12; x++) {
			if (!sheet[y][x]) {
				return x;
			}
		}
		return -1;
	};
	const posfill = () => {
		while (fill < sheet.length) {
			const pos = zeropos(fill);
			if (pos >= 0) {
				return pos;
			}
			fill++;
		}
		addline();
		return 0;
	};
	const sizefill = pf => {
		for (let i = pf + 1; i < 12; i++) {
			if (sheet[fill][i]) {
				return i - pf;
			}
		}
		return 12 - pf;
	};

	let rollbackcount = 0;
	addline();
	let ti = 0;
	for (; ;) {
		for (; ti < list.length; ti++) {
			const pf = posfill();
			const sf = sizefill(pf);
			const im = tilemode[sf < tilemode.length ? sf : 0];
			let ts = htiles[im[randomint(im.length)]];
			const nl = ts[1] - sheet.length + fill;
			for (let i = 0; i < nl; i++) {
				addline();
			}
			for (let x = 0; x < ts[0]; x++) {
				for (let y = 0; y < ts[1]; y++) {
					sheet[fill + y][x + pf] = Number(ti) + 1;
				}
			}

			tiles.push({
				file: list[ti],
				px: pf,
				py: fill,
				sx: ts[0],
				sy: ts[1],
			});
		}

		if (tiles.length < 6 || zeropos(sheet.length - 1) < 0) {
			break;
		}

		// rollback
		if (rollbackcount > 120) {
			console.error("rollback overflows, sheet remains with spaces");
			break;
		}
		if (tiles.length > rollbacktiles) {
			ti = tiles.length - rollbacktiles;
		} else {
			ti = 0;
		}
		// remove from sheet
		fill = tiles[ti].py;
		for (let y = fill; y < sheet.length; y++) {
			for (let x = 0; x < 12; x++) {
				if (sheet[y][x] > ti) {
					sheet[y][x] = 0;
				}
			}
		}
		// remove zero lines
		delzero: for (let y = sheet.length - 1; y >= tiles[ti].py; y--) {
			for (let x = 0; x < 12; x++) {
				if (sheet[y][x] !== 0) {
					break delzero;
				}
			}
			sheet.splice(y, 1);
		}
		// remove tiles
		tiles.splice(ti, rollbacktiles);
		rollbackcount++;
	}
	console.info("rollback count:", rollbackcount)

	return { tiles, sheet };
}

const VueTileCard = {
	template: '#tile-card-tpl',
	props: ["flist"],
	data() {
		return {
			sortorder: 1,
			sortmode: 'byalpha',
			tilemode: 'mode-246',
			tiles: [],
			sheet: [],

			flisthub: makeeventhub(),
			expanded: true,
			iid: makestrid(10), // instance ID
		};
	},
	watch: {
		flist: {
			handler(newlist, oldlist) {
				this.onrebuild();
			}
		},
	},
	computed: {
		photolist() {
			const res = [];
			for (const file of this.flist) {
				const ext = pathext(file.name);
				if (typeTile[ext] && file.size > 512*1024) {
					res.push(file);
				}
			}
			if (this.sortmode === 'byalpha') {
				res.sort((v1, v2) => {
					return v1.name.toLowerCase() > v2.name.toLowerCase() ? 1 : -1;
				});
			} else if (this.sortmode === 'bytime') {
				res.sort((v1, v2) => {
					const t1 = v1.datetime ?? v1.time;
					const t2 = v2.datetime ?? v2.time;
					if (t1 !== t2) {
						return t1 > t2 ? 1 : -1;
					}
					return v1.name.toLowerCase() > v2.name.toLowerCase() ? 1 : -1;
				});
			} else if (this.sortmode === 'bysize') {
				res.sort((v1, v2) => {
					const s1 = v1.size ?? 0;
					const s2 = v2.size ?? 0;
					if (s1 !== s2) {
						return this.sortorder * (s1 > s2 ? 1 : -1);
					}
					return v1.name.toLowerCase() > v2.name.toLowerCase() ? 1 : -1;
				});
			}
			return res;
		},
		isvisible() {
			return this.tiles.length > 0;
		},

		sheetbl() {
			let tbl = [];
			let row0 = [];
			for (let line = 0; line < this.sheet.length; line++) {
				const tr = [];
				const row1 = this.sheet[this.sortorder > 0 ? line : this.sheet.length - line - 1];
				for (let x = 0; x < 12; x++) {
					if (row1[x] && row1[x] !== row1[x - 1] && row0[x] !== row1[x]) {
						const tile = this.tiles[row1[x] - 1];
						tr.push(tile);
					}
				}
				row0 = row1;
				tbl.push(tr);
			}
			return tbl;
		},

		hintorder() {
			return this.sortorder > 0
				? "direct order"
				: "reverse order";
		},
		hintsortmode() {
			return sortmodehint[this.sortmode];
		},
		hinttilemode() {
			return this.tilemode;
		},

		clsorder() {
			return this.sortorder > 0
				? 'arrow_downward'
				: 'arrow_upward';
		},
		clssortmode() {
			return sortmodeicon[this.sortmode];
		},
		clsmode246() {
			return { active: this.tilemode === 'mode-246' };
		},
		clsmode234() {
			return { active: this.tilemode === 'mode-234' };
		},
		clsmode26() {
			return { active: this.tilemode === 'mode-26' };
		},
		clsmode2346() {
			return { active: this.tilemode === 'mode-2346' };
		},
		clsmode444() {
			return { active: this.tilemode === 'mode-444' };
		},
		clsmode3333() {
			return { active: this.tilemode === 'mode-3333' };
		},
		expandchevron() {
			return this.expanded ? 'expand_more' : 'chevron_right';
		},
	},
	methods: {
		async fetchtilescan() {
			// not cached tiles
			const uncached = [];
			for (const tile of this.tiles) {
				const fld = 'mt' + (tile.sx * wdhmult < 10 ? '0' : '') + tile.sx * wdhmult;
				if ((this.$root.access || tile.file.free) && tile.file.type === FT.file && !tile.file[fld]) {
					uncached.push({ puid: tile.file.puid, tm: tile.sx * wdhmult });
				}
			}
			if (!uncached.length) {
				return;
			}

			let stop = false;
			const onrebuild = () => {
				stop = true;
			}

			const self = this;
			const gen = (async function* () {
				const response = await fetchjsonauth("POST", `/id${self.$root.aid}/api/tile/scnstart`, {
					list: uncached,
				});
				traceajax(response);
				if (!response.ok) {
					const data = await response.json();
					throw new HttpError(response.status, data);
				}

				yield;

				// cache folder tiles
				while (true) {
					if (stop || !self.expanded) {
						const response = await fetchjsonauth("POST", `/id${self.$root.aid}/api/tile/scnbreak`, {
							list: uncached,
						});
						traceajax(response);
						if (!response.ok) {
							const data = await response.json();
							throw new HttpError(response.status, data);
						}
						return;
					}
					const response = await fetchjsonauth("POST", `/id${self.$root.aid}/api/tile/chk`, {
						list: uncached
					});
					const data = await response.json();
					traceajax(response, data);
					if (!response.ok) {
						throw new HttpError(response.status, data);
					}

					uncached.splice(0); // refill uncached
					for (const tp of data.list) {
						if (tp.mime) {
							for (const tile of self.tiles) {
								if (tile.file.puid === tp.puid) {
									const fld = 'mt' + (tp.tm < 10 ? '0' : '') + tp.tm;
									tile.file[fld] = tp.mime; // Vue.set
									break;
								}
							}
						} else {
							uncached.push(tp)
						}
					}
					// check cached state loop
					if (!uncached.length) {
						return;
					}

					yield;
				}
			})();

			this.flisthub.on(null, onrebuild);
			await (async () => {
				while (true) {
					const ret = await gen.next();
					if (ret.done) {
						return;
					}
					// waits before new checkup iteration
					await new Promise(resolve => setTimeout(resolve, 1500));
				}
			})()
			this.flisthub.off(null, onrebuild);
		},

		onwdhmult() {
			(async () => {
				try {
					await this.fetchtilescan(); // fetch at backround
				} catch (e) {
					ajaxfail(e);
				}
			})();
		},
		onrebuild() {
			const ret = maketileslide(this.photolist, tilemodetype[this.tilemode]);
			this.tiles = ret.tiles;
			this.sheet = ret.sheet;
			this.flisthub.emit(null);
			(async () => {
				try {
					await this.fetchtilescan(); // fetch at backround
				} catch (e) {
					ajaxfail(e);
				}
			})();
		},

		onorder() {
			this.sortorder = -this.sortorder;
			storageSetItem("card.tile.sortorder", this.sortorder);
		},
		onsortalpha() {
			this.sortmode = 'byalpha';
			storageSetItem("card.tile.sortmode", this.sortmode);
		},
		onsorttime() {
			this.sortmode = 'bytime';
			storageSetItem("card.tile.sortmode", this.sortmode);
		},
		onsortsize() {
			this.sortmode = 'bysize';
			storageSetItem("card.tile.sortmode", this.sortmode);
		},
		onsortunsorted() {
			this.sortmode = 'unsorted';
			storageSetItem("card.tile.sortmode", this.sortmode);
		},
		onmode246() {
			this.tilemode = 'mode-246';
			storageSetItem("card.tile.tilemode", this.tilemode);
		},
		onmode234() {
			this.tilemode = 'mode-234';
			storageSetItem("card.tile.tilemode", this.tilemode);
		},
		onmode26() {
			this.tilemode = 'mode-26';
			storageSetItem("card.tile.tilemode", this.tilemode);
		},
		onmode2346() {
			this.tilemode = 'mode-2346';
			storageSetItem("card.tile.tilemode", this.tilemode);
		},
		onmode444() {
			this.tilemode = 'mode-444';
			storageSetItem("card.tile.tilemode", this.tilemode);
		},
		onmode3333() {
			this.tilemode = 'mode-3333';
			storageSetItem("card.tile.tilemode", this.tilemode);
		},

		onopen(file) {
			eventHub.emit('select', file);
			eventHub.emit('open', file, this.photolist);
		},
		onexpand(e) {
			this.expanded = true;
			storageSetItem("card.ctgr.expanded", this.expanded);
			this.expand();
		},
		oncollapse(e) {
			this.expanded = false;
			storageSetItem("card.ctgr.expanded", this.expanded);
			this.collapse();
		},
		expand() {
			(async () => {
				try {
					await this.fetchtilescan(); // fetch at backround
				} catch (e) {
					ajaxfail(e);
				}
			})();
		},
		collapse() {
		},
	},
	created() {
		this.sortorder = storageGetItem("card.tile.sortorder", this.sortorder);
		this.sortmode = storageGetItem("card.tile.sortmode", this.sortmode);
		this.tilemode = storageGetItem("card.tile.tilemode", this.tilemode);
		this.expanded = storageGetItem("card.tile.expanded", this.expanded);
	},
	mounted() {
		eventHub.on('wdhmult', this.onwdhmult);
		const el = document.getElementById('card' + this.iid);
		if (el) {
			if (this.expanded) { 
				el.classList.add('show');
				this.expand();
			} else {
				this.collapse();
			}
			el.addEventListener('shown.bs.collapse', this.onexpand);
			el.addEventListener('hidden.bs.collapse', this.oncollapse);
		}
	},
	unmounted() {
		eventHub.off('wdhmult', this.onwdhmult);

		const el = document.getElementById('card' + this.iid);
		if (el) {
			el.removeEventListener('shown.bs.collapse', this.onexpand);
			el.removeEventListener('hidden.bs.collapse', this.oncollapse);
		}
	},
};

const VueMapCard = {
	template: '#map-card-tpl',
	props: ["flist"],
	data() {
		return {
			isfullscreen: false,
			styleid: 'mapbox-hybrid',
			markermode: 'thumb',
			showtrack: false,
			tracknum: 0,
			mapmode: mm.view,
			bounds: L.latLngBounds([]),
			iszooming: false,
			zoomlevel: 8,

			map: null, // set it on mounted event
			tiles: null,
			markers: null,
			phototrack: null,
			tracks: null,
			gpslist: [],

			im: [],
			flisthub: makeeventhub(),
			pinned: false,
			expanded: true,
			iid: makestrid(10), // instance ID
		};
	},
	watch: {
		flist: {
			handler(newlist, oldlist) {
				this.flisthub.emit(null);
				this.pinned = this.$root.curpuid === CNID.map;
				this.onnewlist(newlist, oldlist);
			}
		},
		mapmode: {
			handler(val, oldval) {
				switch (val) {
					case mm.view:
						this.toviewmode();
						break;
					case mm.draw:
						this.todrawmode();
						break;
					case mm.remove:
						this.toremovemode();
						break;
					}
			}
		},
	},
	computed: {
		isvisible() {
			return this.pinned || this.gpslist.length > 0 || this.tracknum > 0;
		},
		clsmapboxhybrid() {
			return { active: this.styleid === 'mapbox-hybrid' };
		},
		clsmapboxsatellite() {
			return { active: this.styleid === 'mapbox-satellite' };
		},
		clsmapboxoutdoors() {
			return { active: this.styleid === 'mapbox-outdoors' };
		},
		clsmapboxstreets() {
			return { active: this.styleid === 'mapbox-streets' };
		},
		clsgooglehybrid() {
			return { active: this.styleid === 'google-hybrid' };
		},
		clsgooglesatellite() {
			return { active: this.styleid === 'google-satellite' };
		},
		clsgoogleterrain() {
			return { active: this.styleid === 'google-terrain' };
		},
		clsgooglestreets() {
			return { active: this.styleid === 'google-streets' };
		},
		clsosm() {
			return { active: this.styleid === 'osm' };
		},
		clscyclosm() {
			return { active: this.styleid === 'cyclosm' };
		},
		clsopentopomap() {
			return { active: this.styleid === 'opentopo' };
		},
		clsersiimg() {
			return { active: this.styleid === 'ersiimg' };
		},
		clsersitopo() {
			return { active: this.styleid === 'ersitopo' };
		},
		clsersistreet() {
			return { active: this.styleid === 'ersistreet' };
		},
		cls2gis() {
			return { active: this.styleid === '2gis' };
		},
		clsphototrack() {
			return { active: this.showtrack };
		},

		clsfitbounds() {
			return { 'leaflet-disabled': !this.bounds.isValid() };
		},
		clsdrawmode() {
			return { 
				'active': this.mapmode === mm.draw,
				'leaflet-disabled': this.iszooming || this.zoomlevel < 6,
			};
		},
		clsremovemode() {
			return { 'active': this.mapmode === mm.remove };
		},
		iconscreen() {
			return this.isfullscreen ? 'zoom_in_map' : 'zoom_out_map';
		},
		iconmarkermode() {
			switch (this.markermode) {
				case 'marker': return 'place';
				case 'thumb': return 'local_see';
			}
		},
		hintlandscape() {
			switch (this.styleid) {
				case 'mapbox-hybrid':
					return "Mapbox satellite & streets map";
				case 'mapbox-satellite':
					return "Mapbox satellite map";
				case 'mapbox-outdoors':
					return "Mapbox outdoors map";
				case 'mapbox-streets':
					return "Mapbox streets map";
				case 'google-hybrid':
					return "Google maps hybrid";
				case 'google-satellite':
					return "Google maps satellite";
				case 'google-terrain':
					return "Google maps terrain";
				case 'google-streets':
					return "Google maps streets";
				case 'osm':
					return "Open Street Map";
				case 'cyclosm':
					return "CyclOSM - Open Bicycle render";
				case 'opentopo':
					return "Open topo map";
				case 'ersiimg':
					return "Esri World Imagery";
				case 'ersitopo':
					return "Esri World Topo map";
				case 'ersistreet':
					return "Esri World Street map";
				case '2gis':
					return "2GIS map";
			}
		},
		hintmarkermode() {
			switch (this.markermode) {
				case 'marker': return "photo positions by markers";
				case 'thumb': return "photo positions by thumbnails";
			}
		},
		expandchevron() {
			return this.expanded ? 'expand_more' : 'chevron_right';
		},
	},
	methods: {
		async fetchgpsscan(flist) {
			const response = await fetchjsonauth("POST", `/id${this.$root.aid}/api/gps/scan`, {
				list: flist.map(file => file.puid),
			});
			const data = await response.json();
			traceajax(response, data);
			if (!response.ok) {
				throw new HttpError(response.status, data);
			}
			for (const gsi of data.list || []) {
				for (const file of flist) {
					if (gsi.puid === file.puid) {
						file.latitude = gsi.prop.lat;
						file.longitude = gsi.prop.lon;
						file.altitude = gsi.prop.alt;
						file.datetime = gsi.prop.time;
						break;
					}
				}
			}
		},

		async fetchtmbscan(flist) {
			const mlist = [];
			for (const file of flist) {
				if (!file.mtmb && (file.latitude || file.longitude)) {
					if (!this.hasmarker(file.puid)) {
						mlist.push(file);
					}
				}
			}

			// not cached thumbnails
			const uncached = [];
			// get embedded thumbnails at first
			for (const file of mlist) {
				if ((this.$root.access || file.free) && !file.etmb && !file.mtmb) {
					uncached.push({ puid: file.puid, tm: -1 });
				}
			}
			// then get rendered thumbnails
			for (const file of mlist) {
				if (!file.mtmb) {
					uncached.push({ puid: file.puid, tm: 0 });
				}
			}
			if (!uncached.length) {
				return;
			}

			let stop = false;
			const onnewlist = () => {
				stop = true;
			}

			const self = this;
			const gen = (async function* () {
				const response = await fetchjsonauth("POST", `/id${self.$root.aid}/api/tile/scnstart`, {
					list: uncached,
				});
				traceajax(response);
				if (!response.ok) {
					const data = await response.json();
					throw new HttpError(response.status, data);
				}

				yield;

				let ul = uncached.length;
				let uc = 0;
				// cache folder thumnails
				while (true) {
					if (stop || !self.expanded) {
						const response = await fetchjsonauth("POST", `/id${self.$root.aid}/api/tile/scnbreak`, {
							list: uncached,
						});
						traceajax(response);
						if (!response.ok) {
							const data = await response.json();
							throw new HttpError(response.status, data);
						}
						return;
					}
					const response = await fetchjsonauth("POST", `/id${self.$root.aid}/api/tile/chk`, {
						list: uncached
					});
					const data = await response.json();
					traceajax(response, data);
					if (!response.ok) {
						throw new HttpError(response.status, data);
					}

					const gpslist = [];
					uncached.splice(0); // refill uncached
					for (const tp of data.list) {
						if (tp.mime) {
							for (const file of mlist) {
								if (file.puid === tp.puid) {
									if (tp.tm == -1) {
										file.etmb = tp.mime; // Vue.set
									} else {
										file.mtmb = tp.mime; // Vue.set
									}
									gpslist.push(file); // add new marker
									break;
								}
							}
						} else {
							uncached.push(tp)
						}
					}
					self.addmarkers(gpslist);
					// check cached state loop
					if (!uncached.length) {
						return;
					}
					// check for some files remains uncached
					if (uncached.length >= ul) {
						uc++;
						if (uc > 9) {
							// add remain markers without thumbnails as is
							const gpslist = [];
							for (const tp of uncached) {
								for (const file of mlist) {
									if (file.puid === tp.puid) {
										gpslist.push(file);
										break;
									}
								}
							}
							self.addmarkers(gpslist);
							return;
						}
					} else {
						ul = uncached.length;
						uc = 0;
					}

					yield;
				}
			})();

			this.flisthub.on(null, onnewlist);
			await (async () => {
				while (true) {
					const ret = await gen.next();
					if (ret.done) {
						return;
					}
					// waits before new checkup iteration
					await new Promise(resolve => setTimeout(resolve, 1500));
				}
			})()
			this.flisthub.off(null, onnewlist);
		},

		// create new opened folder
		onnewlist(newlist, oldlist) {
			// new empty list
			this.gpslist = [];
			// remove all markers from the cluster
			if (this.markers) {
				this.markers.clearLayers();
			}
			// remove previous track
			this.phototrack.setLatLngs([]);
			// remove all gpx-tracks
			this.tracks.clearLayers();
			// no any gpx on map
			this.tracknum = 0;

			// prepare list of files to get coords
			const gpsget = [];
			for (const file of newlist) {
				if (file.type !== FT.file || file.latitude || file.longitude) {
					continue;
				}
				const ext = pathext(file.name);
				if (typeEXIF[ext]) {
					gpsget.push(file);
				} else if (ext === ".gpx") {
					this.addgpx(file);
				}
			}

			// update map card with incoming files
			(async () => {
				try {
					if (gpsget.length) {
						await this.fetchgpsscan(gpsget); // fetch at backround
					}
					const gpslist = [];
					for (const file of newlist) {
						if (file.mtmb && (file.latitude || file.longitude)) {
							if (!this.hasmarker(file.puid)) {
								gpslist.push(file);
							}
						}
					}
					this.addmarkers(gpslist);
					await this.fetchtmbscan(newlist);
				} catch (e) {
					ajaxfail(e);
				}
			})();
		},

		hasmarker(puid) {
			for (const file of this.gpslist) {
				if (file.puid === puid) {
					return true;
				}
			}
			return false;
		},

		makemarkericon(file) {
			const res = geticonpath(this.im, file);
			const icp = res.org || res.alt;
			let src = "";
			if (Number(file.mtmb) > 0 && thumbmode) {
				src = `<source srcset="/id${this.$root.aid}/mtmb/${file.puid}" type="${MimeStr[file.mtmb]}">`;
			} else if (Number(file.etmb) > 0 && thumbmode) {
				src = `<source srcset="/id${this.$root.aid}/etmb/${file.puid}" type="${MimeStr[file.etmb]}">`;
			} else {
				for (const fmt of this.im.iconfmt) {
					src += `<source srcset="${icp + fmt.ext}" type="${fmt.mime}">`;
				}
			}
			return imagetmpl(src);
		},

		makemarkerpopup(file) {
			const res = geticonpath(this.im, file);
			const icp = res.org || res.alt;
			let src = "";
			if (Number(file.mtmb) > 0 && thumbmode) {
				src = `<source srcset="/id${this.$root.aid}/mtmb/${file.puid}" type="${MimeStr[file.mtmb]}">`;
			} else if (Number(file.etmb) > 0 && thumbmode) {
				src = `<source srcset="/id${this.$root.aid}/etmb/${file.puid}" type="${MimeStr[file.etmb]}">`;
			} else {
				for (const fmt of this.im.iconfmt) {
					src += `<source srcset="${icp + fmt.ext}" type="${fmt.mime}">`;
				}
			}
			return photoinfotmpl(file, src);
		},

		// make tiles layer
		// see: https://leaflet-extras.github.io/leaflet-providers/preview/
		maketiles(id) {
			if (this.styleid !== id) {
				this.styleid = id;
				storageSetItem("card.map.styleid", id);
			}
			switch (id) {
				case 'mapbox-hybrid':
					return L.tileLayer('https://api.mapbox.com/styles/v1/{id}/tiles/{z}/{x}/{y}?access_token={accessToken}', {
						attribution: 'Map data: &copy; <a href="https://www.openstreetmap.org/copyright" target="_blank">OpenStreetMap</a> contributors, ' +
							'Imagery &copy <a href="https://www.mapbox.com/" target="_blank">Mapbox</a>',
						tileSize: 512,
						minZoom: 2,
						zoomOffset: -1,
						id: 'mapbox/satellite-streets-v11',
						accessToken: 'pk.eyJ1Ijoic2Nod2FyemxpY2h0YmV6aXJrIiwiYSI6ImNrazdseXRxZjBlemsycG8wZ3BodTR1aWUifQ.Mt99AhglX08nolRj_bsyog'
					});
				case 'mapbox-satellite':
					return L.tileLayer('https://api.mapbox.com/styles/v1/{id}/tiles/{z}/{x}/{y}?access_token={accessToken}', {
						attribution: 'Map data: &copy; <a href="https://www.openstreetmap.org/copyright" target="_blank">OpenStreetMap</a> contributors, ' +
							'Imagery &copy <a href="https://www.mapbox.com/" target="_blank">Mapbox</a>',
						tileSize: 512,
						minZoom: 2,
						zoomOffset: -1,
						id: 'mapbox/satellite-v9',
						accessToken: 'pk.eyJ1Ijoic2Nod2FyemxpY2h0YmV6aXJrIiwiYSI6ImNrazdseXRxZjBlemsycG8wZ3BodTR1aWUifQ.Mt99AhglX08nolRj_bsyog'
					});
				case 'mapbox-outdoors':
					return L.tileLayer('https://api.mapbox.com/styles/v1/{id}/tiles/{z}/{x}/{y}?access_token={accessToken}', {
						attribution: 'Map data: &copy; <a href="https://www.openstreetmap.org/copyright" target="_blank">OpenStreetMap</a> contributors, ' +
							'Imagery &copy <a href="https://www.mapbox.com/" target="_blank">Mapbox</a>',
						tileSize: 512,
						minZoom: 2,
						zoomOffset: -1,
						id: 'mapbox/outdoors-v11',
						accessToken: 'pk.eyJ1Ijoic2Nod2FyemxpY2h0YmV6aXJrIiwiYSI6ImNrazdseXRxZjBlemsycG8wZ3BodTR1aWUifQ.Mt99AhglX08nolRj_bsyog'
					});
				case 'mapbox-streets':
					return L.tileLayer('https://api.mapbox.com/styles/v1/{id}/tiles/{z}/{x}/{y}?access_token={accessToken}', {
						attribution: 'Map data: &copy; <a href="https://www.openstreetmap.org/copyright" target="_blank">OpenStreetMap</a> contributors, ' +
							'Imagery &copy <a href="https://www.mapbox.com/" target="_blank">Mapbox</a>',
						tileSize: 512,
						minZoom: 2,
						zoomOffset: -1,
						id: 'mapbox/streets-v11',
						accessToken: 'pk.eyJ1Ijoic2Nod2FyemxpY2h0YmV6aXJrIiwiYSI6ImNrazdseXRxZjBlemsycG8wZ3BodTR1aWUifQ.Mt99AhglX08nolRj_bsyog'
					});
				case 'google-hybrid':
					return L.tileLayer('http://{s}.google.com/vt/lyrs=s,h&x={x}&y={y}&z={z}', {
						subdomains: ['mt0', 'mt1', 'mt2', 'mt3'],
						attribution: 'Map data: &copy; <a href="https://developers.google.com/maps/documentation/javascript/overview" target="_blank">Google Maps Platform</a> contributors',
						minZoom: 2,
						maxZoom: 20
					});
				case 'google-satellite':
					return L.tileLayer('http://{s}.google.com/vt/lyrs=s&x={x}&y={y}&z={z}', {
						subdomains: ['mt0', 'mt1', 'mt2', 'mt3'],
						attribution: 'Map data: &copy; <a href="https://developers.google.com/maps/documentation/javascript/overview" target="_blank">Google Maps Platform</a> contributors',
						minZoom: 2,
						maxZoom: 20
					});
				case 'google-terrain':
					return L.tileLayer('http://{s}.google.com/vt/lyrs=p&x={x}&y={y}&z={z}', {
						subdomains: ['mt0', 'mt1', 'mt2', 'mt3'],
						attribution: 'Map data: &copy; <a href="https://developers.google.com/maps/documentation/javascript/overview" target="_blank">Google Maps Platform</a> contributors',
						minZoom: 2,
						maxZoom: 20
					});
				case 'google-streets':
					return L.tileLayer('http://{s}.google.com/vt/lyrs=m&x={x}&y={y}&z={z}', {
						subdomains: ['mt0', 'mt1', 'mt2', 'mt3'],
						attribution: 'Map data: &copy; <a href="https://developers.google.com/maps/documentation/javascript/overview" target="_blank">Google Maps Platform</a> contributors',
						minZoom: 2,
						maxZoom: 20
					});
				case 'osm':
					return L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
						maxZoom: 19,
						attribution: 'Map data: &copy; <a href="https://www.openstreetmap.org/copyright" target="_blank">OpenStreetMap</a> contributors'
					});
				case 'cyclosm':
					return L.tileLayer('https://{s}.tile-cyclosm.openstreetmap.fr/cyclosm/{z}/{x}/{y}.png', {
						maxZoom: 20,
						attribution: 'Map data: &copy; <a href="https://www.openstreetmap.org/copyright" target="_blank">OpenStreetMap</a> contributors'
					});
				case 'opentopo':
					return L.tileLayer('https://{s}.tile.opentopomap.org/{z}/{x}/{y}.png', {
						attribution: 'Map data: &copy; <a href="https://www.openstreetmap.org/copyright" target="_blank">OpenStreetMap</a> contributors, ' +
							'<a href="http://viewfinderpanoramas.org" target="_blank">SRTM</a> | Map style: &copy; <a href="https://opentopomap.org" target="_blank">OpenTopoMap</a>',
						minZoom: 2,
						maxZoom: 17
					});
				case 'ersiimg':
					return L.tileLayer('https://server.arcgisonline.com/ArcGIS/rest/services/World_Imagery/MapServer/tile/{z}/{y}/{x}', {
						attribution: 'Tiles &copy; Esri &mdash; Source: Esri, i-cubed, USDA, USGS, AEX, GeoEye, Getmapping, Aerogrid, IGN, IGP, UPR-EGP, and the GIS User Community',
						minZoom: 2,
						maxZoom: 17
					});
				case 'ersitopo':
					return L.tileLayer('https://server.arcgisonline.com/ArcGIS/rest/services/World_Topo_Map/MapServer/tile/{z}/{y}/{x}', {
						attribution: 'Tiles &copy; Esri &mdash; Esri, DeLorme, NAVTEQ, TomTom, Intermap, iPC, USGS, FAO, NPS, NRCAN, GeoBase, Kadaster NL, Ordnance Survey, Esri Japan, METI, Esri China (Hong Kong), and the GIS User Community',
						minZoom: 2,
						maxZoom: 19
					});
				case 'ersistreet':
					return L.tileLayer('https://server.arcgisonline.com/ArcGIS/rest/services/World_Street_Map/MapServer/tile/{z}/{y}/{x}', {
						attribution: 'Tiles &copy; Esri &mdash; Esri, DeLorme, NAVTEQ, TomTom, Intermap, iPC, USGS, FAO, NPS, NRCAN, GeoBase, Kadaster NL, Ordnance Survey, Esri Japan, METI, Esri China (Hong Kong), and the GIS User Community',
						minZoom: 2,
						maxZoom: 20
					});
				case '2gis':
					return L.tileLayer('http://tile2.maps.2gis.com/tiles?x={x}&y={y}&z={z}', {
						attribution: 'Map data: &copy; <a href="https://2gis.ru/" target="_blank">2gis</a>',
						minZoom: 2,
						maxZoom: 18
					});
			}
		},
		// change tiles layer
		changetiles(id) {
			this.map.removeLayer(this.tiles);
			this.tiles = this.maketiles(id);
			this.map.addLayer(this.tiles);
		},
		// setup markers on map, remove previous
		addmarkers(gpslist) {
			if (!gpslist.length) {
				return;
			}
			if (this.markers) {
				const size = 60;
				for (const file of gpslist) {
					const opt = {
						title: file.name,
						riseOnHover: true
					};
					if (this.markermode === 'thumb' && file.puid) {
						opt.icon = L.divIcon({
							html: this.makemarkericon(file),
							className: "photomarker",
							iconSize: [size, size],
							iconAnchor: [size / 2, size / 2],
							popupAnchor: [0, -size / 4]
						});
					}

					const template = document.createElement('template');
					template.innerHTML = this.makemarkerpopup(file).trim();
					const popup = template.content.firstChild;
					popup.querySelector(".photoinfo picture")?.addEventListener('click', () => {
						eventHub.emit('open', file, this.gpslist);
					});

					L.marker([file.latitude, file.longitude], opt)
						.addTo(this.markers)
						.bindPopup(popup);
				}

				const update = this.gpslist.length == 0;
				Vue.nextTick(() => {
					if (update) {
						this.map.invalidateSize();
					}
					this.onfitbounds();
				});
			}
			this.gpslist.push(...gpslist);
			this.gpslist.sort((a, b) => {
				return a.datetime ?? a.time < b.datetime ?? b.time ? -1 : +1;
			})
			this.buildphototrack();
			this.updatebounds();
		},
		// produces reduced track polyline
		buildphototrack() {
			let last = L.latLng(this.gpslist[0].latitude, this.gpslist[0].longitude, this.gpslist[0].altitude);
			let prev = last;
			let route = 0, trk = 0, asc = 0;
			const latlngs = [last];
			for (const file of this.gpslist) {
				const p = L.latLng(file.latitude, file.longitude, file.altitude);
				const d = last.distanceTo(p);
				if (d > noderadius) {
					route += d;
					if (p.alt > last.alt) {
						asc += p.alt - last.alt;
					}
					latlngs.push(p);
					last = p;
				}
				trk += prev.distanceTo(p);
				prev = p;
			}
			this.phototrack.setLatLngs(latlngs)
				.bindPopup(`total <b>${latlngs.length}</b> waypoints<br>route <b>${route.toFixed()}</b> m<br>track <b>${trk.toFixed()}</b> m<br>ascent <b>${asc.toFixed()}</b> m`)
		},
		// add GPX track polyline
		addgpx(file) {
			const latlngs = [];
			const ci = this.tracknum % gpxcolors.length;

			(async () => {
				eventHub.emit('ajax', +1);
				try {
					const response = await fetch(`/id${this.$root.aid}/file/${file.puid}`);
					const body = await response.text();
					const re = /lat="(\d+\.\d+)" lon="(\d+\.\d+)"/g;
					const matches = body.matchAll(re);
					let prev = null;
					let route = 0;
					for (const m of matches) {
						const p = L.latLng(Number(m[1]), Number(m[2]));
						if (prev) {
							route += p.distanceTo(prev);
						}
						prev = p;
						latlngs.push(p);
					}
					const layer = L.polyline(latlngs, { color: gpxcolors[ci] })
						.bindPopup(`points <b>${latlngs.length}</b><br>route <b>${route.toFixed()}</b> m`)
						.addTo(this.tracks);
					this.tracknum++;
					this.bounds.extend(layer.getBounds());
				} catch (e) {
					ajaxfail(e);
				} finally {
					eventHub.emit('ajax', -1);
				}
			})();
		},
		updatebounds() {
			if (this.markers) {
				this.bounds = this.markers.getBounds();
			} else {
				this.bounds = L.latLngBounds([]);
			}
			this.tracks.eachLayer(layer => {
				this.bounds.extend(layer.getBounds());
			});
		},

		onmapboxhybrid() {
			this.changetiles('mapbox-hybrid');
		},
		onmapboxsatellite() {
			this.changetiles('mapbox-satellite');
		},
		onmapboxoutdoors() {
			this.changetiles('mapbox-outdoors');
		},
		onmapboxstreets() {
			this.changetiles('mapbox-streets');
		},
		ongooglehybrid() {
			this.changetiles('google-hybrid');
		},
		ongooglesatellite() {
			this.changetiles('google-satellite');
		},
		ongoogleterrain() {
			this.changetiles('google-terrain');
		},
		ongooglestreets() {
			this.changetiles('google-streets');
		},
		onosm() {
			this.changetiles('osm');
		},
		oncyclosm() {
			this.changetiles('cyclosm');
		},
		onopentopo() {
			this.changetiles('opentopo');
		},
		onersiimg() {
			this.changetiles('ersiimg');
		},
		onersitopo() {
			this.changetiles('ersitopo');
		},
		on2gis() {
			this.changetiles('2gis');
		},
		onphototrack() {
			if (this.map.hasLayer(this.phototrack)) {
				this.map.removeLayer(this.phototrack);
				this.showtrack = false;
			} else {
				this.map.addLayer(this.phototrack);
				this.showtrack = true;
			}
		},
		onmarkermode() {
			switch (this.markermode) {
				case 'marker':
					this.markermode = 'thumb';
					break;
				case 'thumb':
					this.markermode = 'marker';
					break;
			}
			storageSetItem("card.map.markermode", this.markermode);
			// recreate markers layers
			const gpslist = this.gpslist;
			this.gpslist = [];
			this.markers.clearLayers();
			this.addmarkers(gpslist);
		},
		onfullscreen() {
			if (isFullscreen()) {
				closeFullscreen();
			} else {
				openFullscreen(this.$refs.map);
			}
		},
		onfitbounds() {
			if (this.bounds.isValid()) {
				this.map.flyToBounds(this.bounds, {
					padding: [20, 20]
				});
			}
		},
		ondrawmode() {
			if (this.mapmode === mm.draw) {
				this.mapmode = mm.view;
			} else {
				this.mapmode = mm.draw;
			}
		},
		onremovemode() {
			if (this.mapmode === mm.remove) {
				this.mapmode = mm.view;
			} else {
				this.mapmode = mm.remove;
			}
		},
		onexpand(e) {
			this.expanded = true;
			storageSetItem("card.ctgr.expanded", this.expanded);
			this.expand();
		},
		oncollapse(e) {
			this.expanded = false;
			storageSetItem("card.ctgr.expanded", this.expanded);
			this.collapse();
		},
		expand() {
			(async () => {
				try {
					await this.fetchtmbscan(this.flist); // fetch at backround
				} catch (e) {
					ajaxfail(e);
				}
			})();
		},
		collapse() {
		},

		_iconset(im) {
			this.im = im;
		},
	},
	created() {
		this.styleid = storageGetItem("card.map.styleid", this.styleid);
		this.markermode = storageGetItem("card.map.markermode", this.markermode);
		this.expanded = storageGetItem("card.map.expanded", this.expanded);
		this.im = iconmapping;
	},
	mounted() {
		eventHub.on('iconset', this._iconset);

		const el = document.getElementById('card' + this.iid);
		if (el) {
			if (this.expanded) { 
				el.classList.add('show');
				this.expand();
			} else {
				this.collapse();
			}
			el.addEventListener('shown.bs.collapse', this.onexpand);
			el.addEventListener('hidden.bs.collapse', this.oncollapse);
		}

		this.tiles = this.maketiles('mapbox-hybrid');
		this.tracks = L.layerGroup();
		const mappaths = L.layerGroup();
		const map = L.map(this.$refs.map, {
			attributionControl: true,
			zoomControl: false,
			center: [44.576825, 33.830575],
			zoom: this.zoomlevel,
			layers: [this.tiles, this.tracks, mappaths],
		});
		this.phototrack = L.polyline([], { color: '#3CB371' }); // MediumSeaGreen
		if (this.showtrack) {
			map.addLayer(this.phototrack);
		}

		this.map = map;

		const resizeObserver = new ResizeObserver(entries => {
			this.isfullscreen = isFullscreen();
			// recreate content only if widget is not hidden
			const size = entries[0].contentBoxSize[0].inlineSize;
			if (size > 0) {
				// recreate markers cluster
				if (this.markers) {
					map.removeLayer(this.markers);
				}
				const gpslist = this.gpslist;
				this.gpslist = [];
				this.markers = L.markerClusterGroup();
				this.addmarkers(gpslist);
				map.addLayer(this.markers);
				// update map
				map.invalidateSize();
			}
		});
		resizeObserver.observe(this.$refs.map);

		L.control.scale().addTo(map);
		L.control.zoom({
			zoomInText: '<span class="material-icons">add</span>',
			zoomOutText: '<span class="material-icons">remove</span>'
		}).addTo(map);

		// disable drag and zoom handlers
		const lockmap = () => {
			map.dragging.disable();
			map.touchZoom.disable();
			map.doubleClickZoom.disable();
			map.scrollWheelZoom.disable();
			map.boxZoom.disable();
			map.keyboard.disable();
			if (map.tap) map.tap.disable();
		};

		// enable drag and zoom handlers
		const unlockmap = () => {
			map.dragging.enable();
			map.touchZoom.enable();
			map.doubleClickZoom.enable();
			map.scrollWheelZoom.enable();
			map.boxZoom.enable();
			map.keyboard.enable();
			if (map.tap) map.tap.enable();
		};

		const postmsg = () => {
			const arg = {
				paths: []
			};
			for (const layer of mappaths.getLayers()) {
				arg.paths.push({
					shape: "circle",
					radius: layer.getRadius(),
					coord: [
						{ lat: layer.getLatLng().lat, lon: layer.getLatLng().lng },
					],
				});
			}
			this.$root.rangesearch(arg);
			this.pinned = true;
		};

		const viewmode = () => {
			const onmousemove = e => {
			};

			const evmap = {
				'mousemove': onmousemove,
			};
			const setup = () => {
				map.on(evmap);
			};
			const cancel = () => {
				map.off(evmap);
			};

			setup();
			this.todrawmode = () => {
				cancel();
				drawmode();
			};
			this.toremovemode = () => {
				cancel();
				removemode();
			};
		};

		const drawmode = () => {
			let centerll, centerpt;
			let layer = null;
			let radius = 0;
			let draw = false;

			const onmousedown = e => {
				if (e.originalEvent.which === 1) { // left button click
					centerll = e.latlng;
					centerpt = e.layerPoint;
					draw = true;
				} else {
					draw = false;
					if (layer) {
						mappaths.removeLayer(layer);
						layer = null;
					}
				}
			};
			const onmousemove = e => {
				if (!draw) {
					return;
				}
				if (centerpt.distanceTo(e.layerPoint) > circularmindist) {
					radius = Math.min(centerll.distanceTo(e.latlng), circularmaxradius);
					if (!layer) {
						// start circular
						layer = L.circle(centerll, {
							radius: radius,
							color: 'DodgerBlue',
							fillColor: 'DodgerBlue',
							fillOpacity: 0.3
						});
						const onmouseover = e => {
						};
						const onmouseout = e => {
						};
						const evmap = {
							'mouseover': onmouseover,
							'mouseout': onmouseout,
						};
						layer.on(evmap);
						mappaths.addLayer(layer);
					} else {
						layer.setRadius(radius);
					}
				} else {
					if (layer) { // remove too small circular
						mappaths.removeLayer(layer);
						layer = null;
					}
				}
			};
			const onmouseup = e => {
				if (e.originalEvent.which === 1) {
					if (draw) {
						draw = false;
						if (layer) {
							layer.bindTooltip(`lat: ${centerll.lat.toFixed(6)},\nlon: ${centerll.lng.toFixed(6)},\nradius: ${layer.getRadius().toFixed()}`);
							layer = null;
							postmsg();
						}
					}
				}
			};
			const onkeyup = e => {
				if (e.originalEvent.key === "Escape") {
					e.originalEvent.preventDefault();
					draw = false;
					if (layer) {
						mappaths.removeLayer(layer);
						layer = null;
					}
				}
			};

			const evmap = {
				'mousedown': onmousedown,
				'mousemove': onmousemove,
				'mouseup': onmouseup,
				'keyup': onkeyup,
			};
			const setup = () => {
				lockmap();
				map.on(evmap);
			};
			const cancel = () => {
				map.off(evmap);
				unlockmap();
			};

			setup();
			this.toviewmode = () => {
				cancel();
				viewmode();
			};
			this.toremovemode = () => {
				cancel();
				removemode();
			};
		};

		const removemode = () => {
			const setup = () => {
				for (const layer of mappaths.getLayers()) {
					layer.on('mouseover', e => {
						layer.setStyle({
							color: 'red',
							fillColor: '#f03',
							fillOpacity: 0.3
						});
					});
					layer.on('mouseout', e => {
						layer.setStyle({
							color: 'DodgerBlue',
							fillColor: 'DodgerBlue',
							fillOpacity: 0.3
						});
					});
					layer.on('click', e => {
						mappaths.removeLayer(layer);
						postmsg();
					});
				}
			};
			const cancel = () => {
				for (const layer of mappaths.getLayers()) {
					layer.off('mouseover mouseout click');
				}
			};

			setup();
			this.toviewmode = () => {
				cancel();
				viewmode();
			};
			this.todrawmode = () => {
				cancel();
				drawmode();
			};
		};

		if (this.mapmode === mm.draw) {
			drawmode();
		} else {
			viewmode();
		}

		map.on('zoomstart', e => {
			this.iszooming = true;
		});

		map.on('zoomend', e => {
			this.iszooming = false;
			this.zoomlevel = map.getZoom();
		});
	},
	unmounted() {
		eventHub.off('iconset', this._iconset);

		const el = document.getElementById('card' + this.iid);
		if (el) {
			el.removeEventListener('shown.bs.collapse', this.onexpand);
			el.removeEventListener('hidden.bs.collapse', this.oncollapse);
		}
	},
};

// The End.
