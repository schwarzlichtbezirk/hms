"use strict";

// Scanning frequency
const scanfreq = 2000;

// Printf for javasript
if (!String.prototype.printf) {
	String.prototype.printf = function () {
		var arr = Array.prototype.slice.call(arguments);
		var i = -1;
		function callback(exp, p0, p1, p2, p3, p4) {
			if (exp === '%%') return '%';
			if (arr[++i] === undefined) return undefined;
			exp = p2 ? parseInt(p2.substr(1)) : undefined;
			var base = p3 ? parseInt(p3.substr(1)) : undefined;
			var val;
			switch (p4) {
				case 's': val = arr[i]; break;
				case 'c': val = arr[i][0]; break;
				case 'f': val = parseFloat(arr[i]).toFixed(exp); break;
				case 'p': val = parseFloat(arr[i]).toPrecision(exp); break;
				case 'e': val = parseFloat(arr[i]).toExponential(exp); break;
				case 'x': val = parseInt(arr[i]).toString(base ? base : 16); break;
				case 'd': val = parseFloat(parseInt(arr[i], base ? base : 10).toPrecision(exp)).toFixed(0); break;
			}
			val = typeof (val) === 'object' ? JSON.stringify(val) : val.toString(base);
			var sz = parseInt(p1); /* padding size */
			var ch = p1 && p1[0] === '0' ? '0' : ' '; /* isnull? */
			while (val.length < sz) val = p0 !== undefined ? val + ch : ch + val; /* isminus? */
			return val;
		}
		var regex = /%(-)?(0?[0-9]+)?([.][0-9]+)?([#][0-9]+)?([scfpexd%])/g;
		return this.replace(regex, callback);
	};
}

const app = new Vue({
	el: '#app', // manage whole visible html page included slideout, header, footer
	data: {
		servinfo: {},
		memgc: {},
		log: [],
		timemode: 1
	},
	computed: {
		consolecontent: function () {
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

		isnoprefix: function () {
			return this.timemode === 0 && 'btn-info' || 'btn-outline-info';
		},

		istime: function () {
			return this.timemode === 1 && 'btn-info' || 'btn-outline-info';
		},

		isdatetime: function () {
			return this.timemode === 2 && 'btn-info' || 'btn-outline-info';
		}
	},
	methods: {
		fmtduration: function (dur) {
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

		ongetlog: function () {
			ajaxjson("GET", "/api/getlog", (xhr) => {
				if (xhr.status === 200) { // OK
					this.log = xhr.response;
				}
			});
		},

		onnoprefix: function () {
			this.timemode = 0;
		},

		ontime: function () {
			this.timemode = 1;
		},

		ondatetime: function () {
			this.timemode = 2;
		}
	}
}); // end of vue application

$(document).ready(() => {
	$('.preloader').hide("fast");
	$('#app').show("fast");

	ajaxjson("GET", "/api/servinfo", (xhr) => {
		if (xhr.status === 200) { // OK
			app.servinfo = xhr.response;
			app.servinfo.buildvers = buildvers;
		}
	});

	$("#collapse-memory").on('show.bs.collapse', () => {
		const scanner = () => {
			ajaxjson("GET", "/api/memusage", (xhr) => {
				if (xhr.status === 200) { // OK
					app.memgc = xhr.response;
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
		app.ongetlog();
	});
});

// The End.
