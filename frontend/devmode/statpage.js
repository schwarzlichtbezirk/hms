"use strict";

// Scanning frequency.
const scanfreq = 2000;

// Maximum number of buttons at pagination component.
const maxpageitem = 5;

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
			log: [],
			timemode: 1,
			usrlst: {},
			usrlstpage: 0,
			usrlstsize: 20
		};
	},
	computed: {
		consolecontent() {
			const text = [];
			for (const i of this.log) {
				let prefix = "";
				const t = new Date(i.time);
				switch (this.timemode) {
					case 1:
						prefix = t.toLocaleTimeString() + ' ';
						break;
					case 2:
						prefix = t.toLocaleString() + ' ';
						break;
				}
				if (i.file) {
					prefix += i.file + ':' + i.line + ': ';
				}
				text.unshift(prefix + i.msg.trimRight());
			}
			return text.join('\n');
		},

		isnoprefix() {
			return this.timemode === 0 && 'btn-info' || 'btn-outline-info';
		},

		istime() {
			return this.timemode === 1 && 'btn-info' || 'btn-outline-info';
		},

		isdatetime() {
			return this.timemode === 2 && 'btn-info' || 'btn-outline-info';
		},

		avrshow() {
			const zn = (this.cchinf.tmbjpgnum ? 1 : 0) + (this.cchinf.tmbpngnum ? 1 : 0) + (this.cchinf.tmbgifnum ? 1 : 0);
			return zn > 1;
		},
		avrtmbcchsize() {
			if (this.cchinf.tmbcchnum) {
				return (this.cchinf.tmbcchsize1 / this.cchinf.tmbcchnum).toFixed();
			} else {
				return "N/A";
			}
		},
		avrtmbjpgsize() {
			if (this.cchinf.tmbjpgnum) {
				return (this.cchinf.tmbjpgsize1 / this.cchinf.tmbjpgnum).toFixed();
			} else {
				return "N/A";
			}
		},
		avrtmbpngsize() {
			if (this.cchinf.tmbpngnum) {
				return (this.cchinf.tmbpngsize1 / this.cchinf.tmbpngnum).toFixed();
			} else {
				return "N/A";
			}
		},
		avrtmbgifsize() {
			if (this.cchinf.tmbgifnum) {
				return (this.cchinf.tmbgifsize1 / this.cchinf.tmbgifnum).toFixed();
			} else {
				return "N/A";
			}
		},
		avrmedcchsize() {
			if (this.cchinf.medcchnum) {
				return (this.cchinf.medcchsize1 / this.cchinf.medcchnum).toFixed();
			} else {
				return "N/A";
			}
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

		ongetlog() {
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

		onusrlstpage(page) {
			this.usrlstpage = page;
		}
	},
	created() {
		eventHub.on('ajax', viewpreloader);
	},
	mounted() {
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
			const el = document.getElementById('collapse-console');
			el?.addEventListener('show.bs.collapse', e => {
				this.ongetlog();
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
			if (this.user.authid) {
				return this.user.isauth ? 'person' : 'person_outline';
			} else {
				return this.user.prfid ? 'radio_button_checked' : 'radio_button_unchecked';
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

// Create application view model
const appws = Vue.createApp(VueStatApp)
	.component('catitem-tag', VueCatItem)
	.component('pagination-tag', VuePagination)
	.component('user-tag', VueUser)
	.mount('#app');

// The End.
