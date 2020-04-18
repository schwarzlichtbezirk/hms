"use strict";

// Scanning frequency
const scanfreq = 2000;

const app = new Vue({
	el: '#app',
	template: '#app-tpl',
	data: {
		servinfo: {},
		memgc: {},
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
			fetchajax("GET", "/api/getlog").then(response => {
				if (response.ok) {
					this.log = response.data;
				}
			});
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
		fetchajax("GET", "/api/srvinf").then(response => {
			if (response.ok) {
				this.servinfo = response.data;
				this.servinfo.buildvers = buildvers;
			}
		});

		$("#collapse-memory").on('show.bs.collapse', () => {
			const scanner = () => {
				fetchajax("GET", "/api/memusg").then(response => {
					if (response.ok) {
						this.memgc = response.data;
					}
				});
			};

			scanner();
			const id = setInterval(scanner, scanfreq);
			$("#collapse-memory").on('hide.bs.collapse', () => {
				clearInterval(id);
			});
		});

		$("#collapse-console").on('show.bs.collapse', () => {
			this.ongetlog();
		});
	}
}); // end of vue application

$(document).ready(() => {
	$('.preloader').hide("fast");
	$('#app').show("fast");
});

// The End.
