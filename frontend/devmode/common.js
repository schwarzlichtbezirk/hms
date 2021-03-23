"use strict";

if (typeof $.fn.popover === 'undefined') {
	throw new Error("Bootstrap library required");
}
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

const pathext = fname => fname.substr(fname.lastIndexOf('.')).toLowerCase();

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
	if (tmax < 60) {
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

const makestrid = length => {
	const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789';
	const len = chars.length;
	let result = '';
	for (let i = 0; i < length; i++) {
		result += chars.charAt(Math.floor(Math.random() * len));
	}
	return result;
};

////////////////////////
// Event handle model //
////////////////////////

const makeeventmodel = () => {
	const listeners = [];

	const t = {
		// dispatch event to listeners
		emit: (name, ...args) => {
			let i = 0;
			while (i < listeners.length) {
				const [ln, lf, lo] = listeners[i];
				if (ln === name) {
					if (lo) { // check "once" before call to prevent loop
						listeners.splice(i, 1);
						i--;
					}
					try {
						lf(...args);
					} catch (e) {
						console.error(e);
					}
				}
				i++;
			}
		},

		// insert new events listener
		on: (name, f, once = false) => listeners.push([name, f, once]),

		// insert new events listener for one call
		once: (name, f) => listeners.push([name, f, true]),

		// remove registered events listener
		off: (name, f) => {
			let i = 0;
			while (i < listeners.length) {
				const [ln, lf] = listeners[i];
				if ((ln === name || !name) && (lf === f || !f)) {
					listeners.splice(i, 1);
				} else {
					i++;
				}
			}
		},

		// insert map of new events listeners,
		// each entry must have valid name and associated closure
		onmap: evmap => {
			for (const name in evmap) {
				listeners.push([name, evmap[name], false]);
			}
		},

		// remove map of registered events listeners,
		// each entry must have valid name and associated closure
		offmap: evmap => {
			for (const name in evmap) {
				const f = evmap[name];
				for (const i in listeners.length) {
					const [ln, lf] = listeners[i];
					if (ln === name && lf === f) {
						listeners.splice(i, 1);
						break;
					}
				}
			}
		},

		listens: (name, f) => {
			let i = 0;
			for (const [ln, lf] of listeners) {
				if ((ln === name || !name) && (lf === f || !f)) {
					i++;
				}
			}
			return i;
		},

		listenlen: () => listeners.length
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
