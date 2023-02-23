"use strict";

// Maximum number of buttons at pagination component.
const maxpageitem = 5;

// Log level class.
const LLC = {
	0: "debug",
	1: "info",
	2: "warn",
	3: "error",
	4: "fatal",
	5: "panic",
}

// Extract SQL-query from log message.
const sqllogregex = /^(\[SQL\] (.*) (\[.*\]))/i;

// User Agent structure sample:
// {"Browser":{"Name":1,"Version":{"Major":86,"Minor":0,"Patch":4240}},"OS":{"Platform":1,"Name":2,"Version":{"Major":10,"Minor":0,"Patch":0}},"DeviceType":1}

// Browser name
const BN = [
	"Unknown",
	"Chrome", "IE", "Safari", "Firefox", "Android", "Opera",
	"Blackberry", "UCBrowser", "Silk", "Nokia",
	"NetFront", "QQ", "Maxthon", "SogouExplorer", "Spotify",
	"Nintendo", "Samsung", "Yandex", "CocCoc",
	"Bot", // Bot list begins here
	"AppleBot", "BaiduBot", "BingBot", "DuckDuckGoBot",
	"FacebookBot", "GoogleBot", "LinkedInBot", "MsnBot",
	"PingdomBot", "TwitterBot", "YandexBot", "CocCocBot",
	"YahooBot", // Bot list ends here
];

// OS Name
const OSN = [
	"Unknown",
	"WindowsPhone", "Windows", "MacOSX",
	"iOS", "Android", "Blackberry",
	"ChromeOS", "Kindle", "WebOS", "Linux",
	"Playstation", "Xbox", "Nintendo",
	"Bot",
];

// OS Platform
const OSP = [
	"Unknown",
	"Windows", "Mac", "Linux",
	"iPad", "iPhone", "iPod", "Blackberry", "WindowsPhone",
	"Playstation", "Xbox", "Nintendo",
	"Bot",
];

// Device Type
const DT = [
	"Unknown", "Computer", "Tablet", "Phone", "Console", "Wearable", "TV",
];

const VueStatApp = {
	template: '#app-tpl',
	data() {
		return {
			skinid: "", // ID of skin CSS
			resmodel: { skinlist: [], iconlist: [] },
		};
	},
	computed: {
	},
	methods: {
		setskin(skinid) {
			if (skinid !== this.skinid) {
				for (const v of this.resmodel.skinlist) {
					if (v.id === skinid) {
						document.getElementById('skinmodel')?.setAttribute('href', v.link);
						sessionStorage.setItem('skinid', skinid);
						this.skinid = skinid;
					}
				}
			}
		},
	},
	mounted() {
		eventHub.on('ajax', viewpreloader);

		// load resources and open route
		(async () => {
			eventHub.emit('ajax', +1);
			try {
				// load resources model at first
				const response = await fetch("/fs/assets/resmodel.json");
				if (!response.ok) {
					throw new HttpError(response.status, { what: "can not load resources model file", when: Date.now(), code: 0 });
				}
				this.resmodel = await response.json();

				// set skin
				const skinid = storageGetString('skinid', this.resmodel.defskinid);
				for (const v of this.resmodel.skinlist) {
					if (v.id === skinid) {
						document.getElementById('skinmodel')?.setAttribute('href', v.link);
						this.skinid = skinid;
					}
				}
			} catch (e) {
				ajaxfail(e);
			} finally {
				eventHub.emit('ajax', -1);
			}
		})();

		// hide start-up preloader
		eventHub.emit('ajax', -1);
	},
	unmounted() {
		eventHub.off('ajax', viewpreloader);
	},
}; // end of vue application

const VueCatItem = {
	template: `
<div class="d-inline-flex mx-md-1 catitem">
	<div v-bind:title="!widen&&text" v-on:click="onexpand"><i class="material-icons">{{icon}}</i></div>
	<div v-show="widen" class="ms-md-1">{{text}}</div>
</div>
`,
	props: ["icon", "text", "wide"],
	data() {
		return {
			widen: true
		};
	},
	computed: {
	},
	methods: {
		onexpand() {
			this.widen = !this.widen;
		}
	},
	created() {
		this.widen = this.wide;
	},
};

const VueUser = {
	template: '#user-tpl',
	props: ["user"],
	data() {
		return {
		};
	},
	computed: {
		clsonline() {
			return this.user.online ? 'text-success' : 'text-secondary';
		},
		txtonline() {
			if (this.user.usrid) {
				return this.user.usrid > 0 ? 'person' : 'person_outline';
			} else {
				return this.user.accid ? 'radio_button_checked' : 'radio_button_unchecked';
			}
		},
		txtdevice() {
			switch (this.user.ua.DeviceType) {
				case 1:
					switch (this.user.ua.OS.Platform) {
						case 1: case 8: return 'laptop_windows';
						case 2: case 4: case 5: case 6: return 'laptop_mac';
						case 3: return 'laptop_chromebook';
						default: return 'laptop';
					}
				case 2:
					switch (this.user.ua.OS.Platform) {
						case 2: case 4: case 5: case 6: return 'tablet_mac';
						case 3: return 'tablet_android';
						default: return 'tablet';
					}
				case 3:
					switch (this.user.ua.OS.Platform) {
						case 2: case 4: case 5: case 6: return 'phone_iphone';
						case 3: return 'phone_android';
						default: return 'smartphone';
					}
				case 4: return 'videogame_asset';
				case 5: return 'watch';
				case 6: return 'tv';
				default: return 'device_unknown';
			}
		},
		online() {
			return this.user.online ? 'Online' : 'Offline';
		},
		browser() {
			return `${BN[this.user.ua.Browser.Name]} (${this.user.ua.Browser.Version.Major}.${this.user.ua.Browser.Version.Minor}.${this.user.ua.Browser.Version.Patch})`;
		},
		os() {
			return `${OSN[this.user.ua.OS.Name]} (${this.user.ua.OS.Version.Major}.${this.user.ua.OS.Version.Minor}.${this.user.ua.OS.Version.Patch})`;
		},
		platform() {
			return OSP[this.user.ua.OS.Platform];
		},
		device() {
			return DT[this.user.ua.DeviceType];
		},
	},
	methods: {
	},
};

const VuePagination = {
	template: '#pagination-tpl',
	props: ["num"],
	emits: {
		'page': page => page >= 0 && page < this.num
	},
	data() {
		return {
			view: maxpageitem,
			left: 0,
			sel: 0
		};
	},
	computed: {
		pagelist() {
			const lst = [];
			for (let i = this.left; i < this.left + this.view && i < this.num; i++) {
				lst.push(i);
			}
			return lst;
		},

		disleft() {
			return this.left === 0;
		},
		disright() {
			return this.left === (this.num > this.view ? this.num - this.view : 0);
		},
		clsleft() {
			return { disabled: this.disleft };
		},
		clsright() {
			return { disabled: this.disright };
		},
	},
	methods: {
		clsactive(page) {
			return { active: page === this.sel };
		},

		onpage(page) {
			if (page !== this.sel) {
				this.sel = page;
				this.$emit('page', page);
			}
		},
		onleft() {
			if (this.left <= 0) {
				this.left = 0;
			} else if (this.left + this.view > this.num) {
				this.left = this.num > this.view ? this.num - this.view : 0;
			} else {
				this.left--;
			}
		},
		onright() {
			if (this.left < 0) {
				this.left = 0;
			} else if (this.left + this.view > this.num - 2) {
				this.left = this.num > this.view ? this.num - this.view : 0;
			} else {
				this.left++;
			}
		},
	},
};

const VueSrvinfCard = {
	template: '#srvinf-card-tpl',
	data() {
		return {
			srvinf: {},
			expanded: true,
			iid: makestrid(10), // instance ID
		};
	},
	computed: {
		expandchevron() {
			return this.expanded ? 'expand_more' : 'chevron_right';
		},
	},
	methods: {
		onexpand(e) {
			this.expanded = true;
			sessionStorage.setItem("card.srvinf.expanded", this.expanded);
			this.expand();
		},
		oncollapse(e) {
			this.expanded = false;
			sessionStorage.setItem("card.srvinf.expanded", this.expanded);
			this.collapse();
		},
		expand() {
		},
		collapse() {
		},
	},
	created() {
		this.expanded = storageGetBoolean("card.srvinf.expanded", this.expanded);
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
			el.addEventListener('show.bs.collapse', this.onexpand);
			el.addEventListener('hide.bs.collapse', this.oncollapse);
		}

		(async () => {
			try {
				const response = await fetch("/api/stat/srvinf");
				if (response.ok) {
					this.srvinf = await response.json();
					this.srvinf.clientbuildvers = buildvers;
					this.srvinf.clientbuilddate = builddate;
				}
			} catch (e) { console.error(e); }
		})();
	},
	unmounted() {
		const el = document.getElementById('card' + this.iid);
		if (el) {
			el.removeEventListener('shown.bs.collapse', this.onexpand);
			el.removeEventListener('hidden.bs.collapse', this.oncollapse);
		}
	},
};

const VueMemgcCard = {
	template: '#memgc-card-tpl',
	data() {
		return {
			memgc: {},
			expanded: false,
			upmode: 5000,
			upid: 0,
			iid: makestrid(10), // instance ID
		};
	},
	computed: {
		clsupdate() {
			return { active: !!this.upmode };
		},

		expandchevron() {
			return this.expanded ? 'expand_more' : 'chevron_right';
		},
	},
	methods: {
		fmtduration(dur) {
			const sec = 1000;
			const min = 60 * sec;
			const hour = 60 * min;
			const day = 24 * hour;

			let fd;
			if (dur > day) {
				fd = "%d days %02d hours %02d min".printf(Math.floor(dur / day), Math.floor(dur % day / hour), Math.floor(dur % hour / min));
			} else if (dur > hour) {
				fd = "%d hours %02d min %02d sec".printf(Math.floor(dur / hour), Math.floor(dur % hour / min), Math.floor(dur % min / sec));
			} else {
				fd = "%02d min %02d sec".printf(Math.floor(dur % hour / min), Math.floor(dur % min / sec));
			}
			return fd;
		},

		onupdate() {
			clearInterval(this.upid);
			this.upid = 0;
			this.upmode = this.upmode ? 0 : 5000;
			if (this.expanded && this.upmode) {
				this.onrefresh();
				this.update();
			}
		},
		onrefresh() {
			(async () => {
				try {
					const response = await fetch("/api/stat/memusg");
					if (response.ok) {
						this.memgc = await response.json();
					}
				} catch (e) { console.error(e); }
			})();
		},
		update() {
			this.upid = setInterval(() => this.onrefresh(), this.upmode);
		},

		onexpand(e) {
			this.expanded = true;
			sessionStorage.setItem("card.memgc.expanded", this.expanded);
			this.expand();
		},
		oncollapse(e) {
			this.expanded = false;
			sessionStorage.setItem("card.memgc.expanded", this.expanded);
			this.collapse();
		},
		expand() {
			this.onrefresh();
			if (this.upmode) {
				this.update();
			}
		},
		collapse() {
			clearInterval(this.upid);
			this.upid = 0;
		},
	},
	created() {
		this.expanded = storageGetBoolean("card.memgc.expanded", this.expanded);
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
			el.addEventListener('show.bs.collapse', this.onexpand);
			el.addEventListener('hide.bs.collapse', this.oncollapse);
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

const VueCchinfCard = {
	template: '#cchinf-card-tpl',
	data() {
		return {
			cchinf: {},
			upmode: 5000,
			upid: 0,
			expanded: false,
			iid: makestrid(10), // instance ID
		};
	},
	computed: {
		clsupdate() {
			return { active: !!this.upmode };
		},

		avrtmbcchsize() {
			return this.cchinf.mtmbcount
				? (this.cchinf.mtmbsumsize1 / this.cchinf.mtmbcount).toFixed()
				: "N/A";
		},
		avrtmbwebpsize() {
			return this.cchinf.webpnum
				? (this.cchinf.webpsumsize1 / this.cchinf.webpnum).toFixed()
				: "N/A";
		},
		avrtmbjpgsize() {
			return this.cchinf.jpgnum
				? (this.cchinf.jpgsumsize1 / this.cchinf.jpgnum).toFixed()
				: "N/A";
		},
		avrtmbpngsize() {
			return this.cchinf.pngnum
				? (this.cchinf.pngsumsize1 / this.cchinf.pngnum).toFixed()
				: "N/A";
		},
		avrtmbgifsize() {
			return this.cchinf.gifnum
				? (this.cchinf.gifsumsize1 / this.cchinf.gifnum).toFixed()
				: "N/A";
		},

		expandchevron() {
			return this.expanded ? 'expand_more' : 'chevron_right';
		},
	},
	methods: {
		onupdate() {
			clearInterval(this.upid);
			this.upid = 0;
			this.upmode = this.upmode ? 0 : 5000;
			if (this.expanded && this.upmode) {
				this.onrefresh();
				this.update();
			}
		},
		onrefresh() {
			(async () => {
				try {
					const response = await fetch("/api/stat/cchinf");
					if (response.ok) {
						this.cchinf = await response.json();
					}
				} catch (e) { console.error(e); }
			})();
		},
		update() {
			this.upid = setInterval(() => this.onrefresh(), this.upmode);
		},

		onexpand(e) {
			this.expanded = true;
			sessionStorage.setItem("card.cchinf.expanded", this.expanded);
			this.expand();
		},
		oncollapse(e) {
			this.expanded = false;
			sessionStorage.setItem("card.cchinf.expanded", this.expanded);
			this.collapse();
		},
		expand() {
			this.onrefresh();
			if (this.upmode) {
				this.update();
			}
		},
		collapse() {
			clearInterval(this.upid);
			this.upid = 0;
		},
	},
	created() {
		this.expanded = storageGetBoolean("card.cchinf.expanded", this.expanded);
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
			el.addEventListener('show.bs.collapse', this.onexpand);
			el.addEventListener('hide.bs.collapse', this.oncollapse);
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

const VueConsoleCard = {
	template: '#console-card-tpl',
	data() {
		return {
			log: [],
			last: Date.now(),
			timemode: 1,
			fitwdh: false,
			upmode: 2500,
			upid: 0,
			expanded: false,
			iid: makestrid(10), // instance ID
		};
	},
	computed: {
		clsupdate() {
			return { active: !!this.upmode };
		},

		clsconsole() {
			return this.fitwdh ? 'w-100' : 'w-console';
		},
		clsfitwdh() {
			return { active: this.fitwdh };
		},
		clsvoid() {
			return { active: this.timemode === 0 };
		},
		clstime() {
			return { active: this.timemode === 1 };
		},
		clsdate() {
			return { active: this.timemode === 2 };
		},
		hintprefix() {
			switch (this.timemode) {
				case 0: return "time format: no prefix";
				case 1: return "time format: time only";
				case 2: return "time format: date-time";
			}
		},

		expandchevron() {
			return this.expanded ? 'expand_more' : 'chevron_right';
		},
	},
	methods: {
		llc(item) {
			return LLC[item.level] ?? "";
		},
		logline(item) {
			let prefix = "";
			const t = new Date(item.time);
			switch (this.timemode) {
				case 1:
					prefix = t.toLocaleTimeString('en-GB') + ' ';
					break;
				case 2:
					prefix = t.toLocaleString('en-GB') + ' ';
					break;
			}
			if (item.file) {
				prefix += item.file + ':' + item.line + ': ';
			}
			return prefix + item.msg.replace(sqllogregex, `<span class="sql">$2</span>`);
		},

		onfitwdh() {
			this.fitwdh = !this.fitwdh;
			sessionStorage.setItem("card.console.fitwdh", this.fitwdh);
		},
		onvoid() {
			this.timemode = 0;
			sessionStorage.setItem("card.console.timemode", this.timemode);
		},
		ontime() {
			this.timemode = 1;
			sessionStorage.setItem("card.console.timemode", this.timemode);
		},
		ondate() {
			this.timemode = 2;
			sessionStorage.setItem("card.console.timemode", this.timemode);
		},

		onupdate() {
			clearInterval(this.upid);
			this.upid = 0;
			this.upmode = this.upmode ? 0 : 2500;
			if (this.expanded && this.upmode) {
				this.onrefresh();
				this.update();
			}
		},
		onrefresh() {
			(async () => {
				try {
					const response = await fetch("/api/stat/getlog");
					if (response.ok) {
						const data = await response.json();
						this.log = data.list;
						this.last = Date.now();
					}
				} catch (e) { console.error(e); }
			})();
		},
		update() {
			this.upid = setInterval(() => (async () => {
				try {
					const response = await fetch(`/api/stat/getlog?unixms=${this.last}`);
					if (response.ok) {
						const data = await response.json();
						this.log.push(...data.list);
						this.last = Date.now();
					}
				} catch (e) { console.error(e); }
			})(), this.upmode);
		},

		onexpand(e) {
			this.expanded = true;
			sessionStorage.setItem("card.console.expanded", this.expanded);
			this.expand();
		},
		oncollapse(e) {
			this.expanded = false;
			sessionStorage.setItem("card.console.expanded", this.expanded);
			this.collapse();
		},
		expand() {
			this.onrefresh();
			if (this.upmode) {
				this.update();
			}
		},
		collapse() {
			clearInterval(this.upid);
			this.upid = 0;
		},
	},
	created() {
		this.timemode = storageGetBoolean("card.console.timemode", this.timemode);
		this.fitwdh = storageGetBoolean("card.console.fitwdh", this.fitwdh);
		this.expanded = storageGetBoolean("card.console.expanded", this.expanded);
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
			el.addEventListener('show.bs.collapse', this.onexpand);
			el.addEventListener('hide.bs.collapse', this.oncollapse);
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

const VueUsercCard = {
	template: '#users-card-tpl',
	data() {
		return {
			usrlst: {},
			usrlstpage: 0,
			usrlstsize: 20,
			upmode: 5000,
			upid: 0,
			expanded: false,
			iid: makestrid(10), // instance ID
		};
	},
	computed: {
		usrlstnum() {
			return Math.ceil(this.usrlst.total / this.usrlstsize);
		},

		clsupdate() {
			return { active: !!this.upmode };
		},

		expandchevron() {
			return this.expanded ? 'expand_more' : 'chevron_right';
		},
	},
	methods: {
		onusrlstpage(page) {
			this.usrlstpage = page;
		},

		onupdate() {
			clearInterval(this.upid);
			this.upid = 0;
			this.upmode = this.upmode ? 0 : 5000;
			if (this.expanded && this.upmode) {
				this.onrefresh();
				this.update();
			}
		},
		onrefresh() {
			(async () => {
				try {
					const response = await fetchjson("POST", "/api/stat/usrlst", {
						pos: this.usrlstpage * this.usrlstsize, num: this.usrlstsize
					});
					if (response.ok) {
						this.usrlst = await response.json();
					}
				} catch (e) { console.error(e); }
			})();
		},
		update() {
			this.upid = setInterval(() => this.onrefresh(), this.upmode);
		},

		onexpand(e) {
			this.expanded = true;
			sessionStorage.setItem("card.users.expanded", this.expanded);
			this.expand();
		},
		oncollapse(e) {
			this.expanded = false;
			sessionStorage.setItem("card.users.expanded", this.expanded);
			this.collapse();
		},
		expand() {
			this.onrefresh();
			if (this.upmode) {
				this.update();
			}
		},
		collapse() {
			clearInterval(this.upid);
			this.upid = 0;
		},
	},
	created() {
		this.expanded = storageGetBoolean("card.users.expanded", this.expanded);
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
			el.addEventListener('show.bs.collapse', this.onexpand);
			el.addEventListener('hide.bs.collapse', this.oncollapse);
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

// Create application view model
const appws = Vue.createApp(VueStatApp)
	.component('catitem-tag', VueCatItem)
	.component('user-tag', VueUser)
	.component('pagination-tag', VuePagination)
	.component('srvinf-card-tag', VueSrvinfCard)
	.component('memgc-card-tag', VueMemgcCard)
	.component('cchinf-card-tag', VueCchinfCard)
	.component('console-card-tag', VueConsoleCard)
	.component('users-card-tag', VueUsercCard)
	.mount('#app');

// The End.
