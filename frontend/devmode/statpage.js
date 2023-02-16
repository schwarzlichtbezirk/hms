"use strict";

// Scanning frequency.
const scanfreq = 2000;

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
	"YahooBot" // Bot list ends here
];

// OS Name
const OSN = [
	"Unknown",
	"WindowsPhone", "Windows", "MacOSX",
	"iOS", "Android", "Blackberry",
	"ChromeOS", "Kindle", "WebOS", "Linux",
	"Playstation", "Xbox", "Nintendo",
	"Bot"
];

// OS Platform
const OSP = [
	"Unknown",
	"Windows", "Mac", "Linux",
	"iPad", "iPhone", "iPod", "Blackberry", "WindowsPhone",
	"Playstation", "Xbox", "Nintendo",
	"Bot"
];

// Device Type
const DT = [
	"Unknown", "Computer", "Tablet", "Phone", "Console", "Wearable", "TV"
];

const VueStatApp = {
	template: '#app-tpl',
	data() {
		return {
			srvinf: {},
			memgc: {},
			cchinf: {},
			usrlst: {},
			usrlstpage: 0,
			usrlstsize: 20
		};
	},
	computed: {
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

		usrlstnum() {
			return Math.ceil(this.usrlst.total / this.usrlstsize);
		}
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

		onusrlstpage(page) {
			this.usrlstpage = page;
		}
	},
	mounted() {
		eventHub.on('ajax', viewpreloader);

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

		{
			const el = document.getElementById('collapse-memory');
			let expanded = false;
			el?.addEventListener('show.bs.collapse', e => {
				expanded = true;
				(async () => {
					try {
						while (expanded) {
							const response = await fetch("/api/stat/memusg");
							if (response.ok) {
								this.memgc = await response.json();
							}
							await new Promise(resolve => setTimeout(resolve, scanfreq));
						}
					} catch (e) { console.error(e); }
				})();
			});
			el?.addEventListener('hide.bs.collapse', e => {
				expanded = false;
			});
		}

		{
			const el = document.getElementById('collapse-cache');
			let expanded = false;
			el?.addEventListener('show.bs.collapse', e => {
				expanded = true;
				(async () => {
					try {
						while (expanded) {
							const response = await fetch("/api/stat/cchinf");
							if (response.ok) {
								this.cchinf = await response.json();
							}
							await new Promise(resolve => setTimeout(resolve, scanfreq));
						}
					} catch (e) { console.error(e); }
				})();
			});
			el?.addEventListener('hide.bs.collapse', e => {
				expanded = false;
			});
		}

		{
			const el = document.getElementById('collapse-users');
			let expanded = false;
			el?.addEventListener('show.bs.collapse', e => {
				expanded = true;
				(async () => {
					try {
						while (expanded) {
							const response = await fetchjson("POST", "/api/stat/usrlst", {
								pos: this.usrlstpage * this.usrlstsize, num: this.usrlstsize
							});
							if (response.ok) {
								this.usrlst = await response.json();
							}
							await new Promise(resolve => setTimeout(resolve, scanfreq));
						}
					} catch (e) { console.error(e); }
				})();
			});
			el?.addEventListener('hide.bs.collapse', e => {
				expanded = false;
			});
		}

		// hide start-up preloader
		eventHub.emit('ajax', -1);
	},
	unmounted() {
		eventHub.off('ajax', viewpreloader);
	}
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
	}
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
			return this.disleft && 'disabled';
		},
		clsright() {
			return this.disright && 'disabled';
		}
	},
	methods: {
		clsactive(page) {
			return page === this.sel && 'active';
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
		}
	}
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
		}
	},
	methods: {
	}
};

const VueConsoleCard = {
	template: '#console-card-tpl',
	data() {
		return {
			log: [],
			timemode: 1,
		};
	},
	computed: {
		isnoprefix() {
			return this.timemode === 0 && 'btn-info' || 'btn-outline-info';
		},
		istime() {
			return this.timemode === 1 && 'btn-info' || 'btn-outline-info';
		},
		isdatetime() {
			return this.timemode === 2 && 'btn-info' || 'btn-outline-info';
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

		onupdate() {
		},
		onrefresh() {
			(async () => {
				try {
					const response = await fetch("/api/stat/getlog");
					if (response.ok) {
						const data = await response.json();
						this.log = data.list;
					}
				} catch (e) { console.error(e); }
			})();
		},

		onnoprefix() {
			this.timemode = 0;
		},
		ontime() {
			this.timemode = 1;
		},
		ondatetime() {
			this.timemode = 2;
		},
	},
	mounted() {
		const el = document.getElementById('collapse-console');
		el?.addEventListener('show.bs.collapse', e => {
			this.onrefresh();
		});
	},
};

// Create application view model
const appws = Vue.createApp(VueStatApp)
	.component('catitem-tag', VueCatItem)
	.component('pagination-tag', VuePagination)
	.component('user-tag', VueUser)
	.component('console-card-tag', VueConsoleCard)
	.mount('#app');

// The End.
