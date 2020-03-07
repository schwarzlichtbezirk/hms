"use strict";

const filehint = file => {
	const lst = [];
	lst.push(file.name);
	if (file.pref) {
		lst.push('share: ' + shareprefix + file.pref);
	}
	if (file.type !== FT.dir) {
		lst.push('size: ' + fmtitemsize(file.size));
	}
	if (file.title) {
		lst.push('title: ' + file.title);
	}
	if (file.album) {
		lst.push('album: ' + file.album);
	}
	if (file.artist) {
		lst.push('artist: ' + file.artist);
	}
	if (file.composer) {
		lst.push('composer: ' + file.composer);
	}
	if (file.genre) {
		lst.push('genre: ' + file.genre);
	}
	if (file.year) {
		lst.push('year: ' + file.year);
	}
	if (file.track && (file.track.number || file.track.total)) {
		lst.push(`track: ${file.track.number || ''}/${file.track.total || ''}`);
	}
	if (file.disc && (file.disc.number || file.disc.total)) {
		lst.push(`disc: ${file.disc.number || ''}/${file.disc.total || ''}`);
	}
	if (file.comment) {
		lst.push('comment: ' + file.comment.substring(0, 80));
	}
	return lst.join('\n');
};

Vue.component('file-icon-tag', {
	template: '#file-icon-tpl',
	props: ["file", "state"],
	computed: {
		fmttitle() {
			return filehint(this.file);
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

Vue.component('img-icon-tag', {
	template: '#img-icon-tpl',
	props: ["file", "state"],
	computed: {
		fmttitle() {
			return filehint(this.file);
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
