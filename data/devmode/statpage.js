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
	template: `
<div>
	<h2>home media server monitor</h2>

	<div id="accordion">

		<div class="card">
			<div class="card-header">
				<a class="card-link" data-toggle="collapse" href="#collapse-servinfo">server info</a>
			</div>
			<div id="collapse-servinfo" class="collapse show">
				<div class="card-body stattable">
					<div class="row">
						<div class="col-sm-6 field-name">client version:</div>
						<div class="col-md-6 field-value">{{servinfo.buildvers}}</div>
					</div>
					<div class="row">
						<div class="col-sm-6 field-name">server start time:</div>
						<div class="col-md-6 field-value">{{(new Date(servinfo.started)).toLocaleString()}}</div>
					</div>
					<div class="row">
						<div class="col-sm-6 field-name">golang version:</div>
						<div class="col-md-6 field-value">{{servinfo.govers}}</div>
					</div>
					<div class="row">
						<div class="col-sm-6 field-name">host operation system:</div>
						<div class="col-md-6 field-value">{{servinfo.os}}</div>
					</div>
					<div class="row">
						<div class="col-sm-6 field-name">number of logical CPUs:</div>
						<div class="col-md-6 field-value">{{servinfo.numcpu}}</div>
					</div>
					<div class="row">
						<div class="col-sm-6 field-name">max number of used CPUs:</div>
						<div class="col-md-6 field-value">{{servinfo.maxprocs}}</div>
					</div>
					<div class="row">
						<div class="col-sm-6 field-name">program directory:</div>
						<div class="col-md-6 field-value">{{servinfo.destpath}}</div>
					</div>
					<div class="row">
						<div class="col-sm-6 field-name">data root directory:</div>
						<div class="col-md-6 field-value">{{servinfo.rootpath}}</div>
					</div>
				</div>
			</div>
		</div>

		<div class="card">
			<div class="card-header">
				<a class="card-link collapsed" data-toggle="collapse" href="#collapse-memory">memory usage &amp; garbage collector</a>
			</div>
			<div id="collapse-memory" class="collapse">
				<div class="card-body stattable">
					<div class="row">
						<div class="col-sm-6 field-name">server running time:</div>
						<div class="col-md-6 field-value">{{fmtduration(memgc.running)}}</div>
					</div>
					<div class="row">
						<div class="col-sm-6 field-name">bytes allocated and not yet freed:</div>
						<div class="col-md-6 field-value">{{memgc.heapalloc}}</div>
					</div>
					<div class="row">
						<div class="col-sm-6 field-name">bytes obtained from system:</div>
						<div class="col-md-6 field-value">{{memgc.heapsys}}</div>
					</div>
					<div class="row">
						<div class="col-sm-6 field-name">bytes allocated (even if freed):</div>
						<div class="col-md-6 field-value">{{memgc.totalalloc}}</div>
					</div>
					<div class="row">
						<div class="col-sm-6 field-name">heap size for next GC cycle:</div>
						<div class="col-md-6 field-value">{{memgc.nextgc}}</div>
					</div>
					<div class="row">
						<div class="col-sm-6 field-name">number of GC cycles:</div>
						<div class="col-md-6 field-value">{{memgc.numgc}}</div>
					</div>
					<div class="row">
						<div class="col-sm-6 field-name">time used by GC (nanoseconds):</div>
						<div class="col-md-6 field-value">{{memgc.pausetotalns}}</div>
					</div>
					<div class="row">
						<div class="col-sm-6 field-name">fraction of CPU time used by GC:</div>
						<div class="col-md-6 field-value">{{memgc.gccpufraction}}</div>
					</div>
				</div>
			</div>
		</div>

		<div class="card">
			<div class="card-header">
				<a class="card-link collapsed" data-toggle="collapse" href="#collapse-console">Console output</a>
			</div>
			<div id="collapse-console" class="collapse">
				<div class="card-body">
					<button v-on:click="ongetlog" type="button" class="btn btn-primary console-but">Update</button>
					<div class="btn-group">
						<button v-on:click="onnoprefix" type="button" v-bind:class="isnoprefix" class="btn console-but">No prefix</button>
						<button v-on:click="ontime" type="button" v-bind:class="istime" class="btn console-but">Time</button>
						<button v-on:click="ondatetime" type="button" v-bind:class="isdatetime" class="btn console-but">Date-time</button>
					</div>
					<pre ref="console" class="console-window">{{consolecontent}}</pre>
				</div>
			</div>
		</div>

	</div>
</div>
`,
	el: '#app', // manage whole visible html page included slideout, header, footer
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
			ajaxjson("GET", "/api/getlog", (xhr) => {
				if (xhr.status === 200) { // OK
					this.log = xhr.response;
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
