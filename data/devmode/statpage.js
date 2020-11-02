"use strict";

// Scanning frequency
const scanfreq = 2000;

const app = new Vue({
	el: '#app',
	template: '#app-tpl',
	data: {
		srvinf: {},
		memgc: {},
		cchinf: {},
		log: [],
		timemode: 1
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

			$("#collapse-memory").one('hide.bs.collapse', () => {
				expanded = false;
			});
		});

		$("#collapse-console").on('show.bs.collapse', () => {
			this.ongetlog();
		});
	},
	beforeDestroy() {
		$("#collapse-memory").off('show.bs.collapse');
		$("#collapse-memory").off('hide.bs.collapse');
		$("#collapse-cache").off('show.bs.collapse');
		$("#collapse-cache").off('hide.bs.collapse');
		$("#collapse-console").off('show.bs.collapse');
	}
}); // end of vue application

$(document).ready(() => {
	$('.preloader-lock').hide("fast");
	$('#app').show("fast");
});

// The End.
