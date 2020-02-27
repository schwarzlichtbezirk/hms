"use strict";

const G = {
};

const globalmixin = {
	data: function () {
		return G;
	},
	computed: {
	},
	methods: {
	}
};

Vue.component('file-icon-tag', {
	template: '#file-icon-tpl',
	props: ["file", "state"],
	computed: {
		fmttitle() {
			let title = this.file.name;
			if (this.file.pref) {
				title += '\nshare: ' + shareprefix + this.file.pref;
			}
			if (this.file.type !== FT.dir) {
				title += '\nsize: ' + fmtitemsize(this.file.size);
			}
			return title;
		},

		webpicon() {
			return this.file.ntmb === 1
				? ''
				: '/asst/file-webp/' + geticonname(this.file) + '.webp';
		},

		pngicon() {
			return this.file.ntmb === 1
				? '/thumb/' + this.file.ktmb
				: '/asst/file-png/' + geticonname(this.file) + '.png';
		},

		// manage items classes
		itemview() {
			return { selected: this.state.selected };
		}
	},
	methods: {
		onselect() {
			this.$emit('select', this.file);
		},

		onrun() {
			this.$emit('run', this.file);
		}
	}
});

// <file-page-tag v-bind:list="playlist">files</file-page-tag>
Vue.component('file-page-tag', {
	template: '#file-page-tpl',
	props: ["list"],
	data: function () {
		return {
			iid: makestrid(10) // instance ID
		};
	},
	computed: {
	},
	methods: {
	}
});

// The End.
