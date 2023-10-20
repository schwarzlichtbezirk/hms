"use strict";

if (!String.prototype.format) {
	String.prototype.format = function () {
		var args = arguments;
		return this.replace(/{(\d+)}/g, function (match, number) {
			return typeof args[number] !== 'undefined'
				? args[number]
				: match;
		});
	};
}

// Printf for javasript
if (!String.prototype.printf) {
	String.prototype.printf = function () {
		var arr = Array.prototype.slice.call(arguments);
		var i = -1;
		function callback(exp, p0, p1, p2, p3, p4) {
			if (exp === '%%') return '%';
			if (arr[++i] === undefined) return undefined;
			exp = p2 ? parseInt(p2.substring(1)) : undefined;
			var base = p3 ? parseInt(p3.substring(1)) : undefined;
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
			val = typeof val === 'object' ? JSON.stringify(val) : val.toString(base);
			var sz = parseInt(p1); /* padding size */
			var ch = p1 && p1[0] === '0' ? '0' : ' '; /* isnull? */
			while (val.length < sz) val = p0 !== undefined ? val + ch : ch + val; /* isminus? */
			return val;
		}
		var regex = /%(-)?(0?[0-9]+)?([.][0-9]+)?([#][0-9]+)?([scfpexd%])/g;
		return this.replace(regex, callback);
	};
}

const pathjoin = (...args) => {
	return args.map((part, i) => {
		if (i === 0) {
			return part.trim().replace(/[\/]*$/g, '');
		} else {
			return part.trim().replace(/(^[\/]*|[\/]*$)/g, '');
		}
	}).filter(x => x.length).join('/');
};

const pathext = fname => fname.substring(fname.lastIndexOf('.')).toLowerCase();

const fmtfilesize = (size) => {
	if (size < 1536) {
		return size + " bytes";
	} else if (size < 1048576) { // 1M
		return (size / 1024).toPrecision(3) + " kB";
	} else if (size < 1073741824) { // 1G
		return (size / 1048576).toPrecision(3) + " MB";
	} else if (size < 1099511627776) { // 1T
		return (size / 1073741824).toPrecision(3) + " GB";
	} else {
		return (size / 1099511627776).toPrecision(3) + " TB";
	}
};

const fmtitemsize = (size) => {
	if (size < 1536) {
		return fmtfilesize(size);
	} else {
		return "%s (%d bytes)".printf(fmtfilesize(size), size);
	}
};

const fmttime = (tval, tmax) => {
	const lead0 = (v, n) => {
		const vs = Math.floor(v).toString();
		const r = n - vs.length;
		return r > 0 ? "0".repeat(r) + vs : vs;
	};
	if (!Number.isFinite(tval)) {
		return "unknown";
	} else if (tmax < 60) {
		return lead0(tval, 2);
	} else if (tmax < 3600) {
		const ss = tval % 60;
		const mm = tval / 60;
		return lead0(mm, 2) + ':' + lead0(ss, 2);
	} else {
		const ss = tval % 60;
		const mm = tval % 3600 / 60;
		const hh = tval / 3600;
		return lead0(hh, 2) + ':' + lead0(mm, 2) + ':' + lead0(ss, 2);
	}
};

const dur_µs = 1000;
const dur_ms = 1000 * dur_µs;
const dur_sec = 1000 * dur_ms;
const dur_min = 60 * dur_sec;
const dur_hour = 60 * dur_min;
const dur_day = 24 * dur_hour;
const fmtduration = (d, prec) => {
	const t = [];
	const day = Math.floor(d / dur_day);
	if (day) {
		t.push(day + "d");
		d -= day * dur_day;
	}
	if (d >= prec) {
		const hour = Math.floor(d / dur_hour);
		if (hour) {
			t.push(hour + "h");
			d -= hour * dur_hour;
		}
		if (d >= prec) {
			const min = Math.floor(d / dur_min);
			if (min) {
				t.push(min + "m");
				d -= min * dur_min;
			}
			if (d >= prec) {
				const sec = Math.floor(d / dur_sec);
				if (sec) {
					t.push(sec + "s");
					d -= sec * dur_sec;
				}
				if (d >= prec) {
					const ms = Math.floor(d / dur_ms);
					if (ms) {
						t.push(ms + "ms");
						d -= ms * dur_ms;
					}
					if (d >= prec) {
						const µs = Math.floor(d / dur_µs);
						if (µs) {
							t.push(µs + "µs");
							d -= µs * dur_µs;
						}
					}
				}
			}
		}
	}
	return t.join("");
};

const makestrid = length => {
	const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789';
	const len = chars.length;
	let result = '';
	for (let i = 0; i < length; i++) {
		result += chars.charAt(Math.floor(Math.random() * len));
	}
	return result;
};

const storageGetItem = (id, def) => {
	return JSON.parse(sessionStorage.getItem(id)) ?? def;
};

const storageSetItem = (id, val) => {
	sessionStorage.setItem(id, JSON.stringify(val));
};

///////////////////////
// Events handle hub //
///////////////////////

const makeeventhub = () => {
	const listeners = [];

	const t = {
		// dispatch event to listeners
		emit: (name, ...args) => {
			let i = listeners.length;
			while (i > 0) {
				const [ln, lf, lo] = listeners[--i];
				if (ln === name || !name) {
					if (lo) { // check "once" before call to prevent loop
						listeners.splice(i, 1);
					}
					lf(...args);
				}
			}
		},

		// insert new events listener
		on: (name, f, once = false) => listeners.push([name, f, once]),

		// insert new events listener for one call
		once: (name, f) => listeners.push([name, f, true]),

		// remove registered events listeners
		off: (name, f) => {
			let i = listeners.length;
			while (i > 0) {
				const [ln, lf] = listeners[--i];
				if ((ln === name || !name) && (lf === f || !f)) {
					listeners.splice(i, 1);
				}
			}
		},

		// insert map of new events listeners,
		// each entry must have valid name and associated closure
		onmap: evmap => {
			for (const [n, f] of Object.entries(evmap)) {
				listeners.push([n, f, false]);
			}
		},

		// remove map of registered events listeners,
		// each entry must have valid name and associated closure
		offmap: evmap => {
			for (const [n, f] of Object.entries(evmap)) {
				let i = listeners.length;
				while (i > 0) {
					const [ln, lf] = listeners[--i];
					if (ln === n && lf === f) {
						listeners.splice(i, 1);
						break;
					}
				}
			}
		},

		listens: (name, f) => {
			let c = 0;
			for (const e of listeners) {
				const [ln, lf] = e;
				if ((ln === name || !name) && (lf === f || !f)) {
					c++;
				}
			}
			return c;
		},

		len: () => listeners.length
	};

	return t;
};

const extend = (dest, src) => {
	for (const i in src) {
		dest[i] = src[i];
	}
	return dest;
};

// The End.
