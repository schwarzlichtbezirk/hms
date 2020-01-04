"use strict";

const G = {
	selected: null, // selected file properties
	playbackmode: false
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
	template: `
<div class="file-item" v-bind:class="itemview" v-on:click.stop="onselect" v-on:dblclick.stop="onrun" v-bind:title="fmttitle">
	<div class="img-overlay">
		<picture>
			<source v-bind:srcset="webpicon" type="image/webp">
			<source v-bind:srcset="pngicon" type="image/png">
			<img src="/asst/file-png/doc-file.png" v-bind:alt="file.name">
		</picture>
		<picture v-if="file.pref">
			<source srcset="/asst/file-webp/shared.webp" type="image/webp">
			<source srcset="/asst/file-png/shared.png" type="image/png">
			<img src="/asst/file-png/shared.png" alt="shared">
		</picture>
		<img v-if="isplay" src="/asst/equalizer-bars.gif" alt="shared">
	</div>
	<p>{{file.name}}</p>
</div>
`,
	mixins: [globalmixin],
	props: ["file"],
	data: function () {
		return {
		};
	},
	computed: {
		fmttitle() {
			let title = this.file.name;
			if (this.file.pref) {
				title += '\nshare: ' + shareprefix + this.file.pref;
			}
			if (this.file.type !== Dir) {
				title += '\nsize: ' + fmtitemsize(this.file.size);
			}
			return title;
		},

		webpicon() {
			return '/asst/file-webp/' + geticonname(this.file) + '.webp';
		},

		pngicon() {
			return '/asst/file-png/' + geticonname(this.file) + '.png';
		},

		// manage items classes
		itemview() {
			return { selected: this.selected === this.file};
		},

		isplay() {
			return this.selected === this.file && this.playbackmode;
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

// The End.
