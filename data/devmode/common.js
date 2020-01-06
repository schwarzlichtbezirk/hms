"use strict";

//@ sourceMappingURL=main.min.map

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
if (!String.prototype.printf) {
	String.prototype.printf = function () {
		var arr = Array.prototype.slice.call(arguments);
		var i = -1;
		function callback(sym, plus, flag, size, exp, base, type) {
			if (sym === '%%') return '%';
			if (arr[++i] === undefined) return undefined;
			exp = exp ? parseInt(exp.substr(1)) : undefined;
			base = base ? parseInt(base.substr(1)) : undefined;
			var val;
			switch (type) {
				case 's': val = arr[i]; break;
				case 'c': val = arr[i][0]; break;
				case 'f': val = parseFloat(arr[i]).toFixed(exp); break;
				case 'p': val = parseFloat(arr[i]).toPrecision(exp); break;
				case 'e': val = parseFloat(arr[i]).toExponential(exp); break;
				case 'x': val = parseInt(arr[i]).toString(base ? base : 16); break;
				case 'd': val = parseFloat(parseInt(arr[i], base ? base : 10).toPrecision(exp)).toFixed(0); break;
			}
			val = typeof val === 'object' ? JSON.stringify(val) : val.toString(base);
			size = parseInt(size); /* padding size */
			if (plus && val[0] !== '-') {
				val = '+' + val;
			}
			switch (flag) {
				case '-': val += ' '.repeat(size - val.length); break;
				case ' ': val = ' '.repeat(size - val.length) + val; break;
				case '0': val = '0'.repeat(size - val.length) + val; break;
			}
			return val;
		}
		var regex = /%([+])?([- 0])?([1-9][0-9]*)?([.][0-9]+)?([#][0-9]+)?([scfpexd%])/g;
		return this.replace(regex, callback);
	};
}

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

class Viewer {
	setup() { }
	close() { }
	setfile(file) {
		this.file = file;
	}
}

// The End.
