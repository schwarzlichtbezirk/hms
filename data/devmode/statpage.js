"use strict";

// Scanning frequency
const scanfreq = 2000;

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

Vue.component('user-tag', {
	template: '#user-tpl',
	props: ["user"],
	data: function () {
		return {
		};
	},
	computed: {
		clsonline() {
			return this.user.online ? 'text-success' : 'text-secondary';
		},
		txtonline() {
			return this.user.online ? 'radio_button_checked' : 'radio_button_unchecked';
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
});

const app = new Vue({
	el: '#app',
	template: '#app-tpl',
	data: {
		srvinf: {},
		memgc: {},
		cchinf: {},
		log: [],
		timemode: 1,
		usrlst: {}
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
						this.log = await response.json();
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
		}
	},
	mounted() {
		(async () => {
			try {
				const response = await fetch("/api/stat/srvinf");
				if (response.ok) {
					this.srvinf = await response.json();
					this.srvinf.buildvers = buildvers;
					this.srvinf.builddate = builddate;
				}
			} catch (e) { console.error(e); }
		})();

		$("#collapse-memory").on('show.bs.collapse', () => {
			let expanded = true;
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

			$("#collapse-memory").one('hide.bs.collapse', () => {
				expanded = false;
			});
		});

		$("#collapse-cache").on('show.bs.collapse', () => {
			let expanded = true;
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

			$("#collapse-cache").one('hide.bs.collapse', () => {
				expanded = false;
			});
		});

		$("#collapse-console").on('show.bs.collapse', () => {
			this.ongetlog();
		});

		$("#collapse-users").on('show.bs.collapse', () => {
			let expanded = true;
			(async () => {
				try {
					while (expanded) {
						const response = await fetchjson("POST", "/api/stat/usrlst", {
							pos: 0, num: 20
						});
						if (response.ok) {
							this.usrlst = await response.json();
						}
						await new Promise(resolve => setTimeout(resolve, scanfreq));
					}
				} catch (e) { console.error(e); }
			})();

			$("#collapse-users").one('hide.bs.collapse', () => {
				expanded = false;
			});
		});
	},
	beforeDestroy() {
		$("#collapse-memory").off('show.bs.collapse');
		$("#collapse-memory").off('hide.bs.collapse');
		$("#collapse-cache").off('show.bs.collapse');
		$("#collapse-cache").off('hide.bs.collapse');
		$("#collapse-console").off('show.bs.collapse');
		$("#collapse-users").off('show.bs.collapse');
		$("#collapse-users").off('hide.bs.collapse');
	}
}); // end of vue application

$(document).ready(() => {
	$('.preloader-lock').hide("fast");
	$('#app').show("fast");
});

// The End.
